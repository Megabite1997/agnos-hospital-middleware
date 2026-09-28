// Package config loads service configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	JWTTTL      time.Duration
	HISTimeout  time.Duration
	GinMode     string
}

// Load reads configuration from the environment. DATABASE_URL and JWT_SECRET
// are required; everything else has a sensible default.
func Load() (Config, error) {
	cfg := Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		GinMode:     getEnv("GIN_MODE", "release"),
	}

	var err error
	if cfg.JWTTTL, err = time.ParseDuration(getEnv("JWT_TTL", "24h")); err != nil {
		return Config{}, fmt.Errorf("invalid JWT_TTL: %w", err)
	}
	if cfg.HISTimeout, err = time.ParseDuration(getEnv("HIS_TIMEOUT", "5s")); err != nil {
		return Config{}, fmt.Errorf("invalid HIS_TIMEOUT: %w", err)
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET is required and must be at least 32 characters")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
