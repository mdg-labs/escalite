package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/engine/statuspageapi"
)

// PublicStatusPageConfig controls signed unsubscribe tokens for public status pages.
type PublicStatusPageConfig struct {
	SigningKey []byte
}

// PublicStatusPageHandler serves unauthenticated status page data.
type PublicStatusPageHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	cfg    PublicStatusPageConfig
}

// NewPublicStatusPageHandler returns a handler for public status page routes.
func NewPublicStatusPageHandler(pool *pgxpool.Pool, logger *slog.Logger, cfg PublicStatusPageConfig) *PublicStatusPageHandler {
	return &PublicStatusPageHandler{
		pool:   pool,
		logger: logger,
		cfg:    cfg,
	}
}

const publicStatusPageResolvedWindowDays = 30

type publicStatusPageResponse struct {
	Title              string                       `json:"title"`
	OverallStatus      string                       `json:"overallStatus"`
	Components         []publicStatusPageComponent  `json:"components"`
	Incidents          []publicStatusPageIncident   `json:"incidents"`
	ResolvedIncidents  []publicStatusPageIncident   `json:"resolvedIncidents"`
}

type publicStatusPageComponent struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Status      string  `json:"status"`
	Position    int     `json:"position"`
}

type publicStatusPageIncident struct {
	ID                   string                            `json:"id"`
	Title                string                            `json:"title"`
	Status               string                            `json:"status"`
	AffectedComponentIDs []string                          `json:"affectedComponentIds"`
	Updates              []publicStatusPageIncidentUpdate  `json:"updates"`
	CreatedAt            time.Time                         `json:"createdAt"`
	ResolvedAt           *time.Time                        `json:"resolvedAt,omitempty"`
}

type publicStatusPageIncidentUpdate struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type publicStatusPageSubscribeRequest struct {
	Email string `json:"email"`
}

type publicStatusPageSubscribeResponse struct {
	Subscribed bool `json:"subscribed"`
}

type publicStatusPageUnsubscribeRequest struct {
	Token string `json:"token"`
}

type publicStatusPageUnsubscribeResponse struct {
	Unsubscribed bool `json:"unsubscribed"`
}

// Get serves GET /api/v1/public/status/{slug}.
func (h *PublicStatusPageHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "slug")))
	if slug == "" {
		WriteAPIError(w, http.StatusNotFound, CodeNotFound, "not found")
		return
	}

	ctx := r.Context()
	queries := db.New(h.pool)

	page, err := queries.GetStatusPageBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteAPIError(w, http.StatusNotFound, CodeNotFound, "not found")
			return
		}
		h.logger.Error("load public status page failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	if page.FrameAncestorsCsp.Valid && strings.TrimSpace(page.FrameAncestorsCsp.String) != "" {
		w.Header().Set("Content-Security-Policy", "frame-ancestors "+strings.TrimSpace(page.FrameAncestorsCsp.String))
	}

	components, err := queries.ListPublicStatusPageComponents(ctx, db.ListPublicStatusPageComponentsParams{
		StatusPageID:   page.ID,
		OrganizationID: page.OrganizationID,
	})
	if err != nil {
		h.logger.Error("list public status page components failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	publicComponents := make([]publicStatusPageComponent, 0, len(components))
	overallStatus := "operational"
	for _, component := range components {
		var description *string
		if component.Description.Valid && strings.TrimSpace(component.Description.String) != "" {
			value := component.Description.String
			description = &value
		}
		publicComponents = append(publicComponents, publicStatusPageComponent{
			ID:          component.ID.String(),
			Name:        component.Name,
			Description: description,
			Status:      component.Status,
			Position:    int(component.Position),
		})
		overallStatus = worstComponentStatus(overallStatus, component.Status)
	}

	incidents, err := queries.ListPublicStatusPageIncidents(ctx, db.ListPublicStatusPageIncidentsParams{
		StatusPageID:   page.ID,
		OrganizationID: page.OrganizationID,
	})
	if err != nil {
		h.logger.Error("list public status page incidents failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	publicIncidents, err := h.buildPublicStatusPageIncidents(ctx, queries, page.OrganizationID, publicStatusPageIncidentSourcesFromActive(incidents))
	if err != nil {
		h.logger.Error("build public status page incidents failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	resolvedIncidents, err := queries.ListPublicStatusPageResolvedIncidents(ctx, db.ListPublicStatusPageResolvedIncidentsParams{
		StatusPageID:       page.ID,
		OrganizationID:     page.OrganizationID,
		ResolvedWindowDays: publicStatusPageResolvedWindowDays,
	})
	if err != nil {
		h.logger.Error("list public status page resolved incidents failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	publicResolvedIncidents, err := h.buildPublicStatusPageIncidents(ctx, queries, page.OrganizationID, publicStatusPageIncidentSourcesFromResolved(resolvedIncidents))
	if err != nil {
		h.logger.Error("build public status page resolved incidents failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(publicStatusPageResponse{
		Title:             page.Title,
		OverallStatus:     overallStatus,
		Components:        publicComponents,
		Incidents:         publicIncidents,
		ResolvedIncidents: publicResolvedIncidents,
	})
}

// Subscribe serves POST /api/v1/public/status/{slug}/subscribe.
func (h *PublicStatusPageHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	slug := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "slug")))
	if slug == "" {
		WriteAPIError(w, http.StatusNotFound, CodeNotFound, "not found")
		return
	}

	var payload publicStatusPageSubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid request body")
		return
	}

	email := strings.ToLower(strings.TrimSpace(payload.Email))
	if email == "" || !strings.Contains(email, "@") {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "valid email is required")
		return
	}

	ctx := r.Context()
	queries := db.New(h.pool)

	page, err := queries.GetStatusPageBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteAPIError(w, http.StatusNotFound, CodeNotFound, "not found")
			return
		}
		h.logger.Error("load public status page failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	if page.FrameAncestorsCsp.Valid && strings.TrimSpace(page.FrameAncestorsCsp.String) != "" {
		w.Header().Set("Content-Security-Policy", "frame-ancestors "+strings.TrimSpace(page.FrameAncestorsCsp.String))
	}

	if _, err := queries.CreateStatusPageSubscription(ctx, db.CreateStatusPageSubscriptionParams{
		ID:             uuid.Must(uuid.NewV7()),
		StatusPageID:   page.ID,
		OrganizationID: page.OrganizationID,
		Email:          email,
	}); err != nil {
		h.logger.Error("create status page subscription failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(publicStatusPageSubscribeResponse{Subscribed: true})
}

// Unsubscribe serves POST /api/v1/public/status/unsubscribe.
func (h *PublicStatusPageHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	var payload publicStatusPageUnsubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid request body")
		return
	}

	token := strings.TrimSpace(payload.Token)
	if token == "" {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "token is required")
		return
	}

	claims, err := statuspageapi.ParseUnsubscribeToken(h.cfg.SigningKey, token)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid unsubscribe token")
		return
	}

	ctx := r.Context()
	queries := db.New(h.pool)

	if _, err := queries.UnsubscribeStatusPageSubscription(ctx, db.UnsubscribeStatusPageSubscriptionParams{
		ID:             claims.SubscriptionID,
		OrganizationID: claims.OrganizationID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid unsubscribe token")
			return
		}
		h.logger.Error("unsubscribe status page subscription failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(publicStatusPageUnsubscribeResponse{Unsubscribed: true})
}

type publicStatusPageIncidentSource struct {
	ID         uuid.UUID
	Title      string
	Status     string
	ResolvedAt pgtype.Timestamptz
	CreatedAt  pgtype.Timestamptz
}

func publicStatusPageIncidentSourcesFromActive(rows []db.ListPublicStatusPageIncidentsRow) []publicStatusPageIncidentSource {
	sources := make([]publicStatusPageIncidentSource, 0, len(rows))
	for _, row := range rows {
		sources = append(sources, publicStatusPageIncidentSource{
			ID:         row.ID,
			Title:      row.Title,
			Status:     row.Status,
			ResolvedAt: row.ResolvedAt,
			CreatedAt:  row.CreatedAt,
		})
	}
	return sources
}

func publicStatusPageIncidentSourcesFromResolved(rows []db.ListPublicStatusPageResolvedIncidentsRow) []publicStatusPageIncidentSource {
	sources := make([]publicStatusPageIncidentSource, 0, len(rows))
	for _, row := range rows {
		sources = append(sources, publicStatusPageIncidentSource{
			ID:         row.ID,
			Title:      row.Title,
			Status:     row.Status,
			ResolvedAt: row.ResolvedAt,
			CreatedAt:  row.CreatedAt,
		})
	}
	return sources
}

func (h *PublicStatusPageHandler) buildPublicStatusPageIncidents(
	ctx context.Context,
	queries *db.Queries,
	orgID uuid.UUID,
	sources []publicStatusPageIncidentSource,
) ([]publicStatusPageIncident, error) {
	publicIncidents := make([]publicStatusPageIncident, 0, len(sources))
	for _, incident := range sources {
		componentIDs, err := queries.ListStatusPageIncidentComponentIDs(ctx, db.ListStatusPageIncidentComponentIDsParams{
			StatusPageIncidentID: incident.ID,
			OrganizationID:       orgID,
		})
		if err != nil {
			return nil, err
		}

		affected := make([]string, 0, len(componentIDs))
		for _, id := range componentIDs {
			affected = append(affected, id.String())
		}

		updates, err := queries.ListPublicStatusPageIncidentUpdates(ctx, db.ListPublicStatusPageIncidentUpdatesParams{
			StatusPageIncidentID: incident.ID,
			OrganizationID:       orgID,
		})
		if err != nil {
			return nil, err
		}

		publicUpdates := make([]publicStatusPageIncidentUpdate, 0, len(updates))
		for _, update := range updates {
			publicUpdates = append(publicUpdates, publicStatusPageIncidentUpdate{
				ID:        update.ID.String(),
				Body:      update.Body,
				Status:    update.Status,
				CreatedAt: update.CreatedAt.Time.UTC(),
			})
		}

		var resolvedAt *time.Time
		if incident.ResolvedAt.Valid {
			t := incident.ResolvedAt.Time.UTC()
			resolvedAt = &t
		}

		publicIncidents = append(publicIncidents, publicStatusPageIncident{
			ID:                   incident.ID.String(),
			Title:                incident.Title,
			Status:               incident.Status,
			AffectedComponentIDs: affected,
			Updates:              publicUpdates,
			CreatedAt:            incident.CreatedAt.Time.UTC(),
			ResolvedAt:           resolvedAt,
		})
	}
	return publicIncidents, nil
}

func worstComponentStatus(current, candidate string) string {
	rank := func(status string) int {
		switch strings.ToLower(strings.TrimSpace(status)) {
		case "major_outage":
			return 4
		case "partial_outage":
			return 3
		case "degraded":
			return 2
		default:
			return 1
		}
	}

	if rank(candidate) > rank(current) {
		return strings.ToLower(strings.TrimSpace(candidate))
	}
	return current
}
