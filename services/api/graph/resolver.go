package graph

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/crypto"
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
	}
}
