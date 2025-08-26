package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Port        string
	MetricsPort string
	DatabaseURL string
	RedisURL    string
	LogLevel    string
	JWTSecret   string
	Environment string
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/naijcloud/")

	// Set defaults
	viper.SetDefault("port", "8080")
	viper.SetDefault("metrics_port", "9091")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("environment", "development")

	// Environment variables
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	config := &Config{
		Port:        viper.GetString("port"),
		MetricsPort: viper.GetString("metrics_port"),
		DatabaseURL: getEnvOrPanic("DATABASE_URL"),
		RedisURL:    getEnvOrPanic("REDIS_URL"),
		LogLevel:    viper.GetString("log_level"),
		JWTSecret:   getEnvOrDefault("JWT_SECRET", "dev-secret-change-in-production"),
		Environment: viper.GetString("environment"),
	}

	return config, nil
}

func getEnvOrPanic(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic("Environment variable " + key + " is required")
	}
	return value
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
