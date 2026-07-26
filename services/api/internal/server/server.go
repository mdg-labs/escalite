package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/config"
	"github.com/mdg-labs/escalite/services/api/internal/cors"
	"github.com/mdg-labs/escalite/services/api/internal/crypto"
	"github.com/mdg-labs/escalite/services/api/internal/email"
	"github.com/mdg-labs/escalite/services/api/internal/graphql"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	"github.com/mdg-labs/escalite/services/api/internal/queue"
	"github.com/mdg-labs/escalite/services/api/internal/ratelimit"
	"github.com/mdg-labs/escalite/services/api/internal/realtime"
)

// InboundEmailOptions overrides inbound email wiring (primarily for tests).
type InboundEmailOptions struct {
	RelaySecret          string
	Domain               string
	RequireAuthenticated bool
}

// Dependencies holds runtime services wired into the HTTP router.
type Dependencies struct {
	Logger         *slog.Logger
	Pool           *pgxpool.Pool
	Jobs           *queue.Producer
	Secrets        *crypto.Box
	OIDC           *OIDCServices
	SAML           *SAMLServices
	SlackOAuth     *SlackOAuthServices
	Mail           email.Sender
	PublicURL      string
	AppOrigin      string
	PasswordReset  *PasswordResetOptions
	HeartbeatPing  *HeartbeatPingOptions
	InboundWebhook   *InboundWebhookOptions
	InboundEmail     *InboundEmailOptions
	SlackInteractive *SlackInteractiveOptions
	GraphQL          graphql.Options
	Realtime       *realtime.Hub
}

// PasswordResetOptions overrides password reset wiring (primarily for tests).
type PasswordResetOptions struct {
	EmailLimiter ratelimit.Limiter
	IPLimiter    ratelimit.Limiter
	TokenTTL     time.Duration
}

// HeartbeatPingOptions overrides heartbeat ping wiring (primarily for tests).
type HeartbeatPingOptions struct {
	TokenLimiter ratelimit.Limiter
}

// InboundWebhookOptions overrides inbound webhook wiring (primarily for tests).
type InboundWebhookOptions struct {
	KeyLimiter ratelimit.Limiter
}

// OIDCServices holds optional OIDC login handlers when ESCALITE_OIDC_* is configured.
type OIDCServices struct {
	Handler  *handlers.OIDCHandler
	Login    http.HandlerFunc
	Callback http.HandlerFunc
}

// SAMLServices holds optional SAML SSO handlers when organization SAML is enabled.
type SAMLServices struct {
	Handler  *handlers.SAMLHandler
	Login    http.HandlerFunc
	ACS      http.HandlerFunc
	Metadata http.HandlerFunc
}

// SlackOAuthServices holds optional Slack app OAuth install handlers when
// ESCALITE_SLACK_CLIENT_ID/ESCALITE_SLACK_CLIENT_SECRET are configured.
type SlackOAuthServices struct {
	Handler  *handlers.SlackOAuthHandler
	Install  http.HandlerFunc
	Callback http.HandlerFunc
}

// SlackInteractiveOptions overrides Slack interactive endpoint wiring.
type SlackInteractiveOptions struct {
	SigningSecret string
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

		heartbeatCfg := handlers.HeartbeatPingConfig{}
		if deps.HeartbeatPing != nil {
			heartbeatCfg.TokenLimiter = deps.HeartbeatPing.TokenLimiter
		}
		heartbeatPing := handlers.NewHeartbeatPingHandler(deps.Pool, deps.Logger, heartbeatCfg)

		webhookCfg := handlers.InboundWebhookConfig{}
		if deps.InboundWebhook != nil {
			webhookCfg.KeyLimiter = deps.InboundWebhook.KeyLimiter
		}
		inboundWebhook := handlers.NewInboundWebhookHandler(deps.Pool, deps.Jobs, deps.Logger, webhookCfg)
		inboundAlerts := handlers.NewInboundAlertsHandler(deps.Pool, deps.Jobs, deps.Logger)

		emailCfg := handlers.InboundEmailConfig{}
		if deps.InboundEmail != nil {
			emailCfg = handlers.InboundEmailConfig{
				RelaySecret:          deps.InboundEmail.RelaySecret,
				Domain:               deps.InboundEmail.Domain,
				RequireAuthenticated: deps.InboundEmail.RequireAuthenticated,
			}
		}
		inboundEmail := handlers.NewInboundEmailHandler(deps.Pool, deps.Jobs, deps.Logger, emailCfg)

		slackInteractiveCfg := handlers.SlackInteractiveConfig{}
		if deps.SlackInteractive != nil {
			slackInteractiveCfg.SigningSecret = deps.SlackInteractive.SigningSecret
		}
		slackInteractive := handlers.NewSlackInteractiveHandler(deps.Pool, deps.Jobs, deps.Logger, slackInteractiveCfg)

		mobileAuth := handlers.NewMobileAuthHandler(deps.Pool, deps.Logger)

		r.Post("/api/v1/setup", setup.ServeHTTP)
		r.Post("/api/v1/login", login.ServeHTTP)
		r.Post("/api/v1/password-reset/request", passwordReset.Request)
		r.Post("/api/v1/password-reset/confirm", passwordReset.Confirm)

		r.Route("/heartbeat", func(r chi.Router) {
			r.Get("/{token}", heartbeatPing.ServeHTTP)
			r.Post("/{token}", heartbeatPing.ServeHTTP)
		})

		r.Post("/webhook/{plugin}/{token}", inboundWebhook.ServeHTTP)
		r.Post("/api/v1/alerts", inboundAlerts.ServeHTTP)
		if emailCfg.Enabled() {
			r.Post("/api/v1/inbound/email", inboundEmail.ServeHTTP)
		}
		if slackInteractiveCfg.SigningSecret != "" {
			r.Post("/api/v1/integrations/slack/interactive", slackInteractive.ServeHTTP)
		}

		r.Post("/api/v1/mobile/auth/exchange", mobileAuth.Exchange)
		r.Post("/api/v1/mobile/auth/refresh", mobileAuth.Refresh)

		if deps.OIDC != nil {
			r.Get("/api/v1/auth/oidc/login", deps.OIDC.Login)
			r.Get("/api/v1/auth/oidc/callback", deps.OIDC.Callback)
		}

		if deps.SAML != nil {
			r.Get("/api/v1/auth/saml/login", deps.SAML.Login)
			r.Post("/api/v1/auth/saml/acs", deps.SAML.ACS)
			r.Get("/api/v1/auth/saml/metadata", deps.SAML.Metadata)
		}

		scimHandler := handlers.NewSCIMHandler(deps.Pool, deps.Logger, deps.PublicURL)
		r.Route("/scim/v2", func(r chi.Router) {
			r.Use(scimHandler.Middleware)
			r.Get("/ServiceProviderConfig", scimHandler.ServiceProviderConfig)
			r.Route("/Users", func(r chi.Router) {
				r.Get("/", scimHandler.Users)
				r.Post("/", scimHandler.Users)
				r.Get("/{id}", scimHandler.UserByID)
				r.Put("/{id}", scimHandler.UserByID)
				r.Patch("/{id}", scimHandler.UserByID)
				r.Delete("/{id}", scimHandler.UserByID)
			})
			r.Route("/Groups", func(r chi.Router) {
				r.Get("/", scimHandler.Groups)
				r.Post("/", scimHandler.Groups)
				r.Get("/{id}", scimHandler.GroupByID)
				r.Put("/{id}", scimHandler.GroupByID)
				r.Patch("/{id}", scimHandler.GroupByID)
				r.Delete("/{id}", scimHandler.GroupByID)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(handlers.RequireSession(deps.Pool, deps.Logger))
			r.Post("/api/v1/mobile/auth/code", mobileAuth.IssueCode)
			r.Post("/api/v1/logout", logout.ServeHTTP)
			r.Get("/api/v1/me", me.ServeHTTP)

			scheduleICal := handlers.NewScheduleICalHandler(deps.Pool, deps.Logger)
			r.Get("/api/v1/schedules/{scheduleID}/calendar.ics", scheduleICal.ServeHTTP)

			r.Route("/api/v1/teams/{teamID}", func(r chi.Router) {
				r.Use(handlers.RequireTeamAccess(deps.Pool, deps.Logger))
				r.Get("/", team.ServeHTTP)
			})

			if deps.SlackOAuth != nil {
				r.Get("/api/v1/integrations/slack/install", deps.SlackOAuth.Install)
				r.Get("/api/v1/integrations/slack/callback", deps.SlackOAuth.Callback)
			}
		})

		r.Group(func(r chi.Router) {
			r.Use(handlers.WithRequestMiddleware)
			r.Use(handlers.AttachSession(deps.Pool, deps.Logger))
			graphqlOpts := deps.GraphQL
			if deps.SlackOAuth != nil {
				graphqlOpts.SlackOAuthInstallURL = SlackOAuthInstallURL(deps.PublicURL)
			}
			r.Handle("/graphql", graphql.NewHandler(deps.Pool, deps.Logger, deps.Jobs, deps.Secrets, deps.Realtime, graphqlOpts))
		})
	}

	deps.Logger.Info("router initialized")
	return r
}

// NewSAMLServices builds SAML route handlers backed by organization settings.
func NewSAMLServices(pool *pgxpool.Pool, logger *slog.Logger, secrets *crypto.Box, publicURL, successURL string) *SAMLServices {
	if secrets == nil || publicURL == "" {
		return nil
	}

	handler := handlers.NewSAMLHandler(pool, logger, secrets, publicURL, successURL)
	return &SAMLServices{
		Handler:  handler,
		Login:    handler.Login,
		ACS:      handler.ACS,
		Metadata: handler.Metadata,
	}
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

// SlackOAuthInstallURL returns the REST install endpoint when Slack OAuth is enabled.
func SlackOAuthInstallURL(publicURL string) string {
	if publicURL == "" {
		return "/api/v1/integrations/slack/install"
	}
	return strings.TrimSuffix(publicURL, "/") + "/api/v1/integrations/slack/install"
}
func NewSlackOAuthServices(pool *pgxpool.Pool, logger *slog.Logger, secrets *crypto.Box, slackCfg *config.SlackOAuthConfig) *SlackOAuthServices {
	if slackCfg == nil {
		return nil
	}

	provider := handlers.NewSlackOAuthClient(slackCfg.ClientID, slackCfg.ClientSecret, slackCfg.RedirectURL, slackCfg.Scopes)
	handler := handlers.NewSlackOAuthHandler(pool, logger, provider, secrets, slackCfg.SuccessURL)
	return &SlackOAuthServices{
		Handler:  handler,
		Install:  handler.Install,
		Callback: handler.Callback,
	}
}
