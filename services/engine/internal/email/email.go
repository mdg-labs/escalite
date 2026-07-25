package email

import (
	"context"
	"fmt"
	"mime"
	"net/smtp"
	"strings"
)

// Message is a single outbound email with plain-text and HTML bodies.
type Message struct {
	To       string
	Subject  string
	TextBody string
	HTMLBody string
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
	payload := buildMIMEMessage(from, msg.To, msg.Subject, msg.TextBody, msg.HTMLBody)

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	if s.cfg.Username == "" && s.cfg.Password == "" {
		return smtp.SendMail(addr, nil, from, []string{msg.To}, payload)
	}
	return smtp.SendMail(addr, auth, from, []string{msg.To}, payload)
}

func buildMIMEMessage(from, to, subject, textBody, htmlBody string) []byte {
	boundary := "escalite-boundary"
	encodedSubject := mime.QEncoding.Encode("utf-8", subject)

	var body strings.Builder
	body.WriteString("From: ")
	body.WriteString(from)
	body.WriteString("\r\nTo: ")
	body.WriteString(to)
	body.WriteString("\r\nSubject: ")
	body.WriteString(encodedSubject)
	body.WriteString("\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=")
	body.WriteString(boundary)
	body.WriteString("\r\n\r\n")

	writePart := func(contentType, partBody string) {
		body.WriteString("--")
		body.WriteString(boundary)
		body.WriteString("\r\nContent-Type: ")
		body.WriteString(contentType)
		body.WriteString("\r\nContent-Transfer-Encoding: 8bit\r\n\r\n")
		body.WriteString(partBody)
		body.WriteString("\r\n\r\n")
	}

	writePart("text/plain; charset=UTF-8", textBody)
	writePart("text/html; charset=UTF-8", htmlBody)

	body.WriteString("--")
	body.WriteString(boundary)
	body.WriteString("--\r\n")

	return []byte(body.String())
}
