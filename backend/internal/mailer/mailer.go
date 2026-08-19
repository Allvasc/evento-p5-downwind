package mailer

import "context"

// TicketAttachment is one QR image to embed inline in the order confirmation email —
// one per benefit purchased, so a combo arrives with each of its QR codes clearly labeled.
type TicketAttachment struct {
	Label string
	PNG   []byte
}

// Sender delivers transactional e-mail. Two implementations exist: SMTPSender (real
// delivery, Hostgator in production / Gmail fallback in dev — selected by whether
// SMTP_HOST is configured) and LogSender (records to email_log + server log only, used
// when no SMTP is configured so auth/checkout flows stay testable without real e-mail).
type Sender interface {
	// Send delivers a simple text e-mail — used for password reset codes.
	Send(ctx context.Context, to, subject, template, body string) error
	// SendOrderConfirmation delivers the purchase confirmation with one embedded QR
	// image per benefit, each independently checked in by staff at the venue.
	SendOrderConfirmation(ctx context.Context, to, studentName, orderNumber string, tickets []TicketAttachment) error
	// SendRescheduleNotice tells the customer their order moved to a new date — sent
	// when staff reschedules a no-show, so they don't show up on the original day. The
	// QR tokens don't change, only their validity window, so the same tickets are
	// re-attached for convenience.
	SendRescheduleNotice(ctx context.Context, to, studentName, orderNumber, newDate string, tickets []TicketAttachment) error
	// SendWelcome delivers the registration confirmation, sent once right after a
	// student creates their account.
	SendWelcome(ctx context.Context, to, fullName string) error
}
