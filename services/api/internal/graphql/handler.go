package graphql

import (
	"log/slog"
	"net/http"

	gqlhandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/mdg-labs/escalite/services/api/graph"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

const (
	defaultMaxDepth      = 15
	defaultMaxComplexity = 100
)

// Options configures the GraphQL HTTP handler.
type Options struct {
	Production    bool
	MaxDepth      int
	MaxComplexity int
}

// NewHandler returns a gqlgen server configured with transport hardening and error codes.
func NewHandler(pool *pgxpool.Pool, logger *slog.Logger, opts Options) http.Handler {
	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = defaultMaxDepth
	}

	maxComplexity := opts.MaxComplexity
	if maxComplexity <= 0 {
		maxComplexity = defaultMaxComplexity
	}

	srv := gqlhandler.New(graph.NewExecutableSchema(graph.Config{
		Resolvers: graph.NewResolver(pool, logger),
	}))

	srv.AddTransport(transport.POST{})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	srv.Use(extension.FixedComplexityLimit(maxComplexity))
	srv.Use(DepthLimit{MaxDepth: maxDepth})
	srv.Use(ConditionalIntrospection{Production: opts.Production})
	srv.SetErrorPresenter(gqlerr.Present)

	return handlers.GraphQLHTTPContextMiddleware(srv)
}
