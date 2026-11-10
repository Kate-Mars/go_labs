package config_test

import (
	"testing"

	"github.com/Kate-Mars/go_labs/internal/config"
)

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
}

func TestLoad_Success(t *testing.T) {
	setEnv(t, "HTTP_ADDR", ":8080")
	setEnv(t, "DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	setEnv(t, "LOG_LEVEL", "debug")
	setEnv(t, "SHUTDOWN_TIMEOUT", "5s")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/db" {
		t.Errorf("DatabaseURL wrong")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.LogLevel)
	}
}

func TestLoad_MissingHTTPAddr(t *testing.T) {
	setEnv(t, "HTTP_ADDR", "")
	setEnv(t, "DATABASE_URL", "postgres://localhost/db")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	setEnv(t, "HTTP_ADDR", ":8080")
	setEnv(t, "DATABASE_URL", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoad_InvalidShutdownTimeout(t *testing.T) {
	setEnv(t, "HTTP_ADDR", ":8080")
	setEnv(t, "DATABASE_URL", "postgres://localhost/db")
	setEnv(t, "SHUTDOWN_TIMEOUT", "abc")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoad_InvalidMaxConns(t *testing.T) {
	setEnv(t, "HTTP_ADDR", ":8080")
	setEnv(t, "DATABASE_URL", "postgres://localhost/db")
	setEnv(t, "DATABASE_MAX_CONNS", "-1")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoad_MinGreaterThanMax(t *testing.T) {
	setEnv(t, "HTTP_ADDR", ":8080")
	setEnv(t, "DATABASE_URL", "postgres://localhost/db")
	setEnv(t, "DATABASE_MAX_CONNS", "2")
	setEnv(t, "DATABASE_MIN_CONNS", "5")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
