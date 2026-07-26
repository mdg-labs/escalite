package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/internal/db"
)

const (
	ActionLogin                    = "auth.login"
	ActionLoginFailed              = "auth.login_failed"
	ActionLogout                   = "auth.logout"
	ActionRoleChanged              = "user.role_changed"
	ActionIntegrationKeyCreated    = "integration_key.created"
	ActionIntegrationKeyRevoked    = "integration_key.revoked"
	ActionEscalationPolicyCreated  = "escalation_policy.created"
	ActionEscalationPolicyUpdated  = "escalation_policy.updated"
	ActionEscalationPolicyDeleted  = "escalation_policy.deleted"
	ActionHeartbeatMonitorCreated  = "heartbeat_monitor.created"
	ActionHeartbeatMonitorUpdated  = "heartbeat_monitor.updated"
	ActionHeartbeatMonitorDeleted  = "heartbeat_monitor.deleted"
	ActionMaintenanceWindowCreated = "maintenance_window.created"
	ActionMaintenanceWindowUpdated = "maintenance_window.updated"
	ActionMaintenanceWindowDeleted = "maintenance_window.deleted"
	ActionIncidentCreated          = "incident.created"
	ActionIncidentStatusUpdated    = "incident.status_updated"
	ActionIncidentRoleDefCreated   = "incident_role_definition.created"
	ActionIncidentRoleDefUpdated   = "incident_role_definition.updated"
	ActionIncidentRoleDefDeleted   = "incident_role_definition.deleted"
	ActionOverrideCreated          = "override.created"
	ActionOverrideDeleted          = "override.deleted"
	ActionAlertEscalationSnoozed   = "alert.escalation_snoozed"
	ActionAlertReEscalated         = "alert.re_escalated"
	ActionServiceCreated           = "service.created"
	ActionServiceUpdated           = "service.updated"
	ActionServiceDeleted           = "service.deleted"
	ActionSlackWorkspaceConnected  = "slack_workspace.connected"
	ActionScimUserProvisioned      = "scim.user_provisioned"
	ActionScimUserDeprovisioned    = "scim.user_deprovisioned"
	ActionScimTokenRotated         = "scim.token_rotated"

	targetTypeUser              = "user"
	targetTypeSlackWorkspace    = "organization_slack_settings"
	targetTypeIntegrationKey    = "integration_key"
	targetTypeEscalationPolicy  = "escalation_policy"
	targetTypeHeartbeatMonitor  = "heartbeat_monitor"
	targetTypeMaintenanceWindow = "maintenance_window"
	targetTypeIncident          = "incident"
	targetTypeIncidentRoleDef   = "incident_role_definition"
	targetTypeOverride          = "override"
	targetTypeAlert             = "alert"
	targetTypeService           = "service"
)

// RequestMeta captures HTTP request context stored in audit metadata.
type RequestMeta struct {
	IP        string `json:"ip,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	Email     string `json:"email,omitempty"`
}

// Recorder writes append-only audit_events rows.
type Recorder struct {
	logger *slog.Logger
}

// NewRecorder returns an audit event recorder.
func NewRecorder(logger *slog.Logger) *Recorder {
	return &Recorder{logger: logger}
}

// RequestMetaFromHTTP extracts client IP and user agent from an HTTP request.
func RequestMetaFromHTTP(r *http.Request) RequestMeta {
	clientIP := strings.TrimSpace(r.RemoteAddr)
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		clientIP = strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	if host, _, ok := strings.Cut(clientIP, ":"); ok && strings.Count(clientIP, ":") == 1 {
		clientIP = host
	}

	userAgent := strings.TrimSpace(r.UserAgent())
	return RequestMeta{
		IP:        clientIP,
		UserAgent: userAgent,
	}
}

// Login records a successful login.
func (r *Recorder) Login(ctx context.Context, q db.Querier, orgID, actorID uuid.UUID, meta RequestMeta) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionLogin,
		TargetType:     pgtype.Text{String: targetTypeUser, Valid: true},
		TargetID:       pgtype.UUID{Bytes: actorID, Valid: true},
		Metadata:       mustMarshalMeta(r.logger, meta),
	})
}

// LoginFailed records a failed login attempt.
func (r *Recorder) LoginFailed(ctx context.Context, q db.Querier, orgID uuid.UUID, targetUserID *uuid.UUID, meta RequestMeta) {
	if orgID == uuid.Nil {
		return
	}

	params := db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		Action:         ActionLoginFailed,
		Metadata:       mustMarshalMeta(r.logger, meta),
	}
	if targetUserID != nil {
		params.TargetType = pgtype.Text{String: targetTypeUser, Valid: true}
		params.TargetID = pgtype.UUID{Bytes: *targetUserID, Valid: true}
	}

	r.insert(ctx, q, params)
}

// Logout records a user logout.
func (r *Recorder) Logout(ctx context.Context, q db.Querier, orgID, actorID uuid.UUID, meta RequestMeta) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionLogout,
		TargetType:     pgtype.Text{String: targetTypeUser, Valid: true},
		TargetID:       pgtype.UUID{Bytes: actorID, Valid: true},
		Metadata:       mustMarshalMeta(r.logger, meta),
	})
}

// RoleChanged records a user role change. Call from user-management handlers when implemented.
func (r *Recorder) RoleChanged(ctx context.Context, q db.Querier, orgID, actorID, targetUserID uuid.UUID, previousRole, newRole string) {
	meta, err := json.Marshal(map[string]string{
		"previous_role": previousRole,
		"new_role":      newRole,
	})
	if err != nil {
		r.logger.Error("marshal role change audit metadata failed", "error", err)
		return
	}

	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionRoleChanged,
		TargetType:     pgtype.Text{String: targetTypeUser, Valid: true},
		TargetID:       pgtype.UUID{Bytes: targetUserID, Valid: true},
		Metadata:       meta,
	})
}

// IntegrationKeyCreated records integration key creation. Call from key lifecycle handlers when implemented.
func (r *Recorder) IntegrationKeyCreated(ctx context.Context, q db.Querier, orgID, actorID, keyID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionIntegrationKeyCreated,
		TargetType:     pgtype.Text{String: targetTypeIntegrationKey, Valid: true},
		TargetID:       pgtype.UUID{Bytes: keyID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// IntegrationKeyRevoked records integration key revocation. Call from key lifecycle handlers when implemented.
func (r *Recorder) IntegrationKeyRevoked(ctx context.Context, q db.Querier, orgID, actorID, keyID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionIntegrationKeyRevoked,
		TargetType:     pgtype.Text{String: targetTypeIntegrationKey, Valid: true},
		TargetID:       pgtype.UUID{Bytes: keyID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// EscalationPolicyCreated records escalation policy creation.
func (r *Recorder) EscalationPolicyCreated(ctx context.Context, q db.Querier, orgID, actorID, policyID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionEscalationPolicyCreated,
		TargetType:     pgtype.Text{String: targetTypeEscalationPolicy, Valid: true},
		TargetID:       pgtype.UUID{Bytes: policyID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// EscalationPolicyUpdated records escalation policy updates.
func (r *Recorder) EscalationPolicyUpdated(ctx context.Context, q db.Querier, orgID, actorID, policyID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionEscalationPolicyUpdated,
		TargetType:     pgtype.Text{String: targetTypeEscalationPolicy, Valid: true},
		TargetID:       pgtype.UUID{Bytes: policyID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// EscalationPolicyDeleted records escalation policy deletion.
func (r *Recorder) EscalationPolicyDeleted(ctx context.Context, q db.Querier, orgID, actorID, policyID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionEscalationPolicyDeleted,
		TargetType:     pgtype.Text{String: targetTypeEscalationPolicy, Valid: true},
		TargetID:       pgtype.UUID{Bytes: policyID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// HeartbeatMonitorCreated records heartbeat monitor creation.
func (r *Recorder) HeartbeatMonitorCreated(ctx context.Context, q db.Querier, orgID, actorID, monitorID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionHeartbeatMonitorCreated,
		TargetType:     pgtype.Text{String: targetTypeHeartbeatMonitor, Valid: true},
		TargetID:       pgtype.UUID{Bytes: monitorID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// HeartbeatMonitorUpdated records heartbeat monitor updates.
func (r *Recorder) HeartbeatMonitorUpdated(ctx context.Context, q db.Querier, orgID, actorID, monitorID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionHeartbeatMonitorUpdated,
		TargetType:     pgtype.Text{String: targetTypeHeartbeatMonitor, Valid: true},
		TargetID:       pgtype.UUID{Bytes: monitorID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// HeartbeatMonitorDeleted records heartbeat monitor deletion.
func (r *Recorder) HeartbeatMonitorDeleted(ctx context.Context, q db.Querier, orgID, actorID, monitorID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionHeartbeatMonitorDeleted,
		TargetType:     pgtype.Text{String: targetTypeHeartbeatMonitor, Valid: true},
		TargetID:       pgtype.UUID{Bytes: monitorID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// MaintenanceWindowCreated records maintenance window creation.
func (r *Recorder) MaintenanceWindowCreated(ctx context.Context, q db.Querier, orgID, actorID, windowID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionMaintenanceWindowCreated,
		TargetType:     pgtype.Text{String: targetTypeMaintenanceWindow, Valid: true},
		TargetID:       pgtype.UUID{Bytes: windowID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// MaintenanceWindowUpdated records maintenance window updates.
func (r *Recorder) MaintenanceWindowUpdated(ctx context.Context, q db.Querier, orgID, actorID, windowID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionMaintenanceWindowUpdated,
		TargetType:     pgtype.Text{String: targetTypeMaintenanceWindow, Valid: true},
		TargetID:       pgtype.UUID{Bytes: windowID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// MaintenanceWindowDeleted records maintenance window deletion.
func (r *Recorder) MaintenanceWindowDeleted(ctx context.Context, q db.Querier, orgID, actorID, windowID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionMaintenanceWindowDeleted,
		TargetType:     pgtype.Text{String: targetTypeMaintenanceWindow, Valid: true},
		TargetID:       pgtype.UUID{Bytes: windowID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// OverrideCreated records on-call override creation.
func (r *Recorder) OverrideCreated(ctx context.Context, q db.Querier, orgID, actorID, overrideID uuid.UUID, metadata map[string]any) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionOverrideCreated,
		TargetType:     pgtype.Text{String: targetTypeOverride, Valid: true},
		TargetID:       pgtype.UUID{Bytes: overrideID, Valid: true},
		Metadata:       mustMarshalAnyMeta(r.logger, metadata),
	})
}

// OverrideDeleted records on-call override soft-deletion.
func (r *Recorder) OverrideDeleted(ctx context.Context, q db.Querier, orgID, actorID, overrideID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionOverrideDeleted,
		TargetType:     pgtype.Text{String: targetTypeOverride, Valid: true},
		TargetID:       pgtype.UUID{Bytes: overrideID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// AlertEscalationSnoozed records a manual escalation snooze.
func (r *Recorder) AlertEscalationSnoozed(ctx context.Context, q db.Querier, orgID, actorID, alertID uuid.UUID, durationMinutes int32, nextEscalationAt string) {
	meta, err := json.Marshal(map[string]any{
		"duration_minutes":   durationMinutes,
		"next_escalation_at": nextEscalationAt,
	})
	if err != nil {
		r.logger.Error("marshal snooze audit metadata failed", "error", err)
		return
	}

	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionAlertEscalationSnoozed,
		TargetType:     pgtype.Text{String: targetTypeAlert, Valid: true},
		TargetID:       pgtype.UUID{Bytes: alertID, Valid: true},
		Metadata:       meta,
	})
}

// AlertReEscalated records a manual re-escalation to step 1.
func (r *Recorder) AlertReEscalated(ctx context.Context, q db.Querier, orgID, actorID, alertID uuid.UUID) {
	meta, err := json.Marshal(map[string]any{
		"reset_to_step": 1,
	})
	if err != nil {
		r.logger.Error("marshal re-escalate audit metadata failed", "error", err)
		return
	}

	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionAlertReEscalated,
		TargetType:     pgtype.Text{String: targetTypeAlert, Valid: true},
		TargetID:       pgtype.UUID{Bytes: alertID, Valid: true},
		Metadata:       meta,
	})
}

// ServiceCreated records service creation.
func (r *Recorder) ServiceCreated(ctx context.Context, q db.Querier, orgID, actorID, serviceID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionServiceCreated,
		TargetType:     pgtype.Text{String: targetTypeService, Valid: true},
		TargetID:       pgtype.UUID{Bytes: serviceID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// ServiceUpdated records service updates.
func (r *Recorder) ServiceUpdated(ctx context.Context, q db.Querier, orgID, actorID, serviceID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionServiceUpdated,
		TargetType:     pgtype.Text{String: targetTypeService, Valid: true},
		TargetID:       pgtype.UUID{Bytes: serviceID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// ServiceDeleted records service soft-deletion.
func (r *Recorder) ServiceDeleted(ctx context.Context, q db.Querier, orgID, actorID, serviceID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionServiceDeleted,
		TargetType:     pgtype.Text{String: targetTypeService, Valid: true},
		TargetID:       pgtype.UUID{Bytes: serviceID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// IncidentCreated records incident creation.
func (r *Recorder) IncidentCreated(ctx context.Context, q db.Querier, orgID, actorID, incidentID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionIncidentCreated,
		TargetType:     pgtype.Text{String: targetTypeIncident, Valid: true},
		TargetID:       pgtype.UUID{Bytes: incidentID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// IncidentStatusUpdated records incident status changes.
func (r *Recorder) IncidentStatusUpdated(ctx context.Context, q db.Querier, orgID, actorID, incidentID uuid.UUID, status string) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionIncidentStatusUpdated,
		TargetType:     pgtype.Text{String: targetTypeIncident, Valid: true},
		TargetID:       pgtype.UUID{Bytes: incidentID, Valid: true},
		Metadata:       mustMarshalAnyMeta(r.logger, map[string]any{"status": status}),
	})
}

// IncidentRoleDefinitionCreated records role definition creation.
func (r *Recorder) IncidentRoleDefinitionCreated(ctx context.Context, q db.Querier, orgID, actorID, roleDefID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionIncidentRoleDefCreated,
		TargetType:     pgtype.Text{String: targetTypeIncidentRoleDef, Valid: true},
		TargetID:       pgtype.UUID{Bytes: roleDefID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// IncidentRoleDefinitionUpdated records role definition updates.
func (r *Recorder) IncidentRoleDefinitionUpdated(ctx context.Context, q db.Querier, orgID, actorID, roleDefID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionIncidentRoleDefUpdated,
		TargetType:     pgtype.Text{String: targetTypeIncidentRoleDef, Valid: true},
		TargetID:       pgtype.UUID{Bytes: roleDefID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// IncidentRoleDefinitionDeleted records role definition deletion.
func (r *Recorder) IncidentRoleDefinitionDeleted(ctx context.Context, q db.Querier, orgID, actorID, roleDefID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionIncidentRoleDefDeleted,
		TargetType:     pgtype.Text{String: targetTypeIncidentRoleDef, Valid: true},
		TargetID:       pgtype.UUID{Bytes: roleDefID, Valid: true},
		Metadata:       []byte("{}"),
	})
}

// SlackWorkspaceConnected records a Slack app OAuth workspace install or reinstall.
func (r *Recorder) SlackWorkspaceConnected(ctx context.Context, q db.Querier, orgID, actorID uuid.UUID, workspaceID string) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionSlackWorkspaceConnected,
		TargetType:     pgtype.Text{String: targetTypeSlackWorkspace, Valid: true},
		TargetID:       pgtype.UUID{Bytes: orgID, Valid: true},
		Metadata:       mustMarshalAnyMeta(r.logger, map[string]any{"workspace_id": workspaceID}),
	})
}

func (r *Recorder) insert(ctx context.Context, q db.Querier, params db.CreateAuditEventParams) {
	if _, err := q.CreateAuditEvent(ctx, params); err != nil {
		r.logger.Error("audit event insert failed", "action", params.Action, "error", err)
	}
}

func mustMarshalMeta(logger *slog.Logger, meta RequestMeta) []byte {
	data, err := json.Marshal(meta)
	if err != nil {
		logger.Error("marshal audit metadata failed", "error", err)
		return []byte("{}")
	}
	return data
}

// ScimUserProvisioned records SCIM user creation.
func (r *Recorder) ScimUserProvisioned(ctx context.Context, q db.Querier, orgID, userID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		TargetType:     pgtype.Text{String: targetTypeUser, Valid: true},
		TargetID:       pgtype.UUID{Bytes: userID, Valid: true},
		Action:         ActionScimUserProvisioned,
		Metadata:       []byte("{}"),
	})
}

// ScimUserDeprovisioned records SCIM user deprovisioning.
func (r *Recorder) ScimUserDeprovisioned(ctx context.Context, q db.Querier, orgID, userID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		TargetType:     pgtype.Text{String: targetTypeUser, Valid: true},
		TargetID:       pgtype.UUID{Bytes: userID, Valid: true},
		Action:         ActionScimUserDeprovisioned,
		Metadata:       []byte("{}"),
	})
}

// ScimTokenRotated records SCIM bearer token rotation by an org admin.
func (r *Recorder) ScimTokenRotated(ctx context.Context, q db.Querier, orgID, actorID uuid.UUID) {
	r.insert(ctx, q, db.CreateAuditEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ActorID:        pgtype.UUID{Bytes: actorID, Valid: true},
		Action:         ActionScimTokenRotated,
		Metadata:       []byte("{}"),
	})
}

func mustMarshalAnyMeta(logger *slog.Logger, meta map[string]any) []byte {
	if meta == nil {
		return []byte("{}")
	}
	data, err := json.Marshal(meta)
	if err != nil {
		logger.Error("marshal audit metadata failed", "error", err)
		return []byte("{}")
	}
	return data
}
