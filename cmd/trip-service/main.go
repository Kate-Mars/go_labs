package main

import (
	"bufio"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Kate-Mars/go_labs/internal/config"
	httptransport "github.com/Kate-Mars/go_labs/internal/transport/http"
)

func main() {
	loadDotEnv(".env") // локально; в проде файла нет — просто ничего не делает

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	log := newLogger(cfg.LogLevel)

	router := httptransport.NewRouter(log)
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
			os.Exit(1)
		}
	case sig := <-stop:
		log.Info("stop signal received", "signal", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", "err", err)
		os.Exit(1)
	}
	log.Info("server stopped")
}

// loadDotEnv(path string)
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // нет файла — ок
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if _, exists := os.LookupEnv(k); !exists {
			os.Setenv(k, v)
		}
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
