package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Options controls which environment variables are required at startup.
type Options struct {
	ServiceName       string
	DefaultListenAddr string
	RequireDatabase   bool
}

const (
	encryptionKeyEnv   = "ESCALITE_ENCRYPTION_KEY"
	encryptionKeyBytes = 32
	encryptionKeyHint  = "generate with: openssl rand -hex 32 (see .env.example and docs/specs/07-security-and-auth.md)"
)

// SMTPConfig holds optional outbound SMTP settings. Nil means email is not sent.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Config holds parsed ESCALITE_* environment configuration.
type Config struct {
	ListenAddr            string
	DatabaseURL           string
	LogLevel              string
	EncryptionKey         []byte
	HeartbeatScanInterval time.Duration
	SMTP                  *SMTPConfig
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

	encryptionKey, err := loadEncryptionKey()
	if err != nil {
		return Config{}, err
	}
	cfg.EncryptionKey = encryptionKey

	if err := validateLogLevel(cfg.LogLevel); err != nil {
		return Config{}, err
	}

	heartbeatScanInterval, err := loadHeartbeatScanInterval()
	if err != nil {
		return Config{}, err
	}
	cfg.HeartbeatScanInterval = heartbeatScanInterval

	smtpCfg, err := loadSMTP()
	if err != nil {
		return Config{}, err
	}
	cfg.SMTP = smtpCfg

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

func loadHeartbeatScanInterval() (time.Duration, error) {
	const (
		envKey  = "ESCALITE_HEARTBEAT_SCAN_INTERVAL"
		defaultInterval = time.Minute
		minInterval     = time.Second
	)

	value := strings.TrimSpace(os.Getenv(envKey))
	if value == "" {
		return defaultInterval, nil
	}

	interval, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: use a Go duration such as 1m or 30s", envKey, value)
	}
	if interval < minInterval {
		return 0, fmt.Errorf("%s must be at least %s", envKey, minInterval)
	}

	return interval, nil
}

func loadEncryptionKey() ([]byte, error) {
	value := strings.TrimSpace(os.Getenv(encryptionKeyEnv))
	if value == "" {
		return nil, fmt.Errorf("missing required environment variable %s: %s", encryptionKeyEnv, encryptionKeyHint)
	}
	if isEncryptionKeyPlaceholder(value) {
		return nil, fmt.Errorf("%s is empty or a placeholder: %s", encryptionKeyEnv, encryptionKeyHint)
	}
	if len(value) != encryptionKeyBytes*2 {
		return nil, fmt.Errorf("invalid %s: expected %d hex characters (%d bytes); %s",
			encryptionKeyEnv, encryptionKeyBytes*2, encryptionKeyBytes, encryptionKeyHint)
	}

	key, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: must be hex-encoded 32-byte key; %s", encryptionKeyEnv, encryptionKeyHint)
	}
	if isAllZeros(key) {
		return nil, fmt.Errorf("%s must not be all zeros: %s", encryptionKeyEnv, encryptionKeyHint)
	}

	return key, nil
}

func isEncryptionKeyPlaceholder(value string) bool {
	switch strings.ToLower(value) {
	case "changeme", "change-me", "change_me", "placeholder", "replace-me", "replace_me",
		"your-key-here", "your_key_here", "secret", "xxx":
		return true
	default:
		return false
	}
}

func isAllZeros(key []byte) bool {
	for _, b := range key {
		if b != 0 {
			return false
		}
	}
	return true
}

func loadSMTP() (*SMTPConfig, error) {
	host := strings.TrimSpace(os.Getenv("ESCALITE_SMTP_HOST"))
	if host == "" {
		return nil, nil
	}

	from := strings.TrimSpace(os.Getenv("ESCALITE_SMTP_FROM"))
	if from == "" {
		return nil, fmt.Errorf("ESCALITE_SMTP_FROM is required when ESCALITE_SMTP_HOST is set")
	}

	port := 587
	if rawPort := strings.TrimSpace(os.Getenv("ESCALITE_SMTP_PORT")); rawPort != "" {
		parsed, err := strconv.Atoi(rawPort)
		if err != nil || parsed < 1 || parsed > 65535 {
			return nil, fmt.Errorf("invalid ESCALITE_SMTP_PORT %q: use a port between 1 and 65535", rawPort)
		}
		port = parsed
	}

	return &SMTPConfig{
		Host:     host,
		Port:     port,
		Username: strings.TrimSpace(os.Getenv("ESCALITE_SMTP_USERNAME")),
		Password: os.Getenv("ESCALITE_SMTP_PASSWORD"),
		From:     from,
	}, nil
}
