package handlers

import (
	"context"
	"fmt"
	"log/slog"

	qrcodelib "github.com/skip2/go-qrcode"

	"p5wellness/backend/internal/mailer"
	"p5wellness/backend/internal/repository/postgres"
)

// sendTicketEmail renders one QR PNG per entitlement and delivers the order confirmation
// e-mail. Shared by the webhook (first send, right after payment) and the resend
// endpoints (self-service and admin-triggered) so both paths produce identical e-mails.
func sendTicketEmail(ctx context.Context, sender mailer.Sender, log *slog.Logger, result postgres.PaidOrderResult) error {
	tickets := make([]mailer.TicketAttachment, 0, len(result.Entitlements))
	for _, e := range result.Entitlements {
		png, err := qrcodelib.Encode(e.QRToken, qrcodelib.Medium, 320)
		if err != nil {
			log.Error("encode qr for email", "error", err)
			continue
		}
		tickets = append(tickets, mailer.TicketAttachment{Label: fmt.Sprintf("%s (%s)", e.Label, e.VendorName), PNG: png})
	}
	return sender.SendOrderConfirmation(ctx, result.StudentEmail, result.StudentName, result.OrderNumber, tickets)
}

// sendRescheduleEmail is sendTicketEmail's counterpart for a reschedule: same QR
// attachments (the tokens don't change, only their validity window), but a notice
// pointing out the new date instead of the original confirmation copy.
func sendRescheduleEmail(ctx context.Context, sender mailer.Sender, log *slog.Logger, result postgres.PaidOrderResult, newDate string) error {
	tickets := make([]mailer.TicketAttachment, 0, len(result.Entitlements))
	for _, e := range result.Entitlements {
		png, err := qrcodelib.Encode(e.QRToken, qrcodelib.Medium, 320)
		if err != nil {
			log.Error("encode qr for reschedule email", "error", err)
			continue
		}
		tickets = append(tickets, mailer.TicketAttachment{Label: fmt.Sprintf("%s (%s)", e.Label, e.VendorName), PNG: png})
	}
	return sender.SendRescheduleNotice(ctx, result.StudentEmail, result.StudentName, result.OrderNumber, newDate, tickets)
}
