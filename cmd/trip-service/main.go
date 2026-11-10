package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/Kate-Mars/go_labs/internal/config"
	"github.com/Kate-Mars/go_labs/internal/container"
	"github.com/Kate-Mars/go_labs/internal/http/router"
	"github.com/Kate-Mars/go_labs/internal/infra/telemetry"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	log := newLogger(cfg.LogLevel)

	ctx := context.Background()

	traceShutdown, err := telemetry.SetupTracing(ctx, cfg)
	if err != nil {
		log.Error("telemetry tracing setup failed", "err", err)
		os.Exit(1)
	}
	metricShutdown, err := telemetry.SetupMetrics(ctx, cfg)
	if err != nil {
		log.Error("telemetry metrics setup failed", "err", err)
		os.Exit(1)
	}

	cnt, err := container.New(ctx, cfg, log)
	if err != nil {
		log.Error("container init failed", "err", err)
		os.Exit(1)
	}

	handler := router.NewRouter(log, cnt.Router, cnt.Metrics)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server starting", "addr", cfg.HTTPAddr)
		errCh <- srv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "err", err)
			shutdownTelemetry(log, traceShutdown, metricShutdown)
			cnt.Pool.Close()
			os.Exit(1)
		}
	case sig := <-stop:
		log.Info("stop signal received", "signal", sig.String())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "err", err)
		cancel()
		shutdownTelemetry(log, traceShutdown, metricShutdown)
		cnt.Pool.Close()
		os.Exit(1)
	}

	shutdownTelemetry(log, traceShutdown, metricShutdown)
	cancel()
	cnt.Pool.Close()
	log.Info("server stopped")
}

func shutdownTelemetry(log *slog.Logger, traceShutdown, metricShutdown telemetry.ShutdownFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := traceShutdown(ctx); err != nil {
		log.Error("tracer shutdown failed", "err", err)
	}
	if err := metricShutdown(ctx); err != nil {
		log.Error("meter shutdown failed", "err", err)
	}
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
