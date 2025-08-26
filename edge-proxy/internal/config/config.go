package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	// Server configuration
	Port        int    `mapstructure:"port"`
	MetricsPort int    `mapstructure:"metrics_port"`
	LogLevel    string `mapstructure:"log_level"`
	Region      string `mapstructure:"region"`

	// Control plane configuration
	ControlPlaneURL string `mapstructure:"control_plane_url"`

	// Redis configuration
	RedisURL string `mapstructure:"redis_url"`

	// Cache configuration
	CacheSize   string `mapstructure:"cache_size"`
	DefaultTTL  int    `mapstructure:"default_ttl"`
	MaxCacheAge int    `mapstructure:"max_cache_age"`
	MinCacheAge int    `mapstructure:"min_cache_age"`

	// Rate limiting configuration
	RateLimitRPS   int `mapstructure:"rate_limit_rps"`
	RateLimitBurst int `mapstructure:"rate_limit_burst"`

	// TLS configuration
	TLSEnabled    bool   `mapstructure:"tls_enabled"`
	TLSCertFile   string `mapstructure:"tls_cert_file"`
	TLSKeyFile    string `mapstructure:"tls_key_file"`
	AutoCertEmail string `mapstructure:"autocert_email"`
	AutoCertHosts string `mapstructure:"autocert_hosts"`

	// Health check configuration
	HealthCheckInterval int `mapstructure:"health_check_interval"`
	HealthCheckTimeout  int `mapstructure:"health_check_timeout"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set defaults
	viper.SetDefault("port", 8081)
	viper.SetDefault("metrics_port", 9092)
	viper.SetDefault("log_level", "info")
	viper.SetDefault("region", "local")
	viper.SetDefault("control_plane_url", "http://localhost:8080")
	viper.SetDefault("redis_url", "redis://localhost:6379")
	viper.SetDefault("cache_size", "100MB")
	viper.SetDefault("default_ttl", 3600)
	viper.SetDefault("max_cache_age", 86400)
	viper.SetDefault("min_cache_age", 60)
	viper.SetDefault("rate_limit_rps", 1000)
	viper.SetDefault("rate_limit_burst", 2000)
	viper.SetDefault("tls_enabled", false)
	viper.SetDefault("health_check_interval", 30)
	viper.SetDefault("health_check_timeout", 10)

	// Allow environment variables
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
