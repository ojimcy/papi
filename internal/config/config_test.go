package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected port 8080, got %s", cfg.Port)
	}
	if cfg.PollInterval != 60*time.Second {
		t.Errorf("expected 60s interval, got %v", cfg.PollInterval)
	}
	if cfg.WebhookThreshold != 0.5 {
		t.Errorf("expected threshold 0.5, got %f", cfg.WebhookThreshold)
	}
	if cfg.BybitBaseURL != "https://api2.bybit.com" {
		t.Errorf("expected bybit URL, got %s", cfg.BybitBaseURL)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected log level info, got %s", cfg.LogLevel)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	os.Clearenv()
	t.Setenv("PORT", "9090")
	t.Setenv("POLL_INTERVAL", "30s")
	t.Setenv("WEBHOOK_THRESHOLD", "1.5")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost/testdb")
	t.Setenv("BYBIT_BASE_URL", "https://custom.bybit.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("expected 9090, got %s", cfg.Port)
	}
	if cfg.PollInterval != 30*time.Second {
		t.Errorf("expected 30s, got %v", cfg.PollInterval)
	}
	if cfg.WebhookThreshold != 1.5 {
		t.Errorf("expected 1.5, got %f", cfg.WebhookThreshold)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected debug, got %s", cfg.LogLevel)
	}
	if cfg.DatabaseURL != "postgres://test:test@localhost/testdb" {
		t.Errorf("expected custom DB URL, got %s", cfg.DatabaseURL)
	}
	if cfg.BybitBaseURL != "https://custom.bybit.com" {
		t.Errorf("expected custom bybit URL, got %s", cfg.BybitBaseURL)
	}
}

func TestLoad_InvalidPollInterval(t *testing.T) {
	os.Clearenv()
	t.Setenv("POLL_INTERVAL", "invalid")

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid poll interval")
	}
}

func TestLoad_Pairs(t *testing.T) {
	os.Setenv("PAIRS", "USDT_NGN,USDT_GHS")
	defer os.Unsetenv("PAIRS")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Pairs) != 2 || cfg.Pairs[0] != "USDT_NGN" || cfg.Pairs[1] != "USDT_GHS" {
		t.Errorf("expected [USDT_NGN, USDT_GHS], got %v", cfg.Pairs)
	}
}

func TestLoad_PairsDefault(t *testing.T) {
	os.Unsetenv("PAIRS")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Pairs) != 1 || cfg.Pairs[0] != "USDT_NGN" {
		t.Errorf("expected [USDT_NGN], got %v", cfg.Pairs)
	}
}

func TestLoad_InvalidThresholdIgnored(t *testing.T) {
	os.Clearenv()
	t.Setenv("WEBHOOK_THRESHOLD", "notanumber")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.WebhookThreshold != 0.5 {
		t.Errorf("expected default 0.5, got %f", cfg.WebhookThreshold)
	}
}
