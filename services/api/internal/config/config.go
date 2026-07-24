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

// OIDCConfig holds optional OIDC client settings. Nil means OIDC is disabled.
type OIDCConfig struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	SuccessURL   string
}

// SMTPConfig holds optional outbound SMTP settings. Nil means email is not sent (noop sender).
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// PasswordResetRateLimitConfig controls password reset request throttling.
type PasswordResetRateLimitConfig struct {
	EmailLimit  int
	EmailWindow time.Duration
	IPLimit     int
	IPWindow    time.Duration
}

// GraphQLConfig controls GraphQL transport hardening limits.
type GraphQLConfig struct {
	MaxDepth      int
	MaxComplexity int
}

// Config holds parsed ESCALITE_* environment configuration.
type Config struct {
	ListenAddr     string
	DatabaseURL    string
	LogLevel       string
	EncryptionKey  []byte
	PublicURL      string
	AppOrigin      string
	Environment    string
	OIDC           *OIDCConfig
	SMTP           *SMTPConfig
	PasswordReset  PasswordResetRateLimitConfig
	GraphQL        GraphQLConfig
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

	oidcCfg, err := loadOIDC()
	if err != nil {
		return Config{}, err
	}
	cfg.OIDC = oidcCfg

	smtpCfg, err := loadSMTP()
	if err != nil {
		return Config{}, err
	}
	cfg.SMTP = smtpCfg

	cfg.PublicURL = strings.TrimSpace(os.Getenv("ESCALITE_PUBLIC_URL"))
	cfg.AppOrigin = strings.TrimSuffix(strings.TrimSpace(os.Getenv("ESCALITE_APP_ORIGIN")), "/")
	cfg.PasswordReset = loadPasswordResetRateLimit()
	cfg.Environment = loadEnvironment()
	cfg.GraphQL = loadGraphQLConfig()

	if err := validateLogLevel(cfg.LogLevel); err != nil {
		return Config{}, err
	}

	if strings.TrimSpace(cfg.ListenAddr) == "" {
		return Config{}, fmt.Errorf("ESCALITE_HTTP_ADDR must not be empty")
	}

	return cfg, nil
}

func loadOIDC() (*OIDCConfig, error) {
	issuerURL := strings.TrimSpace(os.Getenv("ESCALITE_OIDC_ISSUER_URL"))
	clientID := strings.TrimSpace(os.Getenv("ESCALITE_OIDC_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("ESCALITE_OIDC_CLIENT_SECRET"))

	if issuerURL == "" && clientID == "" && clientSecret == "" {
		return nil, nil
	}

	if issuerURL == "" || clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf(
			"partial OIDC configuration: set all of ESCALITE_OIDC_ISSUER_URL, ESCALITE_OIDC_CLIENT_ID, and ESCALITE_OIDC_CLIENT_SECRET, or leave all unset",
		)
	}

	redirectURL := strings.TrimSpace(os.Getenv("ESCALITE_OIDC_REDIRECT_URL"))
	publicURL := strings.TrimSpace(os.Getenv("ESCALITE_PUBLIC_URL"))
	if redirectURL == "" {
		if publicURL == "" {
			return nil, fmt.Errorf(
				"OIDC enabled but redirect URL unknown: set ESCALITE_OIDC_REDIRECT_URL or ESCALITE_PUBLIC_URL",
			)
		}
		redirectURL = strings.TrimSuffix(publicURL, "/") + "/api/v1/auth/oidc/callback"
	}

	successURL := publicURL
	if successURL == "" {
		successURL = "/"
	} else {
		successURL = strings.TrimSuffix(successURL, "/")
	}

	return &OIDCConfig{
		IssuerURL:    issuerURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		SuccessURL:   successURL,
	}, nil
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

func loadPasswordResetRateLimit() PasswordResetRateLimitConfig {
	return PasswordResetRateLimitConfig{
		EmailLimit:  envIntOrDefault("ESCALITE_PASSWORD_RESET_RATE_LIMIT_EMAIL", 5),
		EmailWindow: envDurationOrDefault("ESCALITE_PASSWORD_RESET_RATE_LIMIT_EMAIL_WINDOW", time.Hour),
		IPLimit:     envIntOrDefault("ESCALITE_PASSWORD_RESET_RATE_LIMIT_IP", 20),
		IPWindow:    envDurationOrDefault("ESCALITE_PASSWORD_RESET_RATE_LIMIT_IP_WINDOW", time.Hour),
	}
}

func loadEnvironment() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ESCALITE_ENV"))) {
	case "production", "prod":
		return "production"
	default:
		return "development"
	}
}

func loadGraphQLConfig() GraphQLConfig {
	return GraphQLConfig{
		MaxDepth:      envIntOrDefault("ESCALITE_GRAPHQL_MAX_DEPTH", 15),
		MaxComplexity: envIntOrDefault("ESCALITE_GRAPHQL_MAX_COMPLEXITY", 100),
	}
}

func (c Config) IsProduction() bool {
	return c.Environment == "production"
}

func envIntOrDefault(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
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
