package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	DatabaseURL      string
	PollInterval     time.Duration
	WebhookThreshold float64
	BybitBaseURL     string
	LogLevel         string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	pollInterval, err := time.ParseDuration(getEnv("POLL_INTERVAL", "60s"))
	if err != nil {
		return nil, err
	}

	threshold := 0.5
	if v := os.Getenv("WEBHOOK_THRESHOLD"); v != "" {
		var t float64
		if _, err := fmt.Sscanf(v, "%f", &t); err == nil {
			threshold = t
		}
	}

	return &Config{
		Port:             getEnv("PORT", "8080"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://localhost:5432/p2p_rates?sslmode=disable"),
		PollInterval:     pollInterval,
		WebhookThreshold: threshold,
		BybitBaseURL:     getEnv("BYBIT_BASE_URL", "https://api2.bybit.com"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
