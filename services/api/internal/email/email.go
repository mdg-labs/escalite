package email

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// Message is a single outbound email.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Sender delivers email messages.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// SMTPConfig holds outbound SMTP settings.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// SMTPSender sends mail via SMTP.
type SMTPSender struct {
	cfg SMTPConfig
}

// NewSMTPSender returns an SMTP-backed sender.
func NewSMTPSender(cfg SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

// Send delivers msg using SMTP AUTH when credentials are configured.
func (s *SMTPSender) Send(_ context.Context, msg Message) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	from := s.cfg.From

	var body strings.Builder
	body.WriteString("From: ")
	body.WriteString(from)
	body.WriteString("\r\nTo: ")
	body.WriteString(msg.To)
	body.WriteString("\r\nSubject: ")
	body.WriteString(msg.Subject)
	body.WriteString("\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n")
	body.WriteString(msg.Body)

	payload := []byte(body.String())
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)

	if s.cfg.Username == "" && s.cfg.Password == "" {
		return smtp.SendMail(addr, nil, from, []string{msg.To}, payload)
	}

	return smtp.SendMail(addr, auth, from, []string{msg.To}, payload)
}

// NoopSender discards messages without sending (dev default when SMTP is unset).
type NoopSender struct{}

func (NoopSender) Send(_ context.Context, _ Message) error {
	return nil
}

// RecordingSender stores sent messages for tests.
type RecordingSender struct {
	Messages []Message
}

// Send appends msg to the in-memory log.
func (r *RecordingSender) Send(_ context.Context, msg Message) error {
	r.Messages = append(r.Messages, msg)
	return nil
}
