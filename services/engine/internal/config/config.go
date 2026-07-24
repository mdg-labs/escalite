package config

import (
	"fmt"
	"os"
	"strings"
)

// Options controls which environment variables are required at startup.
type Options struct {
	ServiceName       string
	DefaultListenAddr string
	RequireDatabase   bool
}

// Config holds parsed ESCALITE_* environment configuration.
type Config struct {
	ListenAddr  string
	DatabaseURL string
	LogLevel    string
}

// Load reads and validates configuration from the environment.
func Load(opts Options) (Config, error) {
	cfg := Config{
		ListenAddr: envOrDefault("ESCALITE_HTTP_ADDR", opts.DefaultListenAddr),
		LogLevel:   envOrDefault("ESCALITE_LOG_LEVEL", "info"),
	}

	if opts.RequireDatabase {
		databaseURL, err := requireEnv("ESCALITE_DATABASE_URL",
			"set to your Postgres connection string (see .env.example)")
		if err != nil {
			return Config{}, err
		}
		cfg.DatabaseURL = databaseURL
	}

	if err := validateLogLevel(cfg.LogLevel); err != nil {
		return Config{}, err
	}

	if strings.TrimSpace(cfg.ListenAddr) == "" {
		return Config{}, fmt.Errorf("ESCALITE_HTTP_ADDR must not be empty")
	}

	return cfg, nil
}

func requireEnv(key, hint string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("missing required environment variable %s: %s", key, hint)
	}
	return value, nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func validateLogLevel(level string) error {
	switch strings.ToLower(level) {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("invalid ESCALITE_LOG_LEVEL %q: use debug, info, warn, or error", level)
	}
}
