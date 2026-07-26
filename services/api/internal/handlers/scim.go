package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/authz"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	escalitescim "github.com/mdg-labs/escalite/services/api/internal/scim"
)

// SCIMHandler serves SCIM 2.0 provisioning endpoints.
type SCIMHandler struct {
	pool      *pgxpool.Pool
	logger    *slog.Logger
	publicURL string
	audit     *audit.Recorder
}

// NewSCIMHandler returns a handler for SCIM provisioning routes.
func NewSCIMHandler(pool *pgxpool.Pool, logger *slog.Logger, publicURL string) *SCIMHandler {
	return &SCIMHandler{
		pool:      pool,
		logger:    logger,
		publicURL: strings.TrimSuffix(strings.TrimSpace(publicURL), "/"),
		audit:     audit.NewRecorder(logger),
	}
}

type scimContext struct {
	organizationID uuid.UUID
}

type scimContextKey struct{}

func withScimContext(ctx context.Context, orgID uuid.UUID) context.Context {
	return context.WithValue(ctx, scimContextKey{}, scimContext{organizationID: orgID})
}

func scimContextFrom(ctx context.Context) (scimContext, bool) {
	value, ok := ctx.Value(scimContextKey{}).(scimContext)
	return value, ok
}

// Middleware authenticates SCIM bearer tokens and scopes requests to the token's organization.
func (h *SCIMHandler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := auth.ParseBearerToken(r.Header.Get("Authorization"))
		if !ok {
			h.writeError(w, http.StatusUnauthorized, "invalid_token", "authorization required")
			return
		}

		queries := db.New(h.pool)
		settings, err := queries.GetOrganizationScimSettingsByTokenHash(r.Context(), auth.HashScimBearerToken(bearer))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				h.writeError(w, http.StatusUnauthorized, "invalid_token", "invalid bearer token")
				return
			}
			h.logger.Error("load scim settings failed", "error", err)
			h.writeError(w, http.StatusInternalServerError, "", "internal error")
			return
		}

		next.ServeHTTP(w, r.WithContext(withScimContext(r.Context(), settings.OrganizationID)))
	})
}

// ServiceProviderConfig returns SCIM service provider capabilities.
func (h *SCIMHandler) ServiceProviderConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "", "method not allowed")
		return
	}

	h.writeJSON(w, http.StatusOK, escalitescim.ServiceProviderConfig{
		Schemas: []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		Patch:   map[string]bool{"supported": true},
		Bulk:    map[string]any{"supported": false, "maxOperations": 0, "maxPayloadSize": 0},
		Filter:  map[string]any{"supported": true, "maxResults": 200},
		ChangePass: map[string]bool{"supported": false},
		Sort:    map[string]bool{"supported": false},
		ETag:    map[string]bool{"supported": false},
		AuthSchemes: []map[string]string{
			{
				"type":        "oauthbearertoken",
				"name":        "OAuth Bearer Token",
				"description": "Authentication scheme using the OAuth Bearer Token Standard",
			},
		},
	})
}

// Users routes SCIM /Users requests.
func (h *SCIMHandler) Users(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listUsers(w, r)
	case http.MethodPost:
		h.createUser(w, r)
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "", "method not allowed")
	}
}

// UserByID routes SCIM /Users/{id} requests.
func (h *SCIMHandler) UserByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getUser(w, r)
	case http.MethodPut:
		h.replaceUser(w, r)
	case http.MethodPatch:
		h.patchUser(w, r)
	case http.MethodDelete:
		h.deleteUser(w, r)
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "", "method not allowed")
	}
}

// Groups routes SCIM /Groups requests.
func (h *SCIMHandler) Groups(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listGroups(w, r)
	case http.MethodPost:
		h.createGroup(w, r)
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "", "method not allowed")
	}
}

// GroupByID routes SCIM /Groups/{id} requests.
func (h *SCIMHandler) GroupByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getGroup(w, r)
	case http.MethodPut:
		h.replaceGroup(w, r)
	case http.MethodPatch:
		h.patchGroup(w, r)
	case http.MethodDelete:
		h.deleteGroup(w, r)
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "", "method not allowed")
	}
}

func (h *SCIMHandler) listUsers(w http.ResponseWriter, r *http.Request) {
	sc, ok := scimContextFrom(r.Context())
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	queries := db.New(h.pool)
	users, err := queries.ListScimUsersByOrganizationID(r.Context(), sc.organizationID)
	if err != nil {
		h.logger.Error("list scim users failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	if filter := strings.TrimSpace(r.URL.Query().Get("filter")); filter != "" {
		if userName, ok := escalitescim.ParseUserNameFilter(filter); ok {
			filtered := make([]db.User, 0, 1)
			for _, user := range users {
				if strings.EqualFold(user.Email, userName) {
					filtered = append(filtered, user)
				}
			}
			users = filtered
		}
	}

	resources := make([]escalitescim.User, 0, len(users))
	for _, user := range users {
		resources = append(resources, userToSCIM(user))
	}

	h.writeJSON(w, http.StatusOK, escalitescim.ListResponse{
		Schemas:      []string{escalitescim.SchemaList},
		TotalResults: len(resources),
		StartIndex:   1,
		ItemsPerPage: len(resources),
		Resources:    resources,
	})
}

func (h *SCIMHandler) createUser(w http.ResponseWriter, r *http.Request) {
	sc, ok := scimContextFrom(r.Context())
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	var payload escalitescim.User
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalidSyntax", "invalid JSON body")
		return
	}

	email, err := scimUserEmail(payload)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalidValue", err.Error())
		return
	}

	queries := db.New(h.pool)
	existing, err := queries.GetUserByEmail(r.Context(), db.GetUserByEmailParams{
		OrganizationID: sc.organizationID,
		Email:          email,
	})
	if err == nil {
		if existing.DeprovisionedAt.Valid {
			user, err := h.reprovisionUser(r.Context(), queries, sc.organizationID, existing, email, payload.ExternalID)
			if err != nil {
				h.logger.Error("reprovision scim user failed", "error", err)
				h.writeError(w, http.StatusInternalServerError, "", "internal error")
				return
			}
			h.writeJSON(w, http.StatusOK, userToSCIM(user))
			return
		}
		h.writeError(w, http.StatusConflict, "uniqueness", "user already exists")
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		h.logger.Error("lookup scim user failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	userID := uuid.Must(uuid.NewV7())
	externalID := pgtype.Text{}
	if payload.ExternalID != "" {
		externalID = pgtype.Text{String: payload.ExternalID, Valid: true}
	}

	account, err := auth.FindOrCreateAccountByEmail(r.Context(), queries, email)
	if err != nil {
		h.logger.Error("resolve scim account failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	user, err := queries.CreateScimUser(r.Context(), db.CreateScimUserParams{
		ID:             userID,
		AccountID:      account.ID,
		OrganizationID: sc.organizationID,
		Email:          email,
		Role:           authz.RoleMember,
		ScimExternalID: externalID,
	})
	if err != nil {
		h.logger.Error("create scim user failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	h.audit.ScimUserProvisioned(r.Context(), queries, sc.organizationID, user.ID)
	h.writeJSON(w, http.StatusCreated, userToSCIM(user))
}

func (h *SCIMHandler) getUser(w http.ResponseWriter, r *http.Request) {
	sc, ok := scimContextFrom(r.Context())
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalidValue", "invalid user id")
		return
	}

	queries := db.New(h.pool)
	user, err := queries.GetUserByIDIncludingDeprovisioned(r.Context(), db.GetUserByIDIncludingDeprovisionedParams{
		ID:             userID,
		OrganizationID: sc.organizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "", "user not found")
			return
		}
		h.logger.Error("load scim user failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}
	if !user.ScimExternalID.Valid {
		h.writeError(w, http.StatusNotFound, "", "user not found")
		return
	}

	h.writeJSON(w, http.StatusOK, userToSCIM(user))
}

func (h *SCIMHandler) replaceUser(w http.ResponseWriter, r *http.Request) {
	sc, ok := scimContextFrom(r.Context())
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalidValue", "invalid user id")
		return
	}

	var payload escalitescim.User
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalidSyntax", "invalid JSON body")
		return
	}

	email, err := scimUserEmail(payload)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalidValue", err.Error())
		return
	}

	queries := db.New(h.pool)
	existing, err := queries.GetUserByIDIncludingDeprovisioned(r.Context(), db.GetUserByIDIncludingDeprovisionedParams{
		ID:             userID,
		OrganizationID: sc.organizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "", "user not found")
			return
		}
		h.logger.Error("load scim user failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}
	if !existing.ScimExternalID.Valid {
		h.writeError(w, http.StatusNotFound, "", "user not found")
		return
	}

	externalID := existing.ScimExternalID
	if payload.ExternalID != "" {
		externalID = pgtype.Text{String: payload.ExternalID, Valid: true}
	}

	deprovisionedAt := pgtype.Timestamptz{}
	if !payload.Active {
		if existing.DeprovisionedAt.Valid {
			deprovisionedAt = existing.DeprovisionedAt
		} else {
			user, err := h.deprovisionUser(r.Context(), queries, sc.organizationID, existing.ID)
			if err != nil {
				h.logger.Error("deprovision scim user failed", "error", err)
				h.writeError(w, http.StatusInternalServerError, "", "internal error")
				return
			}
			h.writeJSON(w, http.StatusOK, userToSCIM(user))
			return
		}
	}

	user, err := queries.UpdateScimUser(r.Context(), db.UpdateScimUserParams{
		ID:             userID,
		OrganizationID: sc.organizationID,
		Email:          email,
		ScimExternalID: externalID,
		DeprovisionedAt: deprovisionedAt,
	})
	if err != nil {
		h.logger.Error("update scim user failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	h.writeJSON(w, http.StatusOK, userToSCIM(user))
}

func (h *SCIMHandler) patchUser(w http.ResponseWriter, r *http.Request) {
	sc, ok := scimContextFrom(r.Context())
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalidValue", "invalid user id")
		return
	}

	var payload escalitescim.PatchRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalidSyntax", "invalid JSON body")
		return
	}

	queries := db.New(h.pool)
	existing, err := queries.GetUserByIDIncludingDeprovisioned(r.Context(), db.GetUserByIDIncludingDeprovisionedParams{
		ID:             userID,
		OrganizationID: sc.organizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "", "user not found")
			return
		}
		h.logger.Error("load scim user failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}
	if !existing.ScimExternalID.Valid {
		h.writeError(w, http.StatusNotFound, "", "user not found")
		return
	}

	for _, operation := range payload.Operations {
		if !strings.EqualFold(operation.Op, "replace") {
			continue
		}
		if strings.EqualFold(operation.Path, "active") {
			active, ok := escalitescim.ActiveFromPatchValue(operation.Value)
			if !ok {
				continue
			}
			if !active && !existing.DeprovisionedAt.Valid {
				user, err := h.deprovisionUser(r.Context(), queries, sc.organizationID, existing.ID)
				if err != nil {
					h.logger.Error("deprovision scim user failed", "error", err)
					h.writeError(w, http.StatusInternalServerError, "", "internal error")
					return
				}
				h.writeJSON(w, http.StatusOK, userToSCIM(user))
				return
			}
		}
	}

	h.writeJSON(w, http.StatusOK, userToSCIM(existing))
}

func (h *SCIMHandler) deleteUser(w http.ResponseWriter, r *http.Request) {
	sc, ok := scimContextFrom(r.Context())
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalidValue", "invalid user id")
		return
	}

	queries := db.New(h.pool)
	existing, err := queries.GetUserByIDIncludingDeprovisioned(r.Context(), db.GetUserByIDIncludingDeprovisionedParams{
		ID:             userID,
		OrganizationID: sc.organizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "", "user not found")
			return
		}
		h.logger.Error("load scim user failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}
	if !existing.ScimExternalID.Valid {
		h.writeError(w, http.StatusNotFound, "", "user not found")
		return
	}

	if !existing.DeprovisionedAt.Valid {
		if _, err := h.deprovisionUser(r.Context(), queries, sc.organizationID, existing.ID); err != nil {
			h.logger.Error("deprovision scim user failed", "error", err)
			h.writeError(w, http.StatusInternalServerError, "", "internal error")
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SCIMHandler) listGroups(w http.ResponseWriter, r *http.Request) {
	sc, ok := scimContextFrom(r.Context())
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	queries := db.New(h.pool)
	groups, err := queries.ListScimGroupsByOrganizationID(r.Context(), sc.organizationID)
	if err != nil {
		h.logger.Error("list scim groups failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	if filter := strings.TrimSpace(r.URL.Query().Get("filter")); filter != "" {
		if displayName, ok := escalitescim.ParseDisplayNameFilter(filter); ok {
			filtered := make([]db.ScimGroup, 0, 1)
			for _, group := range groups {
				if strings.EqualFold(group.DisplayName, displayName) {
					filtered = append(filtered, group)
				}
			}
			groups = filtered
		}
	}

	resources := make([]escalitescim.Group, 0, len(groups))
	for _, group := range groups {
		resource, err := h.groupToSCIM(r.Context(), queries, group)
		if err != nil {
			h.logger.Error("build scim group failed", "error", err)
			h.writeError(w, http.StatusInternalServerError, "", "internal error")
			return
		}
		resources = append(resources, resource)
	}

	h.writeJSON(w, http.StatusOK, escalitescim.ListResponse{
		Schemas:      []string{escalitescim.SchemaList},
		TotalResults: len(resources),
		StartIndex:   1,
		ItemsPerPage: len(resources),
		Resources:    resources,
	})
}

func (h *SCIMHandler) createGroup(w http.ResponseWriter, r *http.Request) {
	sc, ok := scimContextFrom(r.Context())
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	var payload escalitescim.Group
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalidSyntax", "invalid JSON body")
		return
	}

	displayName := strings.TrimSpace(payload.DisplayName)
	if displayName == "" {
		h.writeError(w, http.StatusBadRequest, "invalidValue", "displayName is required")
		return
	}

	externalID := strings.TrimSpace(payload.ExternalID)
	if externalID == "" {
		externalID = displayName
	}

	queries := db.New(h.pool)
	if existing, err := queries.GetScimGroupByExternalID(r.Context(), db.GetScimGroupByExternalIDParams{
		OrganizationID: sc.organizationID,
		ExternalID:     externalID,
	}); err == nil {
		resource, err := h.groupToSCIM(r.Context(), queries, existing)
		if err != nil {
			h.logger.Error("build scim group failed", "error", err)
			h.writeError(w, http.StatusInternalServerError, "", "internal error")
			return
		}
		h.writeJSON(w, http.StatusOK, resource)
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		h.logger.Error("lookup scim group failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	group, err := h.createGroupWithTeam(r.Context(), queries, sc.organizationID, externalID, displayName)
	if err != nil {
		h.logger.Error("create scim group failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	if err := h.syncGroupMembers(r.Context(), queries, sc.organizationID, group, payload.Members); err != nil {
		h.logger.Error("sync scim group members failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	resource, err := h.groupToSCIM(r.Context(), queries, group)
	if err != nil {
		h.logger.Error("build scim group failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	h.writeJSON(w, http.StatusCreated, resource)
}

func (h *SCIMHandler) getGroup(w http.ResponseWriter, r *http.Request) {
	sc, ok := scimContextFrom(r.Context())
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalidValue", "invalid group id")
		return
	}

	queries := db.New(h.pool)
	group, err := queries.GetScimGroupByID(r.Context(), db.GetScimGroupByIDParams{
		ID:             groupID,
		OrganizationID: sc.organizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "", "group not found")
			return
		}
		h.logger.Error("load scim group failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	resource, err := h.groupToSCIM(r.Context(), queries, group)
	if err != nil {
		h.logger.Error("build scim group failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	h.writeJSON(w, http.StatusOK, resource)
}

func (h *SCIMHandler) replaceGroup(w http.ResponseWriter, r *http.Request) {
	sc, ok := scimContextFrom(r.Context())
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalidValue", "invalid group id")
		return
	}

	var payload escalitescim.Group
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalidSyntax", "invalid JSON body")
		return
	}

	displayName := strings.TrimSpace(payload.DisplayName)
	if displayName == "" {
		h.writeError(w, http.StatusBadRequest, "invalidValue", "displayName is required")
		return
	}

	queries := db.New(h.pool)
	group, err := queries.UpdateScimGroupDisplayName(r.Context(), db.UpdateScimGroupDisplayNameParams{
		ID:             groupID,
		OrganizationID: sc.organizationID,
		DisplayName:    displayName,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "", "group not found")
			return
		}
		h.logger.Error("update scim group failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	if err := h.syncGroupMembers(r.Context(), queries, sc.organizationID, group, payload.Members); err != nil {
		h.logger.Error("sync scim group members failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	resource, err := h.groupToSCIM(r.Context(), queries, group)
	if err != nil {
		h.logger.Error("build scim group failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	h.writeJSON(w, http.StatusOK, resource)
}

func (h *SCIMHandler) patchGroup(w http.ResponseWriter, r *http.Request) {
	sc, ok := scimContextFrom(r.Context())
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalidValue", "invalid group id")
		return
	}

	var payload escalitescim.PatchRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalidSyntax", "invalid JSON body")
		return
	}

	queries := db.New(h.pool)
	group, err := queries.GetScimGroupByID(r.Context(), db.GetScimGroupByIDParams{
		ID:             groupID,
		OrganizationID: sc.organizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeError(w, http.StatusNotFound, "", "group not found")
			return
		}
		h.logger.Error("load scim group failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	for _, operation := range payload.Operations {
		switch strings.ToLower(operation.Op) {
		case "add":
			if strings.EqualFold(operation.Path, "members") {
				members, err := membersFromPatchValue(operation.Value)
				if err != nil {
					h.writeError(w, http.StatusBadRequest, "invalidValue", err.Error())
					return
				}
				if err := h.addGroupMembers(r.Context(), queries, sc.organizationID, group, members); err != nil {
					h.logger.Error("add scim group members failed", "error", err)
					h.writeError(w, http.StatusInternalServerError, "", "internal error")
					return
				}
			}
		case "remove":
			if strings.HasPrefix(strings.ToLower(operation.Path), "members") {
				memberID, err := memberIDFromRemovePath(operation.Path)
				if err != nil {
					h.writeError(w, http.StatusBadRequest, "invalidValue", err.Error())
					return
				}
				if err := h.removeGroupMember(r.Context(), queries, sc.organizationID, group, memberID); err != nil {
					h.logger.Error("remove scim group member failed", "error", err)
					h.writeError(w, http.StatusInternalServerError, "", "internal error")
					return
				}
			}
		case "replace":
			if strings.EqualFold(operation.Path, "displayName") {
				displayName, ok := operation.Value.(string)
				if !ok || strings.TrimSpace(displayName) == "" {
					h.writeError(w, http.StatusBadRequest, "invalidValue", "displayName is required")
					return
				}
				group, err = queries.UpdateScimGroupDisplayName(r.Context(), db.UpdateScimGroupDisplayNameParams{
					ID:             groupID,
					OrganizationID: sc.organizationID,
					DisplayName:    strings.TrimSpace(displayName),
				})
				if err != nil {
					h.logger.Error("update scim group failed", "error", err)
					h.writeError(w, http.StatusInternalServerError, "", "internal error")
					return
				}
			}
		}
	}

	resource, err := h.groupToSCIM(r.Context(), queries, group)
	if err != nil {
		h.logger.Error("build scim group failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	h.writeJSON(w, http.StatusOK, resource)
}

func (h *SCIMHandler) deleteGroup(w http.ResponseWriter, r *http.Request) {
	sc, ok := scimContextFrom(r.Context())
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	groupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalidValue", "invalid group id")
		return
	}

	queries := db.New(h.pool)
	if err := queries.DeleteScimGroup(r.Context(), db.DeleteScimGroupParams{
		ID:             groupID,
		OrganizationID: sc.organizationID,
	}); err != nil {
		h.logger.Error("delete scim group failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "", "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SCIMHandler) deprovisionUser(ctx context.Context, queries db.Querier, orgID, userID uuid.UUID) (db.User, error) {
	user, err := queries.DeprovisionUser(ctx, db.DeprovisionUserParams{
		ID:             userID,
		OrganizationID: orgID,
	})
	if err != nil {
		return db.User{}, err
	}

	if err := queries.RevokeAllUserSessions(ctx, db.RevokeAllUserSessionsParams{
		UserID:         userID,
		OrganizationID: orgID,
	}); err != nil {
		return db.User{}, fmt.Errorf("revoke sessions: %w", err)
	}

	if err := queries.RevokeAllUserRefreshTokens(ctx, db.RevokeAllUserRefreshTokensParams{
		UserID:         userID,
		OrganizationID: orgID,
	}); err != nil {
		return db.User{}, fmt.Errorf("revoke refresh tokens: %w", err)
	}

	h.audit.ScimUserDeprovisioned(ctx, queries, orgID, userID)
	return user, nil
}

func (h *SCIMHandler) reprovisionUser(
	ctx context.Context,
	queries db.Querier,
	orgID uuid.UUID,
	existing db.User,
	email string,
	externalID string,
) (db.User, error) {
	ext := existing.ScimExternalID
	if externalID != "" {
		ext = pgtype.Text{String: externalID, Valid: true}
	}
	return queries.ReprovisionScimUser(ctx, db.ReprovisionScimUserParams{
		ID:             existing.ID,
		OrganizationID: orgID,
		Email:          email,
		ScimExternalID: ext,
	})
}

func (h *SCIMHandler) createGroupWithTeam(
	ctx context.Context,
	queries db.Querier,
	orgID uuid.UUID,
	externalID string,
	displayName string,
) (db.ScimGroup, error) {
	team, err := queries.GetTeamByName(ctx, db.GetTeamByNameParams{
		OrganizationID: orgID,
		Name:           displayName,
	})
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return db.ScimGroup{}, err
		}
		team, err = queries.CreateTeam(ctx, db.CreateTeamParams{
			ID:             uuid.Must(uuid.NewV7()),
			OrganizationID: orgID,
			Name:           displayName,
		})
		if err != nil {
			return db.ScimGroup{}, err
		}
	}

	return queries.CreateScimGroup(ctx, db.CreateScimGroupParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ExternalID:     externalID,
		DisplayName:    displayName,
		TeamID:         team.ID,
	})
}

func (h *SCIMHandler) syncGroupMembers(
	ctx context.Context,
	queries db.Querier,
	orgID uuid.UUID,
	group db.ScimGroup,
	members []escalitescim.Member,
) error {
	if err := queries.RemoveAllScimGroupMembers(ctx, db.RemoveAllScimGroupMembersParams{
		ScimGroupID:    group.ID,
		OrganizationID: orgID,
	}); err != nil {
		return err
	}
	return h.addGroupMembers(ctx, queries, orgID, group, members)
}

func (h *SCIMHandler) addGroupMembers(
	ctx context.Context,
	queries db.Querier,
	orgID uuid.UUID,
	group db.ScimGroup,
	members []escalitescim.Member,
) error {
	for _, member := range members {
		userID, err := uuid.Parse(strings.TrimSpace(member.Value))
		if err != nil {
			return fmt.Errorf("invalid member id: %w", err)
		}

		user, err := queries.GetUserByIDIncludingDeprovisioned(ctx, db.GetUserByIDIncludingDeprovisionedParams{
			ID:             userID,
			OrganizationID: orgID,
		})
		if err != nil {
			return err
		}
		if !auth.IsUserActive(user.DeprovisionedAt.Valid) {
			continue
		}

		if err := queries.AddScimGroupMember(ctx, db.AddScimGroupMemberParams{
			ScimGroupID:    group.ID,
			UserID:         userID,
			OrganizationID: orgID,
		}); err != nil {
			return err
		}

		hasMembership, err := queries.HasTeamMembership(ctx, db.HasTeamMembershipParams{
			TeamID:         group.TeamID,
			UserID:         userID,
			OrganizationID: orgID,
		})
		if err != nil {
			return err
		}
		if !hasMembership {
			_, err = queries.CreateTeamMembership(ctx, db.CreateTeamMembershipParams{
				ID:             uuid.Must(uuid.NewV7()),
				TeamID:         group.TeamID,
				UserID:         userID,
				OrganizationID: orgID,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *SCIMHandler) removeGroupMember(
	ctx context.Context,
	queries db.Querier,
	orgID uuid.UUID,
	group db.ScimGroup,
	userID uuid.UUID,
) error {
	if err := queries.RemoveScimGroupMember(ctx, db.RemoveScimGroupMemberParams{
		ScimGroupID:    group.ID,
		UserID:         userID,
		OrganizationID: orgID,
	}); err != nil {
		return err
	}

	return queries.DeleteTeamMembership(ctx, db.DeleteTeamMembershipParams{
		TeamID:         group.TeamID,
		UserID:         userID,
		OrganizationID: orgID,
	})
}

func (h *SCIMHandler) groupToSCIM(ctx context.Context, queries db.Querier, group db.ScimGroup) (escalitescim.Group, error) {
	userIDs, err := queries.ListScimGroupMemberUserIDs(ctx, db.ListScimGroupMemberUserIDsParams{
		ScimGroupID:    group.ID,
		OrganizationID: group.OrganizationID,
	})
	if err != nil {
		return escalitescim.Group{}, err
	}

	members := make([]escalitescim.Member, 0, len(userIDs))
	for _, userID := range userIDs {
		user, err := queries.GetUserByIDIncludingDeprovisioned(ctx, db.GetUserByIDIncludingDeprovisionedParams{
			ID:             userID,
			OrganizationID: group.OrganizationID,
		})
		if err != nil {
			return escalitescim.Group{}, err
		}
		members = append(members, escalitescim.Member{
			Value:   user.ID.String(),
			Display: user.Email,
		})
	}

	return escalitescim.Group{
		Schemas:     []string{escalitescim.SchemaGroup},
		ID:          group.ID.String(),
		ExternalID:  group.ExternalID,
		DisplayName: group.DisplayName,
		Members:     members,
	}, nil
}

func userToSCIM(user db.User) escalitescim.User {
	externalID := ""
	if user.ScimExternalID.Valid {
		externalID = user.ScimExternalID.String
	}

	return escalitescim.User{
		Schemas:    []string{escalitescim.SchemaUser},
		ID:         user.ID.String(),
		ExternalID: externalID,
		UserName:   user.Email,
		Name:       &escalitescim.Name{Formatted: user.Email},
		Emails:     []escalitescim.Email{{Value: user.Email, Primary: true}},
		Active:     auth.IsUserActive(user.DeprovisionedAt.Valid),
	}
}

func scimUserEmail(payload escalitescim.User) (string, error) {
	email := strings.ToLower(strings.TrimSpace(payload.UserName))
	if email == "" && len(payload.Emails) > 0 {
		email = strings.ToLower(strings.TrimSpace(payload.Emails[0].Value))
	}
	if email == "" {
		return "", fmt.Errorf("userName is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", fmt.Errorf("userName is invalid")
	}
	return email, nil
}

func membersFromPatchValue(value any) ([]escalitescim.Member, error) {
	switch typed := value.(type) {
	case []any:
		members := make([]escalitescim.Member, 0, len(typed))
		for _, item := range typed {
			userID, err := escalitescim.MemberUserID(item)
			if err != nil {
				return nil, err
			}
			members = append(members, escalitescim.Member{Value: userID})
		}
		return members, nil
	default:
		userID, err := escalitescim.MemberUserID(value)
		if err != nil {
			return nil, err
		}
		return []escalitescim.Member{{Value: userID}}, nil
	}
}

func memberIDFromRemovePath(path string) (uuid.UUID, error) {
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(strings.ToLower(path), "members") {
		return uuid.Nil, fmt.Errorf("unsupported remove path")
	}

	parts := strings.Split(path, "[")
	if len(parts) != 2 {
		return uuid.Nil, fmt.Errorf("invalid members path")
	}
	valuePart := strings.TrimSuffix(parts[1], "]")
	valuePart = strings.Trim(valuePart, `"'`)
	valuePart = strings.TrimPrefix(valuePart, "value eq ")
	valuePart = strings.Trim(valuePart, `"'`)
	return uuid.Parse(valuePart)
}

func (h *SCIMHandler) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", escalitescim.ContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *SCIMHandler) writeError(w http.ResponseWriter, status int, scimType, detail string) {
	if detail == "" {
		detail = http.StatusText(status)
	}
	h.writeJSON(w, status, escalitescim.ErrorResponse{
		Schemas:  []string{escalitescim.SchemaError},
		Detail:   detail,
		Status:   fmt.Sprintf("%d", status),
		ScimType: scimType,
	})
}
