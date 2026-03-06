package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config holds the application configuration
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

	// Service
	ServiceName    string `mapstructure:"SERVICE_NAME"`
	Environment    string `mapstructure:"ENVIRONMENT"`
	LogLevel       string `mapstructure:"LOG_LEVEL"`
}

// Load loads the configuration from environment variables
func Load(serviceName string) (*Config, error) {
	viper.SetEnvPrefix(serviceName)
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("PORT", 8080)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("SERVICE_NAME", serviceName)
	viper.SetDefault("KAFKA_BROKERS", []string{"localhost:9092"})

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields
	if config.DatabaseURL == "" && os.Getenv(serviceName+"_DATABASE_URL") == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return &config, nil
}

// LoadOptional loads configuration but allows missing optional fields
func LoadOptional(serviceName string) *Config {
	config, _ := Load(serviceName)
	if config == nil {
		config = &Config{
			Port:            8080,
			ServiceName:     serviceName,
			Environment:     "development",
			LogLevel:        "info",
			KafkaBrokers:    []string{"localhost:9092"},
		}
	}
	return config
}
