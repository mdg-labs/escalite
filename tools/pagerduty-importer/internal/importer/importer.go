package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/tools/pagerduty-importer/internal/escalite"
	"github.com/mdg-labs/escalite/tools/pagerduty-importer/internal/mapping"
	"github.com/mdg-labs/escalite/tools/pagerduty-importer/internal/pagerduty"
)

// Options configures a PagerDuty import run.
type Options struct {
	OrganizationID uuid.UUID
	TeamID         uuid.UUID
	TeamName       string
	CreateTeam     bool
	MappingPath    string
}

// Service orchestrates dry-run summaries and imports.
type Service struct {
	source pagerduty.Source
	writer *escalite.Writer
}

func NewService(source pagerduty.Source, writer *escalite.Writer) *Service {
	return &Service{source: source, writer: writer}
}

// Summary describes what an import would create without writing.
type Summary struct {
	Counts             pagerduty.Counts
	OrganizationID     uuid.UUID
	TeamID             uuid.UUID
	UserSamples        []string
	ScheduleSamples    []string
	ServiceSamples     []string
	PolicySamples      []string
	UnsupportedObjects []pagerduty.UnsupportedObject
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
	return BuildSummary(s.source, opts.OrganizationID, teamID)
}

func (s *Service) Import(ctx context.Context, opts Options) (*mapping.Store, error) {
	teamID, err := s.resolveTeamID(ctx, opts)
	if err != nil {
		return nil, err
	}

	store := mapping.NewStore(opts.OrganizationID, teamID)
	importedAt := time.Now().UTC()

	for _, obj := range s.source.UnsupportedObjects() {
		store.AddSkipped(obj.ObjectType, obj.SourceID, obj.Reason)
	}

	if err := s.importUsers(ctx, store); err != nil {
		return nil, err
	}
	if err := s.importSchedules(ctx, store); err != nil {
		return nil, err
	}
	if err := s.importServicesAndPolicies(ctx, store); err != nil {
		return nil, err
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
	users, err := s.source.ListUsers()
	if err != nil {
		return err
	}
	for _, user := range users {
		if strings.TrimSpace(user.Email) == "" {
			store.AddSkipped("user", user.ID, "missing email")
			continue
		}
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
		store.Users[user.ID] = newID
	}
	return nil
}

func (s *Service) importSchedules(ctx context.Context, store *mapping.Store) error {
	schedules, err := s.source.ListSchedules()
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
		store.Schedules[schedule.ID] = newID

		for i, layer := range schedule.Layers {
			participants, err := EncodeParticipants(layer.Users, store.Users)
			if err != nil {
				return err
			}
			if len(layer.Users) > 0 && string(participants) == "[]" {
				store.AddSkipped("schedule_layer", layer.ID, "no mapped users in layer")
				continue
			}

			rotationID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generate rotation id: %w", err)
			}
			layerName := layer.Name
			if layerName == "" {
				layerName = fmt.Sprintf("%s layer %d", schedule.Name, i+1)
			}
			rrule := LayerRRule(layer.RotationTurnLengthSeconds)
			if err := s.writer.InsertRotation(ctx, rotationID, newID, store.OrganizationID, layerName, i+1, rrule, participants); err != nil {
				return err
			}
			store.Rotations[layer.ID] = rotationID
		}
	}
	return nil
}

func (s *Service) importServicesAndPolicies(ctx context.Context, store *mapping.Store) error {
	services, err := s.source.ListServices()
	if err != nil {
		return err
	}
	policies, err := s.source.ListEscalationPolicies()
	if err != nil {
		return err
	}
	policyByID := make(map[string]pagerduty.EscalationPolicy, len(policies))
	for _, p := range policies {
		policyByID[p.ID] = p
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

		if service.EscalationPolicyID == "" {
			store.AddSkipped("service", service.ID, "no escalation policy linked")
			continue
		}

		policy, ok := policyByID[service.EscalationPolicyID]
		if !ok {
			store.AddSkipped("service", service.ID, "escalation policy not found")
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
		store.EscalationPolicies[service.EscalationPolicyID+"::"+service.ID] = policyID

		for stepOrder, rule := range policy.EscalationRules {
			stepID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generate step id: %w", err)
			}
			repeatLast := policy.NumLoops > 0
			var maxRepeats *int32
			if repeatLast {
				v := int32(policy.NumLoops)
				maxRepeats = &v
			}
			delay := int32(rule.EscalationDelayInMinutes)
			if err := s.writer.InsertEscalationStep(ctx, stepID, policyID, store.OrganizationID, stepOrder+1, delay, repeatLast, maxRepeats); err != nil {
				return err
			}
			store.EscalationSteps[rule.ID] = stepID

			for _, target := range rule.Targets {
				targetID, err := uuid.NewV7()
				if err != nil {
					return fmt.Errorf("generate target id: %w", err)
				}
				targetType, userID, scheduleID, skipped := mapEscalationTarget(target, store)
				if skipped != "" {
					store.AddSkipped("escalation_target", target.ID, skipped)
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
					nil,
					ChannelsJSON(),
				); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func mapEscalationTarget(target pagerduty.EscalationTarget, store *mapping.Store) (string, *uuid.UUID, *uuid.UUID, string) {
	switch target.Type {
	case "user_reference":
		mapped, ok := store.Users[target.ID]
		if !ok {
			return "", nil, nil, "user not mapped"
		}
		return "user", &mapped, nil, ""
	case "schedule_reference":
		mapped, ok := store.Schedules[target.ID]
		if !ok {
			return "", nil, nil, "schedule not mapped"
		}
		return "rotation", nil, &mapped, ""
	default:
		return "", nil, nil, unsupportedTargetReason(target.Type)
	}
}

func unsupportedTargetReason(targetType string) string {
	switch targetType {
	case "escalation_policy_reference":
		return "nested escalation policy references are not supported"
	case "team_reference":
		return "team escalation targets are not supported"
	default:
		return "unsupported escalation target type: " + targetType
	}
}

// FormatSummary prints a human-readable dry-run report.
func FormatSummary(w io.Writer, summary Summary) error {
	_, err := fmt.Fprintf(w, "PagerDuty → Escalite import dry-run\n")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "Target organization: %s\n", summary.OrganizationID)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "Target team: %s\n", summary.TeamID)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "\nEntity mapping (source count → Escalite table):\n")
	if err != nil {
		return err
	}

	lines := []struct {
		label  string
		count  int
		target string
	}{
		{"users", summary.Counts.Users, "users + team_memberships"},
		{"schedules", summary.Counts.Schedules, "schedules"},
		{"schedule_layers", summary.Counts.ScheduleLayers, "rotations"},
		{"services", summary.Counts.Services, "services"},
		{"escalation_policies", summary.Counts.EscalationPolicies, "escalation_policies (per service)"},
		{"escalation_rules", summary.Counts.EscalationRules, "escalation_steps"},
	}
	for _, line := range lines {
		_, err = fmt.Fprintf(w, "  %s: %d → %s\n", line.label, line.count, line.target)
		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(w, "\nNo writes performed (dry-run).\n")
	if err != nil {
		return err
	}

	if len(summary.UserSamples) > 0 {
		_, err = fmt.Fprintf(w, "\nSample users: %s\n", strings.Join(summary.UserSamples, ", "))
		if err != nil {
			return err
		}
	}
	if len(summary.ScheduleSamples) > 0 {
		_, err = fmt.Fprintf(w, "Sample schedules: %s\n", strings.Join(summary.ScheduleSamples, ", "))
		if err != nil {
			return err
		}
	}
	if len(summary.ServiceSamples) > 0 {
		_, err = fmt.Fprintf(w, "Sample services: %s\n", strings.Join(summary.ServiceSamples, ", "))
		if err != nil {
			return err
		}
	}
	if len(summary.PolicySamples) > 0 {
		_, err = fmt.Fprintf(w, "Sample escalation policies: %s\n", strings.Join(summary.PolicySamples, ", "))
		if err != nil {
			return err
		}
	}

	if len(summary.UnsupportedObjects) > 0 {
		_, err = fmt.Fprintf(w, "\nUnsupported objects (will be skipped):\n")
		if err != nil {
			return err
		}
		for _, obj := range summary.UnsupportedObjects {
			_, err = fmt.Fprintf(w, "  - %s %s: %s\n", obj.ObjectType, obj.SourceID, obj.Reason)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// FormatImportReport prints post-import statistics including skipped objects.
func FormatImportReport(w io.Writer, store *mapping.Store, elapsed time.Duration, mappingPath string) error {
	_, err := fmt.Fprintf(w, "Import complete in %s\n", elapsed.Round(time.Millisecond))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "  users: %d\n", len(store.Users))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "  schedules: %d\n", len(store.Schedules))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "  rotations: %d\n", len(store.Rotations))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "  services: %d\n", len(store.Services))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "  escalation policies: %d\n", len(store.EscalationPolicies))
	if err != nil {
		return err
	}

	if len(store.Skipped) > 0 {
		_, err = fmt.Fprintf(w, "\nSkipped unsupported objects (%d):\n", len(store.Skipped))
		if err != nil {
			return err
		}
		for _, skip := range store.Skipped {
			_, err = fmt.Fprintf(w, "  - %s %s: %s\n", skip.EntityType, skip.SourceID, skip.Reason)
			if err != nil {
				return err
			}
		}
	}

	_, err = fmt.Fprintf(w, "\nID mapping written to %s\n", mappingPath)
	return err
}

// LayerRRule maps PagerDuty rotation turn length to an Escalite RRULE string.
func LayerRRule(rotationTurnLengthSeconds int) string {
	switch rotationTurnLengthSeconds {
	case 3600:
		return "FREQ=HOURLY;INTERVAL=1"
	case 86400:
		return "FREQ=DAILY;INTERVAL=1"
	case 604800:
		return "FREQ=WEEKLY;INTERVAL=1"
	default:
		if rotationTurnLengthSeconds > 0 && rotationTurnLengthSeconds%604800 == 0 {
			return fmt.Sprintf("FREQ=WEEKLY;INTERVAL=%d", rotationTurnLengthSeconds/604800)
		}
		if rotationTurnLengthSeconds > 0 && rotationTurnLengthSeconds%86400 == 0 {
			return fmt.Sprintf("FREQ=DAILY;INTERVAL=%d", rotationTurnLengthSeconds/86400)
		}
		if rotationTurnLengthSeconds > 0 && rotationTurnLengthSeconds%3600 == 0 {
			return fmt.Sprintf("FREQ=HOURLY;INTERVAL=%d", rotationTurnLengthSeconds/3600)
		}
		return "FREQ=DAILY;INTERVAL=1"
	}
}

// MapUserRole converts PagerDuty role to Escalite role.
func MapUserRole(pagerDutyRole string) string {
	switch strings.ToLower(strings.TrimSpace(pagerDutyRole)) {
	case "admin", "owner":
		return "admin"
	default:
		return "member"
	}
}

// EncodeParticipants maps PagerDuty user IDs to Escalite participant JSON.
func EncodeParticipants(pagerDutyUserIDs []string, userMap map[string]uuid.UUID) ([]byte, error) {
	ids := make([]string, 0, len(pagerDutyUserIDs))
	for _, srcID := range pagerDutyUserIDs {
		mapped, ok := userMap[srcID]
		if !ok {
			continue
		}
		ids = append(ids, mapped.String())
	}
	return json.Marshal(ids)
}

// ChannelsJSON returns default notification channels for imported targets.
func ChannelsJSON() []byte {
	return []byte(`["email"]`)
}

// BuildSummary loads source metadata for dry-run display.
func BuildSummary(source pagerduty.Source, orgID, teamID uuid.UUID) (Summary, error) {
	counts := source.Counts()

	users, err := source.ListUsers()
	if err != nil {
		return Summary{}, err
	}
	schedules, err := source.ListSchedules()
	if err != nil {
		return Summary{}, err
	}
	services, err := source.ListServices()
	if err != nil {
		return Summary{}, err
	}
	policies, err := source.ListEscalationPolicies()
	if err != nil {
		return Summary{}, err
	}

	return Summary{
		Counts:             counts,
		OrganizationID:     orgID,
		TeamID:             teamID,
		UserSamples:        sampleNames(users, 3, func(i int) string { return users[i].Email }),
		ScheduleSamples:    sampleNames(schedules, 3, func(i int) string { return schedules[i].Name }),
		ServiceSamples:     sampleNames(services, 3, func(i int) string { return services[i].Name }),
		PolicySamples:      sampleNames(policies, 3, func(i int) string { return policies[i].Name }),
		UnsupportedObjects: source.UnsupportedObjects(),
	}, nil
}

func sampleNames[T any](items []T, limit int, nameFn func(int) string) []string {
	if len(items) == 0 {
		return nil
	}
	if limit > len(items) {
		limit = len(items)
	}
	out := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, nameFn(i))
	}
	return out
}
