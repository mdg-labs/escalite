package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	defaultInitialBackoff = 250 * time.Millisecond
	defaultMaxBackoff     = 10 * time.Second
)

// Listener maintains LISTEN subscriptions and publishes decoded events to a Hub.
type Listener struct {
	connString string
	hub        *Hub
	logger     *slog.Logger
	initialBackoff time.Duration
	maxBackoff     time.Duration
}

// NewListener returns a Postgres LISTEN worker for realtime channels.
func NewListener(connString string, hub *Hub, logger *slog.Logger) *Listener {
	if logger == nil {
		logger = slog.Default()
	}
	return &Listener{
		connString:     connString,
		hub:            hub,
		logger:         logger,
		initialBackoff: defaultInitialBackoff,
		maxBackoff:     defaultMaxBackoff,
	}
}

// Run listens for NOTIFY payloads until ctx is canceled.
func (l *Listener) Run(ctx context.Context) {
	backoff := l.initialBackoff
	for {
		if ctx.Err() != nil {
			return
		}

		err := l.listenOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			backoff = l.initialBackoff
			continue
		}

		l.logger.Warn("realtime listener disconnected", "error", err)
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		backoff = nextBackoff(backoff, l.maxBackoff)
	}
}

func (l *Listener) listenOnce(ctx context.Context) error {
	conn, err := pgx.Connect(ctx, l.connString)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close(context.WithoutCancel(ctx))

	for _, channel := range []string{ChannelAlerts, ChannelSchedules} {
		if _, err := conn.Exec(ctx, "LISTEN "+pgx.Identifier{channel}.Sanitize()); err != nil {
			return fmt.Errorf("listen %s: %w", channel, err)
		}
	}

	l.logger.Info("realtime listener connected")

	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return fmt.Errorf("wait for notification: %w", err)
		}
		if notification == nil {
			continue
		}
		l.handleNotification(notification.Channel, notification.Payload)
	}
}

func (l *Listener) handleNotification(channel, payload string) {
	switch channel {
	case ChannelAlerts:
		var raw struct {
			AlertID        uuid.UUID `json:"alert_id"`
			OrganizationID uuid.UUID `json:"organization_id"`
			Status         string    `json:"status"`
			Op             string    `json:"op"`
		}
		if err := json.Unmarshal([]byte(payload), &raw); err != nil {
			l.logger.Warn("invalid alert notify payload", "error", err, "payload", payload)
			return
		}
		l.hub.PublishAlert(AlertEvent{
			AlertID:        raw.AlertID,
			OrganizationID: raw.OrganizationID,
			Status:         raw.Status,
			Op:             raw.Op,
		})
	case ChannelSchedules:
		var raw struct {
			ScheduleID     uuid.UUID `json:"schedule_id"`
			OrganizationID uuid.UUID `json:"organization_id"`
			Table          string    `json:"table"`
			Op             string    `json:"op"`
		}
		if err := json.Unmarshal([]byte(payload), &raw); err != nil {
			l.logger.Warn("invalid schedule notify payload", "error", err, "payload", payload)
			return
		}
		l.hub.PublishSchedule(ScheduleEvent{
			ScheduleID:     raw.ScheduleID,
			OrganizationID: raw.OrganizationID,
			Table:          raw.Table,
			Op:             raw.Op,
		})
	default:
		l.logger.Warn("unexpected notify channel", "channel", channel)
	}
}

func nextBackoff(current, max time.Duration) time.Duration {
	next := current * 2
	if next > max {
		return max
	}
	return next
}
