package smsprovider

import (
	"context"
	"errors"
)

// ErrNotConfigured is returned when no SMS/voice provider is available.
var ErrNotConfigured = errors.New("sms/voice provider is not configured")

// SMSParams holds outbound SMS delivery context.
type SMSParams struct {
	To      string
	Message string
}

// VoiceParams holds outbound voice call delivery context.
type VoiceParams struct {
	To      string
	Message string
}

// Provider sends SMS and voice notifications through a pluggable backend.
type Provider interface {
	Name() string
	SendSMS(ctx context.Context, params SMSParams) error
	MakeVoiceCall(ctx context.Context, params VoiceParams) error
}
