package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/engine/channels/slackdm"
	"github.com/mdg-labs/escalite/services/engine/escalationapi"
)

const slackInteractiveMaxBodyBytes = 256 * 1024

// SlackInteractiveConfig controls Slack interactive request handling.
type SlackInteractiveConfig struct {
	SigningSecret string
}

// SlackInteractiveHandler handles POST /api/v1/integrations/slack/interactive.
type SlackInteractiveHandler struct {
	pool   *pgxpool.Pool
	jobs   escalationapi.JobProducer
	logger *slog.Logger
	cfg    SlackInteractiveConfig
}

// NewSlackInteractiveHandler returns a handler for Slack interactive payloads.
func NewSlackInteractiveHandler(
	pool *pgxpool.Pool,
	jobs escalationapi.JobProducer,
	logger *slog.Logger,
	cfg SlackInteractiveConfig,
) *SlackInteractiveHandler {
	return &SlackInteractiveHandler{
		pool:   pool,
		jobs:   jobs,
		logger: logger,
		cfg:    cfg,
	}
}

type slackInteractivePayload struct {
	Type string `json:"type"`
	User struct {
		ID string `json:"id"`
	} `json:"user"`
	Team struct {
		ID string `json:"id"`
	} `json:"team"`
	Actions []struct {
		ActionID string `json:"action_id"`
		Value    string `json:"value"`
	} `json:"actions"`
}

type slackButtonValue struct {
	AlertID string `json:"alert_id"`
}

// ServeHTTP verifies the Slack signature and processes block action buttons.
func (h *SlackInteractiveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	if strings.TrimSpace(h.cfg.SigningSecret) == "" {
		WriteAPIError(w, http.StatusNotFound, CodeNotFound, "not found")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, slackInteractiveMaxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			WriteAPIError(w, http.StatusRequestEntityTooLarge, CodePayloadTooLarge, "request body too large")
			return
		}
		h.logger.Error("read slack interactive body failed", "error", err)
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid request body")
		return
	}

	if err := VerifySlackSignature(h.cfg.SigningSecret, body, r.Header); err != nil {
		WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "invalid slack signature")
		return
	}

	form, err := url.ParseQuery(string(body))
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid form body")
		return
	}

	rawPayload := form.Get("payload")
	if strings.TrimSpace(rawPayload) == "" {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "payload is required")
		return
	}

	var payload slackInteractivePayload
	if err := json.Unmarshal([]byte(rawPayload), &payload); err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid payload JSON")
		return
	}

	if payload.Type != "block_actions" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if len(payload.Actions) == 0 {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "action is required")
		return
	}

	ctx := r.Context()
	queries := db.New(h.pool)

	settings, err := queries.GetOrganizationSlackSettingsByWorkspaceID(ctx, pgtype.Text{
		String: payload.Team.ID,
		Valid:  payload.Team.ID != "",
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteAPIError(w, http.StatusNotFound, CodeNotFound, "slack workspace not linked")
			return
		}
		h.logger.Error("lookup slack workspace failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	userID, err := queries.GetUserIDBySlackUserID(ctx, db.GetUserIDBySlackUserIDParams{
		OrganizationID: settings.OrganizationID,
		SlackUserID:    payload.User.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteAPIError(w, http.StatusForbidden, CodeForbidden, "slack user not linked")
			return
		}
		h.logger.Error("lookup slack user failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	action := payload.Actions[0]
	buttonValue, err := parseSlackButtonValue(action.Value)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid button value")
		return
	}

	alertID, err := uuid.Parse(buttonValue.AlertID)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid alert id")
		return
	}

	alert, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: settings.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteAPIError(w, http.StatusNotFound, CodeNotFound, "alert not found")
			return
		}
		h.logger.Error("load alert failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	if !alert.IncidentID.Valid {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "alert is not linked to an incident")
		return
	}

	switch action.ActionID {
	case slackdm.ActionAck:
		if err := h.acknowledgeAlert(ctx, alertID, settings.OrganizationID, userID); err != nil {
			h.writeActionError(w, err)
			return
		}
	case slackdm.ActionEscalate:
		if err := h.reEscalateAlert(ctx, alertID, settings.OrganizationID); err != nil {
			h.writeActionError(w, err)
			return
		}
	default:
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "unsupported action")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *SlackInteractiveHandler) acknowledgeAlert(
	ctx context.Context,
	alertID, organizationID, acknowledgedBy uuid.UUID,
) error {
	if err := escalationapi.AcknowledgeAlert(ctx, h.pool, h.jobs, alertID, organizationID, acknowledgedBy); err != nil {
		return err
	}
	return nil
}

func (h *SlackInteractiveHandler) reEscalateAlert(
	ctx context.Context,
	alertID, organizationID uuid.UUID,
) error {
	return escalationapi.ReEscalateAlert(ctx, h.pool, h.jobs, alertID, organizationID)
}

func (h *SlackInteractiveHandler) writeActionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		WriteAPIError(w, http.StatusRequestTimeout, CodeInternal, "request timed out")
	case strings.Contains(err.Error(), "closed alert"):
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "cannot modify a closed alert")
	case strings.Contains(err.Error(), "already acknowledged"):
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "alert is already acknowledged")
	default:
		h.logger.Error("slack interactive action failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
	}
}

func parseSlackButtonValue(raw string) (slackButtonValue, error) {
	var value slackButtonValue
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return slackButtonValue{}, err
	}
	if strings.TrimSpace(value.AlertID) == "" {
		return slackButtonValue{}, errors.New("alert_id is required")
	}
	return value, nil
}

// BuildSlackInteractiveFormBody builds a signed Slack interactive request body for tests.
func BuildSlackInteractiveFormBody(payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return []byte("payload=" + url.QueryEscape(string(raw))), nil
}

// SlackInteractiveTimestamp returns a current Slack request timestamp string.
func SlackInteractiveTimestamp() string {
	return strconv.FormatInt(time.Now().UTC().Unix(), 10)
}
