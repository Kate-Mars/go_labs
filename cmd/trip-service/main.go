package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Kate-Mars/go_labs/internal/config"
	"github.com/Kate-Mars/go_labs/internal/repository/postgres"
	httptransport "github.com/Kate-Mars/go_labs/internal/transport/http"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		slog.Warn("failed to load .env", "err", err)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	log := newLogger(cfg.LogLevel)

	pool, err := postgres.NewPool(context.Background(), cfg)
	if err != nil {
		log.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}

	txManager := postgres.NewTxManager(pool)
	repo := postgres.NewTripRepository(pool)
	idemRepo := postgres.NewIdempotencyRepository(pool)
	router := httptransport.NewRouter(log, repo, idemRepo, txManager)
	srv := httptransport.NewServer(cfg.HTTPAddr, log, router)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil {
			log.Error("http server failed", "err", err)
			pool.Close()
			os.Exit(1)
		}
	case sig := <-stop:
		log.Info("stop signal received", "signal", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", "err", err)
		pool.Close()
		os.Exit(1)
	}

	pool.Close()
	log.Info("server stopped")
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	return slog.New(h)
}
