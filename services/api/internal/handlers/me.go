package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
)

type meUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type meResponse struct {
	User meUser `json:"user"`
}

// MeHandler returns the authenticated user. Protected by session middleware.
type MeHandler struct{}

// NewMeHandler returns a handler for GET /api/v1/me.
func NewMeHandler() *MeHandler {
	return &MeHandler{}
}

func (h *MeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	sc, ok := auth.SessionFromContext(r.Context())
	if !ok {
		WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "authentication required")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(meResponse{
		User: meUser{
			ID:    sc.User.ID.String(),
			Email: sc.User.Email,
			Role:  sc.User.Role,
		},
	})
}
