package graph

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func validatePositiveDurationSeconds(value int, fieldName string) error {
	if value <= 0 {
		return gqlerr.New(handlers.CodeValidation, fieldName+" must be positive")
	}
	return nil
}

func isHeartbeatDurationCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		return false
	}
	return strings.Contains(pgErr.ConstraintName, "interval_seconds") ||
		strings.Contains(pgErr.ConstraintName, "grace_seconds")
}
