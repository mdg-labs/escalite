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
	ActionLogin                 = "auth.login"
	ActionLoginFailed           = "auth.login_failed"
	ActionLogout                = "auth.logout"
	ActionRoleChanged           = "user.role_changed"
	ActionIntegrationKeyCreated   = "integration_key.created"
	ActionIntegrationKeyRevoked   = "integration_key.revoked"
	ActionEscalationPolicyCreated = "escalation_policy.created"
	ActionEscalationPolicyUpdated = "escalation_policy.updated"
	ActionEscalationPolicyDeleted = "escalation_policy.deleted"
	ActionOverrideCreated         = "override.created"
	ActionOverrideDeleted         = "override.deleted"
	ActionAlertEscalationSnoozed   = "alert.escalation_snoozed"
	ActionAlertReEscalated         = "alert.re_escalated"

	targetTypeUser             = "user"
	targetTypeIntegrationKey   = "integration_key"
	targetTypeEscalationPolicy = "escalation_policy"
	targetTypeOverride         = "override"
	targetTypeAlert            = "alert"
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
		"duration_minutes":     durationMinutes,
		"next_escalation_at":   nextEscalationAt,
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
