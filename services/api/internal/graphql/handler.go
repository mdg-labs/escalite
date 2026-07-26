package graphql

import (
	"log/slog"
	"net/http"
	"time"

	gqlhandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/mdg-labs/escalite/services/api/graph"
	"github.com/mdg-labs/escalite/services/api/internal/crypto"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	"github.com/mdg-labs/escalite/services/api/internal/queue"
	"github.com/mdg-labs/escalite/services/api/internal/realtime"
	"github.com/mdg-labs/escalite/services/engine/channels/slackchannel"
)

const (
	defaultMaxDepth      = 15
	defaultMaxComplexity = 100
)

// Options configures the GraphQL HTTP handler.
type Options struct {
	Production                       bool
	MaxDepth                         int
	MaxComplexity                    int
	PublicURL                        string
	OIDCEnabled                      bool
	SlackOAuthInstallURL             string
	SlackIncidentChannelNameTemplate string
}

// NewHandler returns a gqlgen server configured with transport hardening and error codes.
func NewHandler(pool *pgxpool.Pool, logger *slog.Logger, jobs *queue.Producer, secrets *crypto.Box, hub *realtime.Hub, opts Options) http.Handler {
	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = defaultMaxDepth
	}

	maxComplexity := opts.MaxComplexity
	if maxComplexity <= 0 {
		maxComplexity = defaultMaxComplexity
	}

	channelNameTemplate := opts.SlackIncidentChannelNameTemplate
	if channelNameTemplate == "" {
		channelNameTemplate = slackchannel.DefaultNameTemplate
	}

	srv := gqlhandler.New(graph.NewExecutableSchema(graph.Config{
		Resolvers: graph.NewResolver(
			pool,
			logger,
			jobs,
			secrets,
			hub,
			opts.PublicURL,
			opts.OIDCEnabled,
			opts.SlackOAuthInstallURL,
			channelNameTemplate,
		),
	}))

	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})
	srv.AddTransport(transport.POST{})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	srv.Use(extension.FixedComplexityLimit(maxComplexity))
	srv.Use(DepthLimit{MaxDepth: maxDepth})
	srv.Use(ConditionalIntrospection{Production: opts.Production})
	srv.SetErrorPresenter(gqlerr.Present)

	return handlers.GraphQLHTTPContextMiddleware(srv)
}
