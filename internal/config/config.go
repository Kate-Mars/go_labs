package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	LogLevel        string
	ShutdownTimeout time.Duration

	DatabaseURL             string
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseMaxConnLifetime time.Duration
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
}

// Load() (*Config, error)

func Load() (*Config, error) {
	cfg := &Config{}
	var err error

	if cfg.HTTPAddr, err = requireEnv("HTTP_ADDR"); err != nil {
		return nil, err
	}
	if cfg.DatabaseURL, err = requireEnv("DATABASE_URL"); err != nil {
		return nil, err
	}

	cfg.LogLevel = getEnv("LOG_LEVEL", "info")

	if cfg.ShutdownTimeout, err = getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second); err != nil {
		return nil, err
	}
	if cfg.DatabaseMaxConns, err = getEnvInt32("DATABASE_MAX_CONNS", 10); err != nil {
		return nil, err
	}
	if cfg.DatabaseMinConns, err = getEnvInt32("DATABASE_MIN_CONNS", 2); err != nil {
		return nil, err
	}
	if cfg.DatabaseMaxConnLifetime, err = getEnvDuration("DATABASE_MAX_CONN_LIFETIME", 30*time.Minute); err != nil {
		return nil, err
	}
	if cfg.DatabaseConnectTimeout, err = getEnvDuration("DATABASE_CONNECT_TIMEOUT", 5*time.Second); err != nil {
		return nil, err
	}
	if cfg.DatabaseQueryTimeout, err = getEnvDuration("DATABASE_QUERY_TIMEOUT", 3*time.Second); err != nil {
		return nil, err
	}

	if cfg.ShutdownTimeout <= 0 {
		return nil, fmt.Errorf("SHUTDOWN_TIMEOUT must be > 0, got %s", cfg.ShutdownTimeout)
	}
	if cfg.DatabaseMaxConns <= 0 {
		return nil, fmt.Errorf("DATABASE_MAX_CONNS must be > 0, got %d", cfg.DatabaseMaxConns)
	}
	if cfg.DatabaseMinConns < 0 || cfg.DatabaseMinConns > cfg.DatabaseMaxConns {
		return nil, fmt.Errorf("DATABASE_MIN_CONNS must be in [0, %d], got %d",
			cfg.DatabaseMaxConns, cfg.DatabaseMinConns)
	}

	return cfg, nil
}

// require

func requireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required env %s is not set", key)
	}
	return v, nil
}

// get

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvDuration(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s=%q: %w", key, v, err)
	}
	return d, nil
}

func getEnvInt32(key string, def int32) (int32, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s=%q: %w", key, v, err)
	}
	return int32(n), nil
}
