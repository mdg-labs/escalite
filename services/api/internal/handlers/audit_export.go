package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/auditexport"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/authz"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

const maxAuditExportRows = 10_000

// AuditExportHandler serves CSV exports for organization audit events.
type AuditExportHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewAuditExportHandler returns a handler for GET /api/v1/audit-events/export.csv.
func NewAuditExportHandler(pool *pgxpool.Pool, logger *slog.Logger) *AuditExportHandler {
	return &AuditExportHandler{pool: pool, logger: logger}
}

func (h *AuditExportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	sc, ok := auth.SessionFromContext(r.Context())
	if !ok {
		WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "authentication required")
		return
	}
	if !authz.IsAdmin(sc.User.Role) {
		WriteAPIError(w, http.StatusForbidden, CodeForbidden, "admin access required")
		return
	}

	actionFilter, fromTime, toTime, err := parseAuditExportQuery(r)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, err.Error())
		return
	}

	csvData, err := h.exportAuditEvents(r.Context(), sc.User.OrganizationID, actionFilter, fromTime, toTime)
	if err != nil {
		h.logger.Error("export audit events failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="audit-events.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(csvData)
}

func (h *AuditExportHandler) exportAuditEvents(
	ctx context.Context,
	orgID uuid.UUID,
	actionFilter pgtype.Text,
	fromTime, toTime pgtype.Timestamptz,
) ([]byte, error) {
	queries := db.New(h.pool)

	events, err := queries.ListAuditEventsFiltered(ctx, db.ListAuditEventsFilteredParams{
		OrganizationID: orgID,
		ActionFilter:   actionFilter,
		FromTime:       fromTime,
		ToTime:         toTime,
		PageLimit:      maxAuditExportRows,
		PageOffset:     0,
	})
	if err != nil {
		return nil, err
	}

	actorEmails := make(map[uuid.UUID]string)
	for _, event := range events {
		if !event.ActorID.Valid {
			continue
		}
		actorID := uuid.UUID(event.ActorID.Bytes)
		if _, ok := actorEmails[actorID]; ok {
			continue
		}
		user, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
			ID:             actorID,
			OrganizationID: orgID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, err
		}
		actorEmails[actorID] = user.Email
	}

	rows, err := auditexport.RowsFromDB(events, actorEmails)
	if err != nil {
		return nil, err
	}

	return auditexport.Export(rows)
}

func parseAuditExportQuery(r *http.Request) (pgtype.Text, pgtype.Timestamptz, pgtype.Timestamptz, error) {
	query := r.URL.Query()

	var actionFilter pgtype.Text
	if action := strings.TrimSpace(query.Get("action")); action != "" {
		actionFilter = pgtype.Text{String: action, Valid: true}
	}

	fromTime, err := parseAuditExportDateTime(query.Get("from"), false)
	if err != nil {
		return pgtype.Text{}, pgtype.Timestamptz{}, pgtype.Timestamptz{}, err
	}

	toTime, err := parseAuditExportDateTime(query.Get("to"), true)
	if err != nil {
		return pgtype.Text{}, pgtype.Timestamptz{}, pgtype.Timestamptz{}, err
	}

	return actionFilter, fromTime, toTime, nil
}

func parseAuditExportDateTime(value string, endOfDay bool) (pgtype.Timestamptz, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return pgtype.Timestamptz{}, nil
	}

	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return pgtype.Timestamptz{Time: parsed.UTC(), Valid: true}, nil
	}

	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return pgtype.Timestamptz{}, errors.New("from and to must be RFC3339 timestamps or YYYY-MM-DD dates")
	}

	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}

	return pgtype.Timestamptz{Time: parsed.UTC(), Valid: true}, nil
}
