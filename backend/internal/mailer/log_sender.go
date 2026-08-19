package mailer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LogSender stands in for real SMTP delivery when none is configured — it logs and
// records the attempt so auth/checkout flows remain testable end to end without an
// SMTP account. Never used when SMTP_HOST is set (see SMTPSender).
type LogSender struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

func NewLogSender(pool *pgxpool.Pool, log *slog.Logger) *LogSender {
	return &LogSender{pool: pool, log: log}
}

func (s *LogSender) Send(ctx context.Context, to, subject, template, body string) error {
	s.log.Info("email (dev stub, not actually sent)", "to", to, "subject", subject, "template", template, "body", body)
	return s.record(ctx, to, subject, template)
}

func (s *LogSender) SendOrderConfirmation(ctx context.Context, to, studentName, orderNumber string, tickets []TicketAttachment) error {
	labels := make([]string, len(tickets))
	for i, t := range tickets {
		labels[i] = t.Label
	}
	s.log.Info("email (dev stub, not actually sent)", "to", to, "template", "order_confirmation",
		"studentName", studentName, "orderNumber", orderNumber, "tickets", fmt.Sprintf("%v", labels))
	return s.record(ctx, to, "Sua experiência P5 Wellness Club está confirmada", "order_confirmation")
}

func (s *LogSender) SendRescheduleNotice(ctx context.Context, to, studentName, orderNumber, newDate string, tickets []TicketAttachment) error {
	labels := make([]string, len(tickets))
	for i, t := range tickets {
		labels[i] = t.Label
	}
	s.log.Info("email (dev stub, not actually sent)", "to", to, "template", "reschedule_notice",
		"studentName", studentName, "orderNumber", orderNumber, "newDate", newDate, "tickets", fmt.Sprintf("%v", labels))
	return s.record(ctx, to, "Sua data no P5 Wellness Club foi alterada", "reschedule_notice")
}

func (s *LogSender) SendWelcome(ctx context.Context, to, fullName string) error {
	s.log.Info("email (dev stub, not actually sent)", "to", to, "template", "welcome", "fullName", fullName)
	return s.record(ctx, to, "Bem-vindo(a) ao P5 Wellness Club", "welcome")
}

func (s *LogSender) record(ctx context.Context, to, subject, template string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO email_log (to_email, subject, template, status, sent_at)
		VALUES ($1, $2, $3, 'sent', now())
	`, to, subject, template)
	return err
}
