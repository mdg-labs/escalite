package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
)

type teamResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TeamHandler returns a single team after RBAC middleware authorizes access.
type TeamHandler struct{}

// NewTeamHandler returns a handler for GET /api/v1/teams/{teamID}.
func NewTeamHandler() *TeamHandler {
	return &TeamHandler{}
}

func (h *TeamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	team, ok := auth.TeamFromContext(r.Context())
	if !ok {
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(teamResponse{
		ID:   team.ID.String(),
		Name: team.Name,
	})
}
