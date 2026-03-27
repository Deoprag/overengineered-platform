package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"
)

type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
}

type AuthRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type App struct {
	Logger     *zap.Logger
	DB         *pgxpool.Pool
	PrivateKey *rsa.PrivateKey
}

type PasswordConfig struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

var config = PasswordConfig{
	time:    3,
	memory:  64 * 1024,
	threads: 2,
	keyLen:  32,
}

func main() {
	logger, _ := zap.NewProduction()
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			fmt.Printf("ERROR: Could not sync logger: %v", err)
			os.Exit(1)
		}
	}(logger)

	signKey, err := os.ReadFile(os.Getenv("PRIVATE_KEY_PATH"))
	if err != nil {
		logger.Fatal("Failed to read private key", zap.Error(err))
		return
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(signKey)
	if err != nil {
		logger.Fatal("Failed to parse private key", zap.Error(err))
	}

	pool, err := connectDB(context.Background(), os.Getenv("DB_URL"), logger)

	app := &App{
		Logger:     logger,
		DB:         pool,
		PrivateKey: privateKey,
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.POST("/v1/login", app.login)

	err = r.Run(":" + os.Getenv("APP_PORT"))
	if err != nil {
		logger.Fatal("Error while starting server", zap.Error(err))
		return
	}
}

func connectDB(ctx context.Context, connString string, logger *zap.Logger) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	var err error

	for i := range 5 {
		pool, err = pgxpool.New(ctx, connString)
		if err == nil {
			err = pool.Ping(ctx)
			if err == nil {
				return pool, nil
			}
		}

		logger.Info("Awaiting Postgres...", zap.Int("retry", i+1))
		time.Sleep(2 * time.Second)
	}

	return nil, err
}

func (a *App) login(c *gin.Context) {
	var auth AuthRequest
	if err := c.ShouldBindJSON(&auth); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Payload"})
		return
	}

	a.Logger.Info("Trying to login user", zap.String("username", auth.Username))

	var user User
	err := a.DB.QueryRow(c, "SELECT id, username, password_hash FROM users WHERE username = $1", auth.Username).Scan(&user.ID, &user.Username, &user.PasswordHash)
	if err != nil {
		a.Logger.Error("Failed to login user", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Credentials"})
		return
	}

	if !a.verifyPassword(auth.Password, user.PasswordHash) {
		a.Logger.Error("Failed to login user", zap.String("username", auth.Username))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Credentials"})
		return
	}

	tokenString, err := a.generateJWT(user)
	if err != nil {
		a.Logger.Error("Failed to generate JWT for user ", zap.String("username", auth.Username), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while generating JWT"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

func (a *App) generateJWT(user User) (string, error) {
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"name": user.Username,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
		"iss":  "auth-service",
		"aud":  "overengineered-platform",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(a.PrivateKey)
}

func (a *App) verifyPassword(password, passwordHash string) bool {
	vals := strings.Split(passwordHash, "$")
	if len(vals) != 6 {
		a.Logger.Error("Invalid password hash. Wrong length", zap.String("hash", passwordHash))
		return false
	}

	var version int
	_, err := fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil || version != argon2.Version {
		a.Logger.Error("Invalid password hash. Wrong version", zap.String("hash", passwordHash))
		return false
	}

	var m, t uint32
	var p uint8

	scan, err := fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &m, &t, &p)
	if err != nil || scan != 3 {
		a.Logger.Error("Invalid password hash. Wrong 'mtp' format", zap.String("hash", passwordHash))
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(vals[4])
	if err != nil {
		a.Logger.Error("Invalid password hash. Invalid salt", zap.String("hash", passwordHash))
		return false
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(vals[5])
	if err != nil {
		a.Logger.Error("Invalid password hash. Invalid hash", zap.String("hash", passwordHash))
		return false
	}

	comparisonHash := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(decodedHash)))

	return subtle.ConstantTimeCompare(decodedHash, comparisonHash) == 1
}

func (a *App) hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, config.time, config.memory, config.threads, config.keyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, config.memory, config.time, config.threads, b64Salt, b64Hash)

	return encodedHash, nil
}
