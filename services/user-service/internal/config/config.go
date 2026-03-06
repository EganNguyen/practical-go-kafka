package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config holds the user-service configuration
type Config struct {
	// Server
	Port int `mapstructure:"PORT"`

	// Database
	DatabaseURL string `mapstructure:"DATABASE_URL"`

	// Redis
	RedisURL string `mapstructure:"REDIS_URL"`

	// Kafka
	KafkaBrokers []string `mapstructure:"KAFKA_BROKERS"`

	// JWT
	JWTPrivateKeyPath string `mapstructure:"JWT_PRIVATE_KEY_PATH"`
	JWTPublicKeyPath  string `mapstructure:"JWT_PUBLIC_KEY_PATH"`
	AccessTokenTTL    int    `mapstructure:"ACCESS_TOKEN_TTL"`   // seconds (default 15 min)
	RefreshTokenTTL   int    `mapstructure:"REFRESH_TOKEN_TTL"`  // seconds (default 7 days)

	// Service
	ServiceName string `mapstructure:"SERVICE_NAME"`
	Environment string `mapstructure:"ENVIRONMENT"`
	LogLevel    string `mapstructure:"LOG_LEVEL"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	viper.SetEnvPrefix("USER_SERVICE")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("PORT", 8081)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("SERVICE_NAME", "user-service")
	viper.SetDefault("KAFKA_BROKERS", []string{"kafka:9092"})
	viper.SetDefault("ACCESS_TOKEN_TTL", 900)   // 15 minutes
	viper.SetDefault("REFRESH_TOKEN_TTL", 604800) // 7 days

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields
	if config.DatabaseURL == "" {
		config.DatabaseURL = os.Getenv("DATABASE_URL")
		if config.DatabaseURL == "" {
			return nil, fmt.Errorf("DATABASE_URL is required")
		}
	}

	if config.JWTPrivateKeyPath == "" {
		config.JWTPrivateKeyPath = "/etc/secrets/jwt_private_key"
	}
	if config.JWTPublicKeyPath == "" {
		config.JWTPublicKeyPath = "/etc/secrets/jwt_public_key"
	}

	return &config, nil
}
