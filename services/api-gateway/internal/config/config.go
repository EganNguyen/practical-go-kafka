package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds the API Gateway configuration
type Config struct {
	// Server
	Port int `mapstructure:"PORT"`

	// JWT
	JWTPublicKeyPath string `mapstructure:"JWT_PUBLIC_KEY_PATH"`

	// Rate Limiting
	RateLimitPerMinute int `mapstructure:"RATE_LIMIT_PER_MINUTE"`

	// Service endpoints
	UserServiceURL      string `mapstructure:"USER_SERVICE_URL"`
	ProductServiceURL   string `mapstructure:"PRODUCT_SERVICE_URL"`
	CartServiceURL      string `mapstructure:"CART_SERVICE_URL"`
	OrderServiceURL     string `mapstructure:"ORDER_SERVICE_URL"`
	PaymentServiceURL   string `mapstructure:"PAYMENT_SERVICE_URL"`
	SearchServiceURL    string `mapstructure:"SEARCH_SERVICE_URL"`

	// Service
	ServiceName string `mapstructure:"SERVICE_NAME"`
	Environment string `mapstructure:"ENVIRONMENT"`
	LogLevel    string `mapstructure:"LOG_LEVEL"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	viper.SetEnvPrefix("GATEWAY")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("PORT", 8080)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("SERVICE_NAME", "api-gateway")
	viper.SetDefault("RATE_LIMIT_PER_MINUTE", 1000)
	viper.SetDefault("JWT_PUBLIC_KEY_PATH", "/etc/secrets/jwt_public_key")
	viper.SetDefault("USER_SERVICE_URL", "http://user-service:8081")
	viper.SetDefault("PRODUCT_SERVICE_URL", "http://product-service:8082")
	viper.SetDefault("CART_SERVICE_URL", "http://cart-service:8084")
	viper.SetDefault("ORDER_SERVICE_URL", "http://order-service:8085")
	viper.SetDefault("PAYMENT_SERVICE_URL", "http://payment-service:8086")
	viper.SetDefault("SEARCH_SERVICE_URL", "http://search-service:8088")

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
