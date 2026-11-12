package mailer

import "context"

// EventInfo names the event an e-mail is about. The zero value is the main event (the
// one at "/"), so every e-mail about it keeps exactly the copy it had before events
// existed; any other event gets its own name and its own buy link.
type EventInfo struct {
	Name string // "" = main event
	Slug string // "" = main event; otherwise its page is /<slug>
}

// RegistrationInfo is the content of a form sign-up confirmation e-mail.
type RegistrationInfo struct {
	EventName        string
	DateLabel        string
	LocationLabel    string
	Rules            string // one rule per line
	Waitlist         bool
	WaitlistPosition int
	PagePath         string // "/<slug>"
}

// DisplayName is how the e-mail copy calls the event.
func (e EventInfo) DisplayName() string {
	if e.Name == "" {
		return "P5 DownWind Day"
	}
	return e.Name
}

// BuyPath is the site path where the event's tickets are sold.
func (e EventInfo) BuyPath() string {
	if e.Slug == "" {
		return "/comprar"
	}
	return "/" + e.Slug + "/comprar"
}

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
	SendOrderConfirmation(ctx context.Context, to, studentName, orderNumber string, event EventInfo, tickets []TicketAttachment) error
	// SendRescheduleNotice tells the customer their order moved to a new date — sent
	// when staff reschedules a no-show, so they don't show up on the original day. The
	// QR tokens don't change, only their validity window, so the same tickets are
	// re-attached for convenience.
	SendRescheduleNotice(ctx context.Context, to, studentName, orderNumber, newDate string, event EventInfo, tickets []TicketAttachment) error
	// SendEventRegistration confirms a sign-up made through an event page form (no
	// account, no payment) — or tells the person they're on the waitlist.
	SendEventRegistration(ctx context.Context, to, fullName string, info RegistrationInfo) error
	// SendWelcome delivers the registration confirmation, sent once right after a
	// student creates their account. event is the event whose page they signed up from.
	SendWelcome(ctx context.Context, to, fullName string, event EventInfo) error
}
