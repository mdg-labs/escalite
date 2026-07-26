package importer

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/tools/goalert-importer/internal/escalite"
	"github.com/mdg-labs/escalite/tools/goalert-importer/internal/goalert"
	"github.com/mdg-labs/escalite/tools/goalert-importer/internal/mapping"
)

// Options configures a GoAlert import run.
type Options struct {
	OrganizationID uuid.UUID
	TeamID         uuid.UUID
	TeamName       string
	CreateTeam     bool
	IncludeAlerts  bool
	MappingPath    string
}

// Service orchestrates dry-run summaries and imports.
type Service struct {
	reader *goalert.Reader
	writer *escalite.Writer
}

func NewService(reader *goalert.Reader, writer *escalite.Writer) *Service {
	return &Service{reader: reader, writer: writer}
}

func (s *Service) DryRun(ctx context.Context, opts Options) (Summary, error) {
	teamID := opts.TeamID
	if opts.CreateTeam {
		if err := s.writer.VerifyOrganization(ctx, opts.OrganizationID); err != nil {
			return Summary{}, err
		}
		teamID = uuid.Nil
	} else if teamID == uuid.Nil {
		return Summary{}, fmt.Errorf("team-id is required unless --create-team is set")
	} else {
		if err := s.writer.VerifyOrganization(ctx, opts.OrganizationID); err != nil {
			return Summary{}, err
		}
		if err := s.writer.VerifyTeam(ctx, opts.OrganizationID, teamID); err != nil {
			return Summary{}, err
		}
	}
	return BuildSummary(s.reader, opts.OrganizationID, teamID, opts.IncludeAlerts)
}

func (s *Service) Import(ctx context.Context, opts Options) (*mapping.Store, error) {
	teamID, err := s.resolveTeamID(ctx, opts)
	if err != nil {
		return nil, err
	}

	store := mapping.NewStore(opts.OrganizationID, teamID)
	importedAt := time.Now().UTC()

	if err := s.importUsers(ctx, store); err != nil {
		return nil, err
	}
	if err := s.importSchedules(ctx, store); err != nil {
		return nil, err
	}
	if err := s.importServicesAndPolicies(ctx, store); err != nil {
		return nil, err
	}
	if opts.IncludeAlerts {
		if err := s.importAlerts(ctx, store); err != nil {
			return nil, err
		}
	}

	if err := mapping.WriteJSON(opts.MappingPath, store.ToFile(importedAt)); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Service) resolveTeamID(ctx context.Context, opts Options) (uuid.UUID, error) {
	if opts.CreateTeam {
		if err := s.writer.VerifyOrganization(ctx, opts.OrganizationID); err != nil {
			return uuid.Nil, err
		}
		return s.writer.CreateTeam(ctx, opts.OrganizationID, opts.TeamName)
	}
	if opts.TeamID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("team-id is required unless --create-team is set")
	}
	if err := s.writer.VerifyOrganization(ctx, opts.OrganizationID); err != nil {
		return uuid.Nil, err
	}
	if err := s.writer.VerifyTeam(ctx, opts.OrganizationID, opts.TeamID); err != nil {
		return uuid.Nil, err
	}
	return opts.TeamID, nil
}

func (s *Service) importUsers(ctx context.Context, store *mapping.Store) error {
	users, err := s.reader.ListUsers(ctx)
	if err != nil {
		return err
	}
	for _, user := range users {
		newID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate user id: %w", err)
		}
		role := MapUserRole(user.Role)
		if err := s.writer.InsertUser(ctx, newID, store.OrganizationID, user.Email, role); err != nil {
			return err
		}
		memID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate membership id: %w", err)
		}
		if err := s.writer.InsertTeamMembership(ctx, memID, store.TeamID, newID, store.OrganizationID); err != nil {
			return err
		}
		store.Users[user.ID.String()] = newID
	}
	return nil
}

func (s *Service) importSchedules(ctx context.Context, store *mapping.Store) error {
	schedules, err := s.reader.ListSchedules(ctx)
	if err != nil {
		return err
	}
	for _, schedule := range schedules {
		newID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate schedule id: %w", err)
		}
		timezone := schedule.TimeZone
		if timezone == "" {
			timezone = "UTC"
		}
		if err := s.writer.InsertSchedule(ctx, newID, store.OrganizationID, store.TeamID, schedule.Name, timezone); err != nil {
			return err
		}
		store.Schedules[schedule.ID.String()] = newID
	}

	rotations, err := s.reader.ListRotations(ctx)
	if err != nil {
		return err
	}
	layerBySchedule := make(map[uuid.UUID]int)
	for _, rotation := range rotations {
		escScheduleID, ok := store.Schedules[rotation.ScheduleID.String()]
		if !ok {
			store.AddSkipped("rotation", rotation.ID.String(), "schedule not mapped")
			continue
		}
		layerBySchedule[escScheduleID]++
		layer := layerBySchedule[escScheduleID]

		participants, err := EncodeParticipants(rotation.Participants, store.Users)
		if err != nil {
			return err
		}
		if string(participants) == "[]" || string(participants) == "null" {
			participants = []byte("[]")
		}

		newID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate rotation id: %w", err)
		}
		rrule := RotationRRule(rotation.Type, rotation.ShiftLength)
		if err := s.writer.InsertRotation(ctx, newID, escScheduleID, store.OrganizationID, rotation.Name, layer, rrule, participants); err != nil {
			return err
		}
		store.Rotations[rotation.ID.String()] = newID
		store.RotationSchedules[rotation.ID.String()] = escScheduleID
	}
	return nil
}

func (s *Service) importServicesAndPolicies(ctx context.Context, store *mapping.Store) error {
	services, err := s.reader.ListServices(ctx)
	if err != nil {
		return err
	}
	steps, err := s.reader.ListEscalationSteps(ctx)
	if err != nil {
		return err
	}
	actions, err := s.reader.ListEscalationActions(ctx)
	if err != nil {
		return err
	}
	policies, err := s.reader.ListEscalationPolicies(ctx)
	if err != nil {
		return err
	}
	policyByID := make(map[string]goalert.EscalationPolicy, len(policies))
	for _, p := range policies {
		policyByID[p.ID] = p
	}
	stepsByPolicy := make(map[string][]goalert.EscalationStep)
	for _, step := range steps {
		stepsByPolicy[step.EscalationPolicyID] = append(stepsByPolicy[step.EscalationPolicyID], step)
	}
	actionsByStep := make(map[string][]goalert.EscalationAction)
	for _, action := range actions {
		actionsByStep[action.StepID] = append(actionsByStep[action.StepID], action)
	}

	for _, service := range services {
		serviceID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate service id: %w", err)
		}
		if err := s.writer.InsertService(ctx, serviceID, store.OrganizationID, store.TeamID, service.Name); err != nil {
			return err
		}
		store.Services[service.ID] = serviceID

		policy, ok := policyByID[service.EscalationPolicyID]
		if !ok || service.EscalationPolicyID == "" {
			store.AddSkipped("service", service.ID, "no escalation policy linked")
			continue
		}

		policyID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate policy id: %w", err)
		}
		policyName := policy.Name
		if policyName == "" {
			policyName = service.Name + " policy"
		}
		if err := s.writer.InsertEscalationPolicy(ctx, policyID, store.OrganizationID, serviceID, policyName); err != nil {
			return err
		}
		store.EscalationPolicies[service.EscalationPolicyID] = policyID

		policySteps := stepsByPolicy[service.EscalationPolicyID]
		for _, step := range policySteps {
			stepID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generate step id: %w", err)
			}
			stepOrder := int(step.StepNumber) + 1
			repeatLast := policy.Repeat > 0
			var maxRepeats *int32
			if repeatLast {
				maxRepeats = &policy.Repeat
			}
			if err := s.writer.InsertEscalationStep(ctx, stepID, policyID, store.OrganizationID, stepOrder, step.Delay, repeatLast, maxRepeats); err != nil {
				return err
			}
			store.EscalationSteps[step.ID] = stepID

			for _, action := range actionsByStep[step.ID] {
				targetID, err := uuid.NewV7()
				if err != nil {
					return fmt.Errorf("generate target id: %w", err)
				}
				targetType, userID, scheduleID, webhookURL, skipped := mapEscalationAction(action, store)
				if skipped != "" {
					store.AddSkipped("escalation_action", step.ID, skipped)
					continue
				}
				if err := s.writer.InsertEscalationStepTarget(
					ctx,
					targetID,
					stepID,
					store.OrganizationID,
					targetType,
					userID,
					scheduleID,
					webhookURL,
					ChannelsJSON(),
				); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func mapEscalationAction(action goalert.EscalationAction, store *mapping.Store) (string, *uuid.UUID, *uuid.UUID, *string, string) {
	if action.UserID != nil {
		mapped, ok := store.Users[action.UserID.String()]
		if !ok {
			return "", nil, nil, nil, "user not mapped"
		}
		return "user", &mapped, nil, nil, ""
	}
	if action.ScheduleID != nil {
		mapped, ok := store.Schedules[action.ScheduleID.String()]
		if !ok {
			return "", nil, nil, nil, "schedule not mapped"
		}
		return "rotation", nil, &mapped, nil, ""
	}
	if action.RotationID != nil {
		mappedSchedule, ok := store.RotationSchedules[action.RotationID.String()]
		if !ok {
			return "", nil, nil, nil, "rotation schedule not mapped"
		}
		return "rotation", nil, &mappedSchedule, nil, ""
	}
	if action.ChannelDest != nil && *action.ChannelDest != "" {
		url := *action.ChannelDest
		return "webhook", nil, nil, &url, ""
	}
	return "", nil, nil, nil, "empty escalation action"
}

func (s *Service) importAlerts(ctx context.Context, store *mapping.Store) error {
	alerts, err := s.reader.ListAlerts(ctx)
	if err != nil {
		return err
	}
	for _, alert := range alerts {
		serviceID, ok := store.Services[alert.ServiceID]
		if !ok {
			store.AddSkipped("alert", fmt.Sprintf("%d", alert.ID), "service not mapped")
			continue
		}
		alertID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate alert id: %w", err)
		}
		status := MapAlertStatus(alert.Status)
		dedupKey := alert.DedupKey
		if dedupKey == "" {
			dedupKey = fmt.Sprintf("goalert-%d", alert.ID)
		}
		if err := s.writer.InsertAlert(
			ctx,
			alertID,
			store.OrganizationID,
			serviceID,
			status,
			dedupKey,
			alert.Summary,
			alert.Details,
			alert.CreatedAt,
		); err != nil {
			return err
		}
		store.Alerts[fmt.Sprintf("%d", alert.ID)] = alertID
	}
	return nil
}
