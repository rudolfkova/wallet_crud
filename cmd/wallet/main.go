// Package main ...
package main

import (
	"log"
	"log/slog"
	"wallet/pkg/logger"
)

func main() {
	cfg, err := parseConfig()
	if err != nil {
		log.Fatal(err)
	}
	logger := logger.NewLogger(cfg.BindAddr)

	log := logger.With(
		slog.String("BindAddr", cfg.BindAddr),
		slog.String("DatabaseURL", cfg.DatabaseURL),
		slog.String("LogLevel", cfg.LogLevel),
	)

	log.Info("start init service")
}
