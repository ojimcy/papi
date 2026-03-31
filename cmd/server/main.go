package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zapi-api/zapi/internal/api"
	"github.com/zapi-api/zapi/internal/cache"
	"github.com/zapi-api/zapi/internal/config"
	"github.com/zapi-api/zapi/internal/poller"
	"github.com/zapi-api/zapi/internal/store"
	"github.com/zapi-api/zapi/internal/webhook"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	setupLogger(cfg.LogLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := store.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := store.Migrate(ctx, db.Pool(), "migrations"); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	rateCache := cache.New()

	httpClient := &http.Client{Timeout: 10 * time.Second}
	fetcher := poller.NewBybitFetcher(cfg.BybitBaseURL, httpClient)

	webhookClient := &http.Client{Timeout: 5 * time.Second}
	dispatcher := webhook.New(cfg.WebhookThreshold, webhookClient)

	for _, pair := range cfg.Pairs {
		p := poller.NewPoller(fetcher, db, rateCache, dispatcher, pair)
		go p.Start(ctx, cfg.PollInterval)
	}

	router := api.NewRouter(rateCache, db)
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	server.Shutdown(shutdownCtx)
}

func setupLogger(level string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})))
}
