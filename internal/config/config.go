// Package config reads the process environment once, at startup, so no other
// package needs to know that environment variables exist.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL     string
	Port            string
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
	// DBConnectTimeout bounds the startup retry loop. Postgres accepts TCP
	// before it accepts queries, so the first Ping usually fails.
	DBConnectTimeout time.Duration
}

func Load() (Config, error) {
	c := Config{
		DatabaseURL:      env("DATABASE_URL", "postgres://devops:devops@localhost:5432/devops?sslmode=disable"),
		Port:             env("PORT", "8080"),
		ShutdownTimeout:  15 * time.Second,
		DBConnectTimeout: 30 * time.Second,
	}
	if err := c.LogLevel.UnmarshalText([]byte(env("LOG_LEVEL", "info"))); err != nil {
		return c, fmt.Errorf("LOG_LEVEL: %w", err)
	}
	if _, err := strconv.Atoi(c.Port); err != nil {
		return c, fmt.Errorf("PORT must be a number, got %q", c.Port)
	}
	return c, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
