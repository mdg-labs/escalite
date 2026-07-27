package graph

import (
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/crypto"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/email"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	"github.com/mdg-labs/escalite/services/api/internal/queue"
	"github.com/mdg-labs/escalite/services/api/internal/realtime"
)

// Resolver is the root GraphQL resolver with shared dependencies.
type Resolver struct {
	pool                             *pgxpool.Pool
	logger                           *slog.Logger
	audit                            *audit.Recorder
	jobs                             *queue.Producer
	secrets                          *crypto.Box
	realtime                         *realtime.Hub
	publicURL                        string
	oidcEnabled                      bool
	slackOAuthInstallURL             string
	slackIncidentChannelNameTemplate string
	mail                             email.Sender
	passwordReset                    handlers.PasswordResetConfig
}

// NewResolver returns a resolver wired with database and logging dependencies.
func NewResolver(
	pool *pgxpool.Pool,
	logger *slog.Logger,
	jobs *queue.Producer,
	secrets *crypto.Box,
	hub *realtime.Hub,
	publicURL string,
	oidcEnabled bool,
	slackOAuthInstallURL string,
	slackIncidentChannelNameTemplate string,
	mail email.Sender,
	passwordReset handlers.PasswordResetConfig,
) *Resolver {
	return &Resolver{
		pool:                             pool,
		logger:                           logger,
		audit:                            audit.NewRecorder(logger),
		jobs:                             jobs,
		secrets:                          secrets,
		realtime:                         hub,
		publicURL:                        publicURL,
		oidcEnabled:                      oidcEnabled,
		slackOAuthInstallURL:             slackOAuthInstallURL,
		slackIncidentChannelNameTemplate: slackIncidentChannelNameTemplate,
		mail:                             mail,
		passwordReset:                    passwordReset,
	}
}

func (r *Resolver) passwordResetService() *handlers.PasswordResetService {
	cfg := r.passwordReset
	if cfg.PublicURL == "" {
		cfg.PublicURL = r.publicURL
	}
	if cfg.ResetTokenTTL <= 0 {
		cfg.ResetTokenTTL = time.Hour
	}
	mail := r.mail
	if mail == nil {
		mail = email.NoopSender{}
	}
	return handlers.NewPasswordResetService(r.pool, db.New(r.pool), mail, cfg, r.logger)
}

func (r *Resolver) userInviteService() *handlers.UserInviteService {
	cfg := handlers.UserInviteConfig{PublicURL: r.publicURL}
	mail := r.mail
	if mail == nil {
		mail = email.NoopSender{}
	}
	return handlers.NewUserInviteService(r.pool, db.New(r.pool), mail, cfg, r.logger)
}
