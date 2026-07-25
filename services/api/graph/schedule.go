package graph

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func (r *mutationResolver) CreateSchedule(ctx context.Context, input model.CreateScheduleInput) (*model.Schedule, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, gqlerr.New(handlers.CodeValidation, "name is required")
	}
	if err := validateTimezone(input.Timezone); err != nil {
		return nil, err
	}

	teamID, err := parseUUIDField(input.TeamID, "teamId")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	if _, err := queries.GetTeamByID(ctx, db.GetTeamByIDParams{
		ID:             teamID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeNotFound, "team not found")
		}
		r.logger.Error("load team failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	scheduleID := uuid.Must(uuid.NewV7())
	schedule, err := queries.CreateSchedule(ctx, db.CreateScheduleParams{
		ID:             scheduleID,
		OrganizationID: sc.User.OrganizationID,
		TeamID:         teamID,
		Name:           name,
		Timezone:       strings.TrimSpace(input.Timezone),
	})
	if err != nil {
		if isTimezoneCheckViolation(err) {
			return nil, gqlerr.New(handlers.CodeValidation, "timezone must be a valid IANA timezone")
		}
		r.logger.Error("create schedule failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return scheduleFromDB(schedule, nil), nil
}

func (r *mutationResolver) UpdateSchedule(ctx context.Context, input model.UpdateScheduleInput) (*model.Schedule, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, gqlerr.New(handlers.CodeValidation, "name is required")
	}
	if err := validateTimezone(input.Timezone); err != nil {
		return nil, err
	}

	scheduleID, err := parseUUIDField(input.ID, "id")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	if _, err := queries.GetScheduleByID(ctx, db.GetScheduleByIDParams{
		ID:             scheduleID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeNotFound, "schedule not found")
		}
		r.logger.Error("load schedule failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	schedule, err := queries.UpdateSchedule(ctx, db.UpdateScheduleParams{
		ID:             scheduleID,
		OrganizationID: sc.User.OrganizationID,
		Name:           name,
		Timezone:       strings.TrimSpace(input.Timezone),
	})
	if err != nil {
		if isTimezoneCheckViolation(err) {
			return nil, gqlerr.New(handlers.CodeValidation, "timezone must be a valid IANA timezone")
		}
		r.logger.Error("update schedule failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	rotations, err := queries.ListRotationsByScheduleID(ctx, db.ListRotationsByScheduleIDParams{
		ScheduleID:     scheduleID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("list rotations failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return scheduleFromDB(schedule, rotations), nil
}

func (r *mutationResolver) DeleteSchedule(ctx context.Context, id string) (bool, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return false, err
	}

	scheduleID, err := parseUUIDField(id, "id")
	if err != nil {
		return false, err
	}

	queries := db.New(r.pool)
	if _, err := queries.GetScheduleByID(ctx, db.GetScheduleByIDParams{
		ID:             scheduleID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, gqlerr.New(handlers.CodeNotFound, "schedule not found")
		}
		r.logger.Error("load schedule failed", "error", err)
		return false, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if err := queries.DeleteSchedule(ctx, db.DeleteScheduleParams{
		ID:             scheduleID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		r.logger.Error("delete schedule failed", "error", err)
		return false, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return true, nil
}

func (r *mutationResolver) CreateRotation(ctx context.Context, input model.CreateRotationInput) (*model.Rotation, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, gqlerr.New(handlers.CodeValidation, "name is required")
	}
	if err := validateRotationLayer(input.Layer); err != nil {
		return nil, err
	}
	if err := validateRRule(input.Rrule); err != nil {
		return nil, err
	}

	scheduleID, err := parseUUIDField(input.ScheduleID, "scheduleId")
	if err != nil {
		return nil, err
	}

	participantIDs, err := parseParticipantIDs(input.ParticipantIds)
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	if _, err := queries.GetScheduleByID(ctx, db.GetScheduleByIDParams{
		ID:             scheduleID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeNotFound, "schedule not found")
		}
		r.logger.Error("load schedule failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if err := ensureParticipantsExist(ctx, queries, sc.User.OrganizationID, participantIDs); err != nil {
		return nil, err
	}

	participantsJSON, err := encodeParticipantIDs(participantIDs)
	if err != nil {
		r.logger.Error("encode participants failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	rotation, err := queries.CreateRotation(ctx, db.CreateRotationParams{
		ID:             uuid.Must(uuid.NewV7()),
		ScheduleID:     scheduleID,
		OrganizationID: sc.User.OrganizationID,
		Name:           name,
		Layer:          int32(input.Layer),
		Rrule:          strings.TrimSpace(input.Rrule),
		Participants:   participantsJSON,
	})
	if err != nil {
		if isRotationLayerConflict(err) {
			return nil, gqlerr.New(handlers.CodeValidation, "layer already exists on this schedule")
		}
		r.logger.Error("create rotation failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	gqlRotation, err := rotationFromDB(rotation)
	if err != nil {
		r.logger.Error("decode rotation participants failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	return gqlRotation, nil
}

func (r *mutationResolver) UpdateRotation(ctx context.Context, input model.UpdateRotationInput) (*model.Rotation, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, gqlerr.New(handlers.CodeValidation, "name is required")
	}
	if err := validateRotationLayer(input.Layer); err != nil {
		return nil, err
	}
	if err := validateRRule(input.Rrule); err != nil {
		return nil, err
	}

	rotationID, err := parseUUIDField(input.ID, "id")
	if err != nil {
		return nil, err
	}

	participantIDs, err := parseParticipantIDs(input.ParticipantIds)
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	if _, err := queries.GetRotationByID(ctx, db.GetRotationByIDParams{
		ID:             rotationID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeNotFound, "rotation not found")
		}
		r.logger.Error("load rotation failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if err := ensureParticipantsExist(ctx, queries, sc.User.OrganizationID, participantIDs); err != nil {
		return nil, err
	}

	participantsJSON, err := encodeParticipantIDs(participantIDs)
	if err != nil {
		r.logger.Error("encode participants failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	rotation, err := queries.UpdateRotation(ctx, db.UpdateRotationParams{
		ID:             rotationID,
		OrganizationID: sc.User.OrganizationID,
		Name:           name,
		Layer:          int32(input.Layer),
		Rrule:          strings.TrimSpace(input.Rrule),
		Participants:   participantsJSON,
	})
	if err != nil {
		if isRotationLayerConflict(err) {
			return nil, gqlerr.New(handlers.CodeValidation, "layer already exists on this schedule")
		}
		r.logger.Error("update rotation failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	gqlRotation, err := rotationFromDB(rotation)
	if err != nil {
		r.logger.Error("decode rotation participants failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	return gqlRotation, nil
}

func (r *mutationResolver) DeleteRotation(ctx context.Context, id string) (bool, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return false, err
	}

	rotationID, err := parseUUIDField(id, "id")
	if err != nil {
		return false, err
	}

	queries := db.New(r.pool)
	if _, err := queries.GetRotationByID(ctx, db.GetRotationByIDParams{
		ID:             rotationID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, gqlerr.New(handlers.CodeNotFound, "rotation not found")
		}
		r.logger.Error("load rotation failed", "error", err)
		return false, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if err := queries.DeleteRotation(ctx, db.DeleteRotationParams{
		ID:             rotationID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		r.logger.Error("delete rotation failed", "error", err)
		return false, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return true, nil
}

func (r *queryResolver) Schedule(ctx context.Context, id string) (*model.Schedule, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return nil, err
	}

	scheduleID, err := parseUUIDField(id, "id")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	schedule, err := queries.GetScheduleByID(ctx, db.GetScheduleByIDParams{
		ID:             scheduleID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		r.logger.Error("load schedule failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	rotations, err := queries.ListRotationsByScheduleID(ctx, db.ListRotationsByScheduleIDParams{
		ScheduleID:     scheduleID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("list rotations failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return scheduleFromDB(schedule, rotations), nil
}

func (r *queryResolver) Schedules(ctx context.Context, teamID string) ([]*model.Schedule, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return nil, err
	}

	teamUUID, err := parseUUIDField(teamID, "teamId")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	if _, err := queries.GetTeamByID(ctx, db.GetTeamByIDParams{
		ID:             teamUUID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeNotFound, "team not found")
		}
		r.logger.Error("load team failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	schedules, err := queries.ListSchedulesByTeamID(ctx, db.ListSchedulesByTeamIDParams{
		TeamID:         teamUUID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("list schedules failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	result := make([]*model.Schedule, 0, len(schedules))
	for _, schedule := range schedules {
		rotations, err := queries.ListRotationsByScheduleID(ctx, db.ListRotationsByScheduleIDParams{
			ScheduleID:     schedule.ID,
			OrganizationID: sc.User.OrganizationID,
		})
		if err != nil {
			r.logger.Error("list rotations failed", "error", err)
			return nil, gqlerr.New(handlers.CodeInternal, "internal error")
		}
		result = append(result, scheduleFromDB(schedule, rotations))
	}

	return result, nil
}

func ensureParticipantsExist(ctx context.Context, q db.Querier, orgID uuid.UUID, participantIDs []uuid.UUID) error {
	for _, participantID := range participantIDs {
		if _, err := q.GetUserByID(ctx, db.GetUserByIDParams{
			ID:             participantID,
			OrganizationID: orgID,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return gqlerr.New(handlers.CodeValidation, "participant user not found")
			}
			return gqlerr.New(handlers.CodeInternal, "internal error")
		}
	}
	return nil
}

func isTimezoneCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514" && strings.Contains(pgErr.ConstraintName, "timezone")
}

func isRotationLayerConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "layer")
}
