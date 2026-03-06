package main

import (
	"context"
	"database/sql"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/practical-go-kafka/shared/events"
	"github.com/practical-go-kafka/shared/pkg/jwt"
	"github.com/practical-go-kafka/user-service/internal/config"
	"github.com/practical-go-kafka/user-service/internal/handler"
	"github.com/practical-go-kafka/user-service/internal/middleware"
	"github.com/practical-go-kafka/user-service/internal/repository"
	"github.com/practical-go-kafka/user-service/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := initDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize JWT manager
	jwtManager, err := initJWTManager(cfg.JWTPrivateKeyPath, cfg.JWTPublicKeyPath)
	if err != nil {
		log.Fatalf("Failed to initialize JWT manager: %v", err)
	}

	// Initialize Kafka producer
	kafkaProducer := events.NewProducer(cfg.KafkaBrokers)
	defer kafkaProducer.Close()

	// Initialize repositories, services, and handlers
	userRepo := repository.NewPostgresUserRepository(db)
	userService := service.NewUserService(
		userRepo,
		jwtManager,
		time.Duration(cfg.AccessTokenTTL)*time.Second,
		time.Duration(cfg.RefreshTokenTTL)*time.Second,
	)
	userHandler := handler.NewUserHandler(userService)

	// Setup Gin router
	router := gin.Default()

	// Register middleware
	router.Use(middleware.ErrorHandlingMiddleware())

	// Public routes (no auth required)
	public := router.Group("/v1/auth")
	{
		public.POST("/register", userHandler.Register)
		public.POST("/login", userHandler.Login)
		public.POST("/refresh", userHandler.RefreshToken)
	}

	// Protected routes (auth required)
	protected := router.Group("/v1/users")
	protected.Use(middleware.AuthMiddleware(jwtManager))
	{
		protected.GET("/me", userHandler.GetProfile)
		protected.PATCH("/me", userHandler.UpdateProfile)
		protected.DELETE("/me", userHandler.DeleteAccount)
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Starting user-service on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// initDB initializes the PostgreSQL database connection
func initDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// initJWTManager initializes the JWT manager with RSA keys
func initJWTManager(privateKeyPath, publicKeyPath string) (*jwt.Manager, error) {
	// Read private key
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Read public key
	publicKeyBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	publicKey, err := x509.ParsePKIXPublicKey(publicKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not an RSA key")
	}

	return jwt.NewManager(privateKey, rsaPublicKey), nil
}
