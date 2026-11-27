package config

import (
	"fmt"
	"os"
	"runtime/debug"
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

	OTELEndpoint           string
	OTELServiceName        string
	OTELExporterProtocol   string
	OTELTracesSampler      string
	OTELTracesSamplerArg   string
	OTELResourceAttributes string
	ServiceVersion         string
}

func Load() (*Config, error) {
	cfg := &Config{}
	var err error

	// --- обязательные ---
	if cfg.HTTPAddr, err = requireEnv("HTTP_ADDR"); err != nil {
		return nil, err
	}
	if cfg.DatabaseURL, err = requireEnv("DATABASE_URL"); err != nil {
		return nil, err
	}

	// --- с дефолтами ---
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

	// --- телеметрия (ЛР2) ---
	cfg.OTELEndpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318")
	cfg.OTELServiceName = getEnv("OTEL_SERVICE_NAME", "trip-service")
	cfg.OTELExporterProtocol = getEnv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	cfg.OTELTracesSampler = getEnv("OTEL_TRACES_SAMPLER", "parentbased_always_on")
	cfg.OTELTracesSamplerArg = getEnv("OTEL_TRACES_SAMPLER_ARG", "")
	cfg.OTELResourceAttributes = getEnv("OTEL_RESOURCE_ATTRIBUTES", "")
	cfg.ServiceVersion = detectVersion()

	// --- валидация ---
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

	switch cfg.OTELExporterProtocol {
	case "http/protobuf", "grpc":
	default:
		return nil, fmt.Errorf("OTEL_EXPORTER_OTLP_PROTOCOL must be http/protobuf or grpc, got %q",
			cfg.OTELExporterProtocol)
	}

	switch cfg.OTELTracesSampler {
	case "always_on", "always_off", "traceidratio",
		"parentbased_always_on", "parentbased_always_off", "parentbased_traceidratio":
	default:
		return nil, fmt.Errorf("OTEL_TRACES_SAMPLER has unsupported value %q", cfg.OTELTracesSampler)
	}

	return cfg, nil
}

// detectVersion достаёт версию сборки из build info (git tag или module version).
// Если build info недоступна (go run), возвращает "dev".
func detectVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	// ищем vcs.revision
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && len(s.Value) >= 7 {
			return s.Value[:7]
		}
	}
	return "dev"
}

// --- хелперы ---

func requireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required env %s is not set", key)
	}
	return v, nil
}

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
