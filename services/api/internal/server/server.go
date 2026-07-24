package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

// Dependencies holds runtime services wired into the HTTP router.
type Dependencies struct {
	Logger *slog.Logger
	Pool   *pgxpool.Pool
}

// New returns an HTTP server with health, readiness, and API routes.
func New(deps Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

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

		r.Post("/api/v1/setup", setup.ServeHTTP)
		r.Post("/api/v1/login", login.ServeHTTP)

		r.Group(func(r chi.Router) {
			r.Use(handlers.RequireSession(deps.Pool, deps.Logger))
			r.Post("/api/v1/logout", logout.ServeHTTP)
			r.Get("/api/v1/me", me.ServeHTTP)

			r.Route("/api/v1/teams/{teamID}", func(r chi.Router) {
				r.Use(handlers.RequireTeamAccess(deps.Pool, deps.Logger))
				r.Get("/", team.ServeHTTP)
			})
		})
	}

	deps.Logger.Info("router initialized")
	return r
}
