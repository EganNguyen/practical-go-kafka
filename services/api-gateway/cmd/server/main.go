package main

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/practical-go-kafka/shared/pkg/jwt"
	"github.com/practical-go-kafka/api-gateway/internal/config"
	"github.com/practical-go-kafka/api-gateway/internal/handler"
	"github.com/practical-go-kafka/api-gateway/internal/middleware"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize JWT manager (for public key verification)
	jwtManager, err := initJWTManager(cfg.JWTPublicKeyPath)
	if err != nil {
		log.Fatalf("Failed to initialize JWT manager: %v", err)
	}

	// Initialize proxy handler
	proxyHandler := handler.NewProxy(
		cfg.UserServiceURL,
		cfg.ProductServiceURL,
		cfg.CartServiceURL,
		cfg.OrderServiceURL,
		cfg.PaymentServiceURL,
		cfg.SearchServiceURL,
	)

	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitPerMinute)

	// Setup Gin router
	router := gin.Default()

	// Global middleware
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.RateLimitMiddlewareFunc(rateLimiter, cfg.RateLimitPerMinute))

	// Health check (no auth required)
	router.GET("/health", proxyHandler.Health)

	// Auth routes (no auth required, forward to user-service)
	authGroup := router.Group("/v1/auth")
	{
		authGroup.POST("/register", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.UserServiceURL)
		})
		authGroup.POST("/login", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.UserServiceURL)
		})
		authGroup.POST("/refresh", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.UserServiceURL)
		})
	}

	// Product routes (public, no auth required)
	productsGroup := router.Group("/v1/products")
	{
		productsGroup.GET("", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.ProductServiceURL)
		})
		productsGroup.GET("/:id", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.ProductServiceURL)
		})
	}

	// Search routes (public, no auth required)
	searchGroup := router.Group("/v1/search")
	{
		searchGroup.GET("", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.SearchServiceURL)
		})
		searchGroup.GET("/autocomplete", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.SearchServiceURL)
		})
	}

	// Protected routes (auth required)
	protectedAuthMiddleware := middleware.JWTAuthMiddleware(jwtManager)

	// User routes (protected)
	userGroup := router.Group("/v1/users")
	userGroup.Use(protectedAuthMiddleware)
	{
		userGroup.GET("/me", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.UserServiceURL)
		})
		userGroup.PATCH("/me", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.UserServiceURL)
		})
		userGroup.DELETE("/me", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.UserServiceURL)
		})
	}

	// Cart routes (protected)
	cartGroup := router.Group("/v1/cart")
	cartGroup.Use(protectedAuthMiddleware)
	{
		cartGroup.GET("", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.CartServiceURL)
		})
		cartGroup.PUT("/items", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.CartServiceURL)
		})
		cartGroup.DELETE("/items/:sku", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.CartServiceURL)
		})
		cartGroup.DELETE("", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.CartServiceURL)
		})
		cartGroup.POST("/checkout-preview", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.CartServiceURL)
		})
	}

	// Order routes (protected)
	orderGroup := router.Group("/v1/orders")
	orderGroup.Use(protectedAuthMiddleware)
	{
		orderGroup.POST("", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.OrderServiceURL)
		})
		orderGroup.GET("", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.OrderServiceURL)
		})
		orderGroup.GET("/:id", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.OrderServiceURL)
		})
	}

	// Payment routes (protected)
	paymentGroup := router.Group("/v1/payments")
	paymentGroup.Use(protectedAuthMiddleware)
	{
		paymentGroup.POST("", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.PaymentServiceURL)
		})
		paymentGroup.GET("/:id", func(c *gin.Context) {
			proxyHandler.ForwardRequest(c, cfg.PaymentServiceURL)
		})
	}

	// Start server with timeouts
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting API Gateway on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// initJWTManager initializes the JWT manager with the public key
func initJWTManager(publicKeyPath string) (*jwt.Manager, error) {
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

	// Create a dummy private key for the manager (not used in gateway)
	return jwt.NewManager(nil, rsaPublicKey), nil
}
