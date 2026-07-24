package handlers

import (
	"encoding/json"
	"net/http"
)

const (
	CodeUnauthenticated  = "UNAUTHENTICATED"
	CodeValidation       = "VALIDATION"
	CodeForbidden        = "FORBIDDEN"
	CodeInternal         = "INTERNAL"
	CodeMethodNotAllowed = "METHOD_NOT_ALLOWED"
)

type apiError struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// WriteAPIError writes a JSON error response with a GraphQL-equivalent error code.
func WriteAPIError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiError{
		Error: message,
		Code:  code,
	})
}
