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

	appCtx, appCancel := context.WithCancel(context.Background())
	cnt.Positions.Start(appCtx)

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
			shutdownAll(log, cfg, srv, cnt, appCancel, traceShutdown, metricShutdown)
			os.Exit(1)
		}
	case sig := <-stop:
		log.Info("stop signal received", "signal", sig.String())
	}

	shutdownAll(log, cfg, srv, cnt, appCancel, traceShutdown, metricShutdown)
	log.Info("server stopped")
}

func shutdownAll(
	log *slog.Logger,
	cfg *config.Config,
	srv *http.Server,
	cnt *container.Container,
	appCancel context.CancelFunc,
	traceShutdown telemetry.ShutdownFunc,
	metricShutdown telemetry.ShutdownFunc,
) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if srv != nil {
		if err := srv.Shutdown(ctx); err != nil {
			log.Error("http shutdown failed", "err", err)
		}
	}

	if cnt.Positions != nil {
		if err := cnt.Positions.Stop(ctx); err != nil {
			log.Error("positions manager stop failed", "err", err)
		}
	}
	appCancel()

	if traceShutdown != nil {
		if err := traceShutdown(ctx); err != nil {
			log.Error("tracer shutdown failed", "err", err)
		}
	}
	if metricShutdown != nil {
		if err := metricShutdown(ctx); err != nil {
			log.Error("meter shutdown failed", "err", err)
		}
	}

	if cnt != nil && cnt.Pool != nil {
		cnt.Pool.Close()
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
