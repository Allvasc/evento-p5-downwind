package mailer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/gomail.v2"
)

// SMTPSender delivers real e-mail over SMTP. Point it at Hostgator (your domain, with
// SPF/DKIM/DMARC configured — see docs/EMAIL_DNS_SETUP.md) in production; Gmail works
// for local testing but is not meant for production volume (~500/day cap, and generic
// automated sends from a personal account are flagged by other providers more readily).
type SMTPSender struct {
	dialer        *gomail.Dialer
	from          string
	publicSiteURL string
	pool          *pgxpool.Pool
	log           *slog.Logger
}

func NewSMTPSender(host string, port int, user, pass, from, publicSiteURL string, pool *pgxpool.Pool, log *slog.Logger) *SMTPSender {
	return &SMTPSender{
		dialer:        gomail.NewDialer(host, port, user, pass),
		from:          from,
		publicSiteURL: publicSiteURL,
		pool:          pool,
		log:           log,
	}
}

func (s *SMTPSender) Send(ctx context.Context, to, subject, template, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	if err := s.dialer.DialAndSend(m); err != nil {
		s.recordFailure(ctx, to, subject, template, err)
		return err
	}
	return s.record(ctx, to, subject, template)
}

func (s *SMTPSender) SendOrderConfirmation(ctx context.Context, to, studentName, orderNumber string, tickets []TicketAttachment) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	subject := "Seu ingresso P5 DownWind Day está confirmado"
	m.SetHeader("Subject", subject)

	var html bytes.Buffer
	html.WriteString(fmt.Sprintf(`<div style="font-family:sans-serif;max-width:480px;margin:0 auto">
		<h1 style="color:#0a0f1a">Tudo certo, %s!</h1>
		<p style="color:#57677e">Seu pedido <strong>%s</strong> foi confirmado. Apresente o QR Code na entrada — ele é a sua vaga garantida no P5 DownWind Day.</p>`, studentName, orderNumber))

	for i, t := range tickets {
		cid := fmt.Sprintf("ticket%d", i)
		html.WriteString(fmt.Sprintf(`<div style="margin:24px 0;text-align:center;border:1px solid #dbe6ef;border-radius:16px;padding:16px">
			<p style="font-weight:600;color:#0a0f1a">%s</p>
			<img src="cid:%s" width="220" height="220" alt="QR Code" />
		</div>`, t.Label, cid))
		pngData := t.PNG
		m.Embed(fmt.Sprintf("%s.png", t.Label),
			gomail.SetCopyFunc(func(w io.Writer) error {
				_, err := w.Write(pngData)
				return err
			}),
			gomail.SetHeader(map[string][]string{"Content-ID": {"<" + cid + ">"}}),
		)
	}
	html.WriteString(`</div>`)
	m.SetBody("text/html", html.String())

	if err := s.dialer.DialAndSend(m); err != nil {
		s.recordFailure(ctx, to, subject, "order_confirmation", err)
		return err
	}
	return s.record(ctx, to, subject, "order_confirmation")
}

func (s *SMTPSender) SendRescheduleNotice(ctx context.Context, to, studentName, orderNumber, newDate string, tickets []TicketAttachment) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	subject := "Sua data no P5 DownWind Day foi alterada"
	m.SetHeader("Subject", subject)

	var html bytes.Buffer
	html.WriteString(fmt.Sprintf(`<div style="font-family:sans-serif;max-width:480px;margin:0 auto">
		<h1 style="color:#0a0f1a">Nova data confirmada, %s!</h1>
		<p style="color:#57677e">Seu pedido <strong>%s</strong> foi remarcado. Sua nova data é <strong>%s</strong>. Apresente o QR Code na entrada — ele é a sua vaga garantida no P5 DownWind Day.</p>`, studentName, orderNumber, formatBRDate(newDate)))

	for i, t := range tickets {
		cid := fmt.Sprintf("ticket%d", i)
		html.WriteString(fmt.Sprintf(`<div style="margin:24px 0;text-align:center;border:1px solid #dbe6ef;border-radius:16px;padding:16px">
			<p style="font-weight:600;color:#0a0f1a">%s</p>
			<img src="cid:%s" width="220" height="220" alt="QR Code" />
		</div>`, t.Label, cid))
		pngData := t.PNG
		m.Embed(fmt.Sprintf("%s.png", t.Label),
			gomail.SetCopyFunc(func(w io.Writer) error {
				_, err := w.Write(pngData)
				return err
			}),
			gomail.SetHeader(map[string][]string{"Content-ID": {"<" + cid + ">"}}),
		)
	}
	html.WriteString(`</div>`)
	m.SetBody("text/html", html.String())

	if err := s.dialer.DialAndSend(m); err != nil {
		s.recordFailure(ctx, to, subject, "reschedule_notice", err)
		return err
	}
	return s.record(ctx, to, subject, "reschedule_notice")
}

func (s *SMTPSender) SendWelcome(ctx context.Context, to, fullName string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	subject := "Bem-vindo(a) ao P5 DownWind Day"
	m.SetHeader("Subject", subject)

	html := fmt.Sprintf(`<div style="font-family:sans-serif;max-width:480px;margin:0 auto">
		<p style="font-family:monospace;font-size:11px;letter-spacing:0.1em;text-transform:uppercase;color:#0b63d6;margin-bottom:8px">P5 Kite House</p>
		<h1 style="color:#0a0f1a;margin-top:0">Bem-vindo(a), %s!</h1>
		<p style="color:#57677e;line-height:1.5">Seu cadastro foi criado com sucesso. Agora você já pode garantir sua vaga no percurso guiado até a Praia do Presídio — transporte, apoio completo e estrutura do início ao fim.</p>
		<p style="margin:28px 0">
			<a href="%s/comprar" style="background:#0b63d6;color:#fff;padding:12px 24px;border-radius:999px;text-decoration:none;font-weight:600;font-size:14px">Garantir minha vaga</a>
		</p>
		<p style="color:#57677e;font-size:13px">Depois da compra, o QR Code do seu ingresso chega por e-mail e também fica disponível no seu <a href="%s/portal" style="color:#0b63d6">portal do aluno</a>.</p>
	</div>`, fullName, s.publicSiteURL, s.publicSiteURL)
	m.SetBody("text/html", html)

	if err := s.dialer.DialAndSend(m); err != nil {
		s.recordFailure(ctx, to, subject, "welcome", err)
		return err
	}
	return s.record(ctx, to, subject, "welcome")
}

// formatBRDate renders a "YYYY-MM-DD" date param as dd/mm/yyyy for e-mail copy; falls
// back to the raw value if it doesn't parse rather than failing the whole send.
func formatBRDate(isoDate string) string {
	t, err := time.Parse("2006-01-02", isoDate)
	if err != nil {
		return isoDate
	}
	return t.Format("02/01/2006")
}

func (s *SMTPSender) record(ctx context.Context, to, subject, template string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO email_log (to_email, subject, template, status, sent_at)
		VALUES ($1, $2, $3, 'sent', now())
	`, to, subject, template)
	return err
}

func (s *SMTPSender) recordFailure(ctx context.Context, to, subject, template string, sendErr error) {
	s.log.Error("smtp send failed", "to", to, "error", sendErr)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO email_log (to_email, subject, template, status, error_message, attempts)
		VALUES ($1, $2, $3, 'failed', $4, 1)
	`, to, subject, template, sendErr.Error())
	if err != nil {
		s.log.Error("record email failure", "error", err)
	}
}
