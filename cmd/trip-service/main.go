package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/Kate-Mars/go_labs/internal/config"
	"github.com/Kate-Mars/go_labs/internal/container"
	"github.com/Kate-Mars/go_labs/internal/http/router"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	log := newLogger(cfg.LogLevel)

	cnt, err := container.New(context.Background(), cfg, log)
	if err != nil {
		log.Error("container init failed", "err", err)
		os.Exit(1)
	}
	defer cnt.Pool.Close()

	handler := router.NewRouter(log, cnt.Router)

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: handler,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server starting", "addr", cfg.HTTPAddr)
		errCh <- srv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "err", err)
			os.Exit(1)
		}
	case sig := <-stop:
		log.Info("stop signal received", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", "err", err)
		os.Exit(1)
	}

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
