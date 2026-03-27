package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Order struct {
	ID         int       `json:"id"`
	CustomerID string    `json:"customer_id" binding:"required"`
	Amount     float64   `json:"amount" binding:"required"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type App struct {
	DB     *pgxpool.Pool
	Kafka  *kafka.Writer
	Logger *zap.Logger
}

func main() {
	logger, _ := zap.NewProduction()
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			logger.Fatal("Failed to sync logger", zap.Error(err))
			return
		}
	}(logger)

	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	if err != nil {
		logger.Fatal("DB Error", zap.Error(err))
		return
	}

	kw := &kafka.Writer{
		Addr:                   kafka.TCP(os.Getenv("KAFKA_BROKERS")),
		Topic:                  "orders_events",
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
	}

	app := &App{DB: pool, Kafka: kw, Logger: logger}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.POST("/vi/orders", app.createOrder)

	err = r.Run(":" + os.Getenv("ORDER_SERVICE_PORT"))
	if err != nil {
		logger.Fatal("Error while starting server", zap.Error(err))
		return
	}

	logger.Info("Order Service is running...", zap.String("port", os.Getenv("ORDER_SERVICE_PORT")))
}

func (a *App) createOrder(c *gin.Context) {
	var order Order
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Payload"})
		return
	}

	tx, err := a.DB.Begin(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while starting transaction"})
		return
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil {

		}
	}(tx, c)

	err = a.DB.QueryRow(c,
		"INSERT INTO orders (customer_id, amount, status) VALUES ($1, $2, $3) RETURNING id, created_at",
		order.CustomerID, order.Amount, "CREATED").Scan(&order.ID, &order.CreatedAt)
	if err != nil {
		a.Logger.Error("Failed to save ORDER", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while saving ORDER: " + err.Error()})
		return
	}

	payload, _ := json.Marshal(order)
	_, err = tx.Exec(c,
		"INSERT INTO outbox (topic, payload) VALUES ($1, $2)",
		"orders_events", payload)
	if err != nil {
		a.Logger.Error("Failed to save OUTBOX", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error while saving OUTBOX: " + err.Error()})
		return
	}

	if err := tx.Commit(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while commiting changes: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}
