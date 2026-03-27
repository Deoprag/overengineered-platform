package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Auth struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type App struct {
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

	app := &App{Logger: logger}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.POST("/v1/auth", app.authenticate)

	err := r.Run(":" + os.Getenv("ORDER_SERVICE_PORT"))
	if err != nil {
		logger.Fatal("Error while starting server", zap.Error(err))
		return
	}

	logger.Info("Auth Service is running...", zap.String("port", os.Getenv("AUTH_SERVICE_PORT")))
}

func (a *App) authenticate(c *gin.Context) {
	var auth Auth
	if err := c.ShouldBindJSON(&auth); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Payload"})
		return
	}

}
