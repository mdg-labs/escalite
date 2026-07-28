package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/config"
	"github.com/mdg-labs/escalite/services/api/internal/crypto"
	"github.com/mdg-labs/escalite/services/api/internal/email"
	"github.com/mdg-labs/escalite/services/api/internal/graphql"
	"github.com/mdg-labs/escalite/services/api/internal/log"
	"github.com/mdg-labs/escalite/services/api/internal/migrate"
	"github.com/mdg-labs/escalite/services/api/internal/oidc"
	"github.com/mdg-labs/escalite/services/api/internal/queue"
	"github.com/mdg-labs/escalite/services/api/internal/ratelimit"
	"github.com/mdg-labs/escalite/services/api/internal/realtime"
	"github.com/mdg-labs/escalite/services/api/internal/server"
	_ "github.com/mdg-labs/escalite/services/engine/channelsinstall"
	_ "github.com/mdg-labs/escalite/services/integrations/install"
	_ "github.com/mdg-labs/escalite/services/outboundintegrations/install"
)

const serviceName = "api"

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load(config.Options{
		ServiceName:       serviceName,
		DefaultListenAddr: ":8080",
		RequireDatabase:   true,
	})
	if err != nil {
		slog.Error("configuration error", "error", err.Error())
		return 1
	}

	logger := log.NewJSONLogger(serviceName, cfg.LogLevel)
	logger.Info("starting service", "listen_addr", cfg.ListenAddr)

	ctx := context.Background()
	if err := migrate.Up(ctx, cfg.DatabaseURL, logger); err != nil {
		logger.Error("database migration failed", "error", err)
		return 1
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database pool failed", "error", err)
		return 1
	}
	defer pool.Close()

	jobs, err := queue.NewProducer(ctx, pool, logger)
	if err != nil {
		logger.Error("river producer failed", "error", err)
		return 1
	}

	secrets, err := crypto.NewBox(cfg.EncryptionKey)
	if err != nil {
		logger.Error("encryption setup failed", "error", err)
		return 1
	}

	realtimeBridge := realtime.NewBridge(cfg.DatabaseURL, logger)
	realtimeBridge.Start(ctx)

	var oidcProvider *oidc.Provider
	if cfg.OIDC != nil {
		oidcProvider, err = oidc.NewProvider(ctx, oidc.Config{
			IssuerURL:    cfg.OIDC.IssuerURL,
			ClientID:     cfg.OIDC.ClientID,
			ClientSecret: cfg.OIDC.ClientSecret,
			RedirectURL:  cfg.OIDC.RedirectURL,
		})
		if err != nil {
			logger.Error("oidc provider failed", "error", err)
			return 1
		}
		logger.Info("oidc login enabled", "issuer", cfg.OIDC.IssuerURL)
	}
	if cfg.SlackOAuth != nil {
		logger.Info("slack oauth install enabled")
	}
	if cfg.SlackInteractive.SigningSecret != "" {
		logger.Info("slack interactive endpoint enabled")
	}

	handler := server.New(server.Dependencies{
		Logger:     logger,
		Pool:       pool,
		Jobs:       jobs,
		Secrets:       secrets,
		EncryptionKey: cfg.EncryptionKey,
		OIDC:          server.NewOIDCServices(pool, logger, cfg.OIDC, oidcProvider),
		SAML:       server.NewSAMLServices(pool, logger, secrets, cfg.PublicURL, cfg.AppOrigin),
		SlackOAuth: server.NewSlackOAuthServices(pool, logger, secrets, cfg.SlackOAuth),
		Mail:       newMailSender(cfg, logger),
		PublicURL:  cfg.PublicURL,
		AppOrigin:  cfg.AppOrigin,
		PasswordReset: &server.PasswordResetOptions{
			EmailLimiter: ratelimit.NewMemoryLimiter(
				cfg.PasswordReset.EmailLimit,
				cfg.PasswordReset.EmailWindow,
			),
			IPLimiter: ratelimit.NewMemoryLimiter(
				cfg.PasswordReset.IPLimit,
				cfg.PasswordReset.IPWindow,
			),
		},
		HeartbeatPing: &server.HeartbeatPingOptions{
			TokenLimiter: ratelimit.NewMemoryLimiter(
				cfg.HeartbeatPing.Limit,
				cfg.HeartbeatPing.Window,
			),
		},
		InboundEmail: &server.InboundEmailOptions{
			RelaySecret:          cfg.InboundEmail.RelaySecret,
			Domain:               cfg.InboundEmail.Domain,
			RequireAuthenticated: cfg.InboundEmail.RequireAuthenticated,
		},
		SlackInteractive: &server.SlackInteractiveOptions{
			SigningSecret: cfg.SlackInteractive.SigningSecret,
		},
		GraphQL: graphql.Options{
			Production:                       cfg.IsProduction(),
			MaxDepth:                         cfg.GraphQL.MaxDepth,
			MaxComplexity:                    cfg.GraphQL.MaxComplexity,
			PublicURL:                        cfg.PublicURL,
			OIDCEnabled:                      cfg.OIDC != nil,
			SlackIncidentChannelNameTemplate: cfg.SlackIncidentChannelNameTemplate,
		},
		Realtime: realtimeBridge.Hub,
	})
	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		logger.Error("server error", "error", err)
		return 1
	case sig := <-stop:
		logger.Info("shutdown signal received", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		return 1
	}

	realtimeBridge.Close()

	logger.Info("service stopped")
	return 0
}

func newMailSender(cfg config.Config, logger *slog.Logger) email.Sender {
	if cfg.SMTP == nil {
		logger.Info("smtp not configured; password reset emails are discarded")
		return email.NoopSender{}
	}

	logger.Info("smtp configured for outbound email", "host", cfg.SMTP.Host, "port", cfg.SMTP.Port)
	return email.NewSMTPSender(email.SMTPConfig{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		Username: cfg.SMTP.Username,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
	})
}
