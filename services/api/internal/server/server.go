package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/config"
	"github.com/mdg-labs/escalite/services/api/internal/cors"
	"github.com/mdg-labs/escalite/services/api/internal/email"
	"github.com/mdg-labs/escalite/services/api/internal/graphql"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	"github.com/mdg-labs/escalite/services/api/internal/queue"
	"github.com/mdg-labs/escalite/services/api/internal/ratelimit"
)

// Dependencies holds runtime services wired into the HTTP router.
type Dependencies struct {
	Logger        *slog.Logger
	Pool          *pgxpool.Pool
	Jobs          *queue.Producer
	OIDC          *OIDCServices
	Mail          email.Sender
	PublicURL     string
	AppOrigin     string
	PasswordReset *PasswordResetOptions
	GraphQL       graphql.Options
}

// PasswordResetOptions overrides password reset wiring (primarily for tests).
type PasswordResetOptions struct {
	EmailLimiter ratelimit.Limiter
	IPLimiter    ratelimit.Limiter
	TokenTTL     time.Duration
}

// OIDCServices holds optional OIDC login handlers when ESCALITE_OIDC_* is configured.
type OIDCServices struct {
	Handler    *handlers.OIDCHandler
	Login      http.HandlerFunc
	Callback   http.HandlerFunc
}

// New returns an HTTP server with health, readiness, and API routes.
func New(deps Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	if deps.AppOrigin != "" {
		r.Use(cors.Middleware(deps.AppOrigin))
	}

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("escalite api\n"))
	})

	if deps.Pool != nil {
		setup := handlers.NewSetupHandler(deps.Pool, deps.Logger)
		login := handlers.NewLoginHandler(deps.Pool, deps.Logger)
		logout := handlers.NewLogoutHandler(deps.Pool, deps.Logger)
		me := handlers.NewMeHandler()
		team := handlers.NewTeamHandler()

		mail := deps.Mail
		if mail == nil {
			mail = email.NoopSender{}
		}

		resetCfg := handlers.PasswordResetConfig{
			PublicURL:     deps.PublicURL,
			ResetTokenTTL: time.Hour,
		}
		if deps.PasswordReset != nil {
			resetCfg.EmailLimiter = deps.PasswordReset.EmailLimiter
			resetCfg.IPLimiter = deps.PasswordReset.IPLimiter
			if deps.PasswordReset.TokenTTL > 0 {
				resetCfg.ResetTokenTTL = deps.PasswordReset.TokenTTL
			}
		}
		passwordReset := handlers.NewPasswordResetHandler(deps.Pool, deps.Logger, mail, resetCfg)

		r.Post("/api/v1/setup", setup.ServeHTTP)
		r.Post("/api/v1/login", login.ServeHTTP)
		r.Post("/api/v1/password-reset/request", passwordReset.Request)
		r.Post("/api/v1/password-reset/confirm", passwordReset.Confirm)

		if deps.OIDC != nil {
			r.Get("/api/v1/auth/oidc/login", deps.OIDC.Login)
			r.Get("/api/v1/auth/oidc/callback", deps.OIDC.Callback)
		}

		r.Group(func(r chi.Router) {
			r.Use(handlers.RequireSession(deps.Pool, deps.Logger))
			r.Post("/api/v1/logout", logout.ServeHTTP)
			r.Get("/api/v1/me", me.ServeHTTP)

			r.Route("/api/v1/teams/{teamID}", func(r chi.Router) {
				r.Use(handlers.RequireTeamAccess(deps.Pool, deps.Logger))
				r.Get("/", team.ServeHTTP)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(handlers.WithRequestMiddleware)
			r.Use(handlers.AttachSession(deps.Pool, deps.Logger))
			r.Handle("/graphql", graphql.NewHandler(deps.Pool, deps.Logger, deps.Jobs, deps.GraphQL))
		})
	}

	deps.Logger.Info("router initialized")
	return r
}

// NewOIDCServices builds OIDC route handlers when configuration is present.
func NewOIDCServices(pool *pgxpool.Pool, logger *slog.Logger, oidcCfg *config.OIDCConfig, provider handlers.OIDCAuthenticator) *OIDCServices {
	if oidcCfg == nil || provider == nil {
		return nil
	}

	handler := handlers.NewOIDCHandler(pool, logger, provider, oidcCfg.SuccessURL)
	return &OIDCServices{
		Handler:  handler,
		Login:    handler.Login,
		Callback: handler.Callback,
	}
}
