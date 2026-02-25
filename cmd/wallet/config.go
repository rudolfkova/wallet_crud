// Package main ...
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

// Config ...
type Config struct {
	DatabaseURL string `env:"DATABASE_URL,required"`
	BindAddr    string `env:"BIND_ADDR,default=:8080"`
	LogLevel    string `env:"LOG_LEVEL,default=info"`
}

// parseConfig ...
func parseConfig() (Config, error) {
	if err := godotenv.Load("config.env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load env variables from file: %w", err)
	}

	var c Config
	if err := envconfig.Process(context.Background(), &c); err != nil {
		return Config{}, fmt.Errorf("parse env variables to config: %w", err)
	}

	if c.BindAddr == "" {
		return Config{}, errors.New("var BIND_ADDRESS is required")
	}

	if c.DatabaseURL == "" {
		return Config{}, errors.New("var DATABASE_URL is required")
	}

	return c, nil
}
