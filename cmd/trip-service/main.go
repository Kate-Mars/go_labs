package main

import (
	"log"
	"github.com/Kate-Mars/go_labs/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config: %v", err)
	}
	log.Printf("config loaded: HTTP_ADDR=%s LOG_LEVEL=%s SHUTDOWN_TIMEOUT=%s",
									cfg.HTTPAddr, cfg.LogLevel, cfg.ShutdownTimeout)
}