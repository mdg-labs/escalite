package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/authz"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/schedexport"
)

// ScheduleICalHandler serves RFC 5545 calendar exports for schedules.
type ScheduleICalHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewScheduleICalHandler returns a handler for GET /api/v1/schedules/{scheduleID}/calendar.ics.
func NewScheduleICalHandler(pool *pgxpool.Pool, logger *slog.Logger) *ScheduleICalHandler {
	return &ScheduleICalHandler{pool: pool, logger: logger}
}

func (h *ScheduleICalHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleID"))
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "scheduleId is invalid")
		return
	}

	ics, err := h.exportSchedule(r.Context(), sc, scheduleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteAPIError(w, http.StatusNotFound, CodeNotFound, "schedule not found")
			return
		}
		h.logger.Error("export schedule calendar failed", "error", err, "scheduleId", scheduleID)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="schedule.ics"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(ics))
}

func (h *ScheduleICalHandler) exportSchedule(ctx context.Context, sc auth.SessionContext, scheduleID uuid.UUID) (string, error) {
	queries := db.New(h.pool)

	schedule, err := queries.GetScheduleByID(ctx, db.GetScheduleByIDParams{
		ID:             scheduleID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		return "", err
	}

	rotations, err := queries.ListRotationsByScheduleID(ctx, db.ListRotationsByScheduleIDParams{
		ScheduleID:     scheduleID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		return "", err
	}

	userEmails, err := loadRotationUserEmails(ctx, queries, sc.User.OrganizationID, rotations)
	if err != nil {
		return "", err
	}

	exportRotations := make([]schedexport.Rotation, 0, len(rotations))
	for _, rotation := range rotations {
		participantIDs, err := decodeParticipantIDs(rotation.Participants)
		if err != nil {
			return "", err
		}
		exportRotations = append(exportRotations, schedexport.Rotation{
			ID:             rotation.ID,
			Name:           rotation.Name,
			Layer:          int(rotation.Layer),
			RRule:          rotation.Rrule,
			Anchor:         timeFromDB(rotation.CreatedAt),
			ParticipantIDs: participantIDs,
		})
	}

	return schedexport.Export(schedexport.Input{
		ScheduleID:   schedule.ID,
		ScheduleName: schedule.Name,
		Timezone:     schedule.Timezone,
		Rotations:    exportRotations,
		UserEmails:   userEmails,
		ExportedAt:   time.Now().UTC(),
	})
}

func loadRotationUserEmails(ctx context.Context, queries *db.Queries, orgID uuid.UUID, rotations []db.Rotation) (map[string]string, error) {
	seen := make(map[uuid.UUID]struct{})
	for _, rotation := range rotations {
		participantIDs, err := decodeParticipantIDs(rotation.Participants)
		if err != nil {
			return nil, err
		}
		for _, participantID := range participantIDs {
			id, err := uuid.Parse(participantID)
			if err != nil {
				continue
			}
			seen[id] = struct{}{}
		}
	}

	emails := make(map[string]string, len(seen))
	for userID := range seen {
		user, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
			ID:             userID,
			OrganizationID: orgID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, err
		}
		emails[userID.String()] = user.Email
	}
	return emails, nil
}

func decodeParticipantIDs(data []byte) ([]string, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var participantIDs []string
	if err := json.Unmarshal(data, &participantIDs); err != nil {
		return nil, err
	}
	return participantIDs, nil
}

func timeFromDB(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time.UTC()
}
