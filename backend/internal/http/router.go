package http

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5/pgxpool"

	"p5wellness/backend/internal/asaas"
	"p5wellness/backend/internal/config"
	"p5wellness/backend/internal/domain/auth"
	"p5wellness/backend/internal/http/handlers"
	appmw "p5wellness/backend/internal/http/middleware"
	"p5wellness/backend/internal/mailer"
	"p5wellness/backend/internal/repository/postgres"
)

func NewRouter(cfg *config.Config, pool *pgxpool.Pool, log *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.PublicSiteURL, cfg.PublicAdminURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	catalogRepo := postgres.NewCatalogRepository(pool)
	sessionRepo := postgres.NewSessionRepository(pool)
	publicCatalog := handlers.NewPublicCatalogHandler(catalogRepo, sessionRepo, log)

	studentRepo := postgres.NewStudentRepository(pool)
	teamRepo := postgres.NewTeamRepository(pool)
	resetRepo := postgres.NewPasswordResetRepository(pool)
	tokenIssuer := auth.NewTokenIssuer(cfg.JWTSecret)

	var emailSender mailer.Sender
	if cfg.SMTPHost != "" {
		port, err := strconv.Atoi(cfg.SMTPPort)
		if err != nil {
			port = 587
		}
		emailSender = mailer.NewSMTPSender(cfg.SMTPHost, port, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom, cfg.PublicSiteURL, pool, log)
		log.Info("mailer: SMTP configurado", "host", cfg.SMTPHost)
	} else {
		log.Warn("SMTP_HOST não configurado — e-mails apenas registrados em log/email_log (dev)")
		emailSender = mailer.NewLogSender(pool, log)
	}

	authStudent := handlers.NewAuthStudentHandler(studentRepo, resetRepo, tokenIssuer, emailSender, cfg.PasswordPepper, log)
	authTeam := handlers.NewAuthTeamHandler(teamRepo, resetRepo, tokenIssuer, emailSender, cfg.PasswordPepper, log)
	studentAreaRepo := postgres.NewStudentAreaRepository(pool)

	orderRepo := postgres.NewOrderRepository(pool)
	webhookRepo := postgres.NewWebhookRepository(pool)
	studentArea := handlers.NewStudentAreaHandler(studentRepo, studentAreaRepo, orderRepo, emailSender, cfg.PasswordPepper, log)

	var asaasClient asaas.Client
	var fakeAsaasClient *asaas.FakeClient
	if cfg.AsaasAPIKey != "" {
		asaasClient = asaas.NewHTTPClient(cfg.AsaasBaseURL, cfg.AsaasAPIKey, log)
	} else {
		log.Warn("ASAAS_API_KEY não configurada — usando cliente Asaas fake (apenas para desenvolvimento)")
		fakeAsaasClient = asaas.NewFakeClient()
		asaasClient = fakeAsaasClient
	}

	checkout := handlers.NewCheckoutHandler(catalogRepo, orderRepo, studentRepo, asaasClient, cfg.PasswordPepper, log)
	webhookAsaas := handlers.NewWebhookAsaasHandler(orderRepo, webhookRepo, emailSender, cfg.AsaasWebhookToken, cfg.QRHMACSecret, log)

	checkinRepo := postgres.NewCheckinRepository(pool)
	checkin := handlers.NewCheckinHandler(checkinRepo, cfg.QRHMACSecret, log)

	adminCatalogRepo := postgres.NewAdminCatalogRepository(pool)
	vendorRepo := postgres.NewVendorRepository(pool)
	adminCatalog := handlers.NewAdminCatalogHandler(adminCatalogRepo, vendorRepo, log)
	adminTeam := handlers.NewAdminTeamHandler(teamRepo, cfg.PasswordPepper, log)
	dashboardRepo := postgres.NewDashboardRepository(pool)
	dashboard := handlers.NewAdminDashboardHandler(dashboardRepo, log)
	adminSessions := handlers.NewAdminSessionsHandler(sessionRepo, log)

	adminCustomerRepo := postgres.NewAdminCustomerRepository(pool)
	adminCustomers := handlers.NewAdminCustomersHandler(adminCustomerRepo, studentRepo, log)
	adminOrderRepo := postgres.NewAdminOrderRepository(pool)
	adminOrders := handlers.NewAdminOrdersHandler(adminOrderRepo, orderRepo, asaasClient, emailSender, log)
	adminReportsRepo := postgres.NewAdminReportsRepository(pool)
	adminReports := handlers.NewAdminReportsHandler(adminReportsRepo, log)

	authRateLimit := httprate.LimitByIP(10, time.Minute)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api/v1", func(api chi.Router) {
		api.Route("/public", func(pub chi.Router) {
			pub.Get("/products", publicCatalog.ListProducts)
			pub.Get("/activities/{activityId}/sessions", publicCatalog.ListActivitySessions)
			pub.Get("/next-sessions", publicCatalog.NextSessions)
		})

		api.Route("/auth/student", func(a chi.Router) {
			a.Use(authRateLimit)
			a.Post("/register", authStudent.Register)
			a.Post("/login", authStudent.Login)
			a.Post("/request-password-reset", authStudent.RequestPasswordReset)
			a.Post("/reset-password", authStudent.ResetPassword)
		})

		api.Route("/auth/team", func(a chi.Router) {
			a.Use(authRateLimit)
			a.Post("/login", authTeam.Login)
			a.Post("/request-password-reset", authTeam.RequestPasswordReset)
			a.Post("/reset-password", authTeam.ResetPassword)
		})

		api.Route("/me", func(me chi.Router) {
			me.Use(appmw.RequireAuth(tokenIssuer, auth.SubjectStudent))
			me.Get("/", studentArea.Me)
			me.Put("/cpf", studentArea.UpdateCPF)
			me.Get("/orders", studentArea.ListOrders)
			me.With(authRateLimit).Post("/orders/{id}/resend-email", studentArea.ResendEmail)
			me.Get("/tickets", studentArea.ListTickets)
			me.Get("/tickets/{id}/qrcode.png", studentArea.TicketQRCode)
		})

		api.Route("/checkout", func(co chi.Router) {
			co.Use(appmw.RequireAuth(tokenIssuer, auth.SubjectStudent))
			co.Post("/", checkout.Create)
			co.Get("/orders/{id}/status", checkout.Status)
		})

		api.Post("/webhooks/asaas", webhookAsaas.Handle)

		api.Route("/checkin", func(ci chi.Router) {
			ci.Use(appmw.RequireAuth(tokenIssuer, auth.SubjectTeam, "admin", "staff", "reports"))
			ci.Post("/validate", checkin.Validate)
			ci.Post("/validate/batch", checkin.ValidateBatch)
			ci.Post("/validate/by-id", checkin.ValidateByID)
			ci.Get("/roster", checkin.Roster)
			ci.Get("/roster/dates", checkin.RosterDates)
		})

		api.Route("/admin", func(ad chi.Router) {
			// Config do admin (catálogo, equipe, clientes, e as ações de pedido que mexem
			// em dinheiro/e-mail) — role "admin" apenas. A role "reports" fica de fora.
			ad.Group(func(cfg chi.Router) {
				cfg.Use(appmw.RequireAuth(tokenIssuer, auth.SubjectTeam, "admin"))

				cfg.Get("/vendors", adminCatalog.ListVendors)

				cfg.Get("/activities", adminCatalog.ListActivities)
				cfg.Post("/activities", adminCatalog.CreateActivity)
				cfg.Put("/activities/{id}", adminCatalog.UpdateActivity)
				cfg.Patch("/activities/{id}/active", adminCatalog.SetActivityActive)

				cfg.Get("/products", adminCatalog.ListProducts)
				cfg.Post("/products", adminCatalog.CreateProduct)
				cfg.Put("/products/{id}", adminCatalog.UpdateProduct)
				cfg.Patch("/products/{id}/active", adminCatalog.SetProductActive)

				cfg.Get("/team", adminTeam.List)
				cfg.Post("/team", adminTeam.Create)
				cfg.Patch("/team/{id}/active", adminTeam.SetActive)

				cfg.Get("/sessions", adminSessions.List)
				cfg.Post("/sessions", adminSessions.Create)
				cfg.Put("/sessions/{id}", adminSessions.Update)
				cfg.Patch("/sessions/{id}/status", adminSessions.SetStatus)

				cfg.Get("/customers", adminCustomers.Search)
				cfg.Get("/customers/{id}", adminCustomers.Detail)
				cfg.Post("/customers/{id}/anonymize", adminCustomers.Anonymize)

				cfg.Post("/orders/{id}/refund", adminOrders.Refund)
				cfg.Post("/orders/{id}/resend-email", adminOrders.ResendEmail)
			})

			// Gráficos, relatórios e a lista/detalhe de pedidos (só leitura) — role
			// "admin" ou "reports".
			ad.Group(func(rep chi.Router) {
				rep.Use(appmw.RequireAuth(tokenIssuer, auth.SubjectTeam, "admin", "reports"))

				rep.Get("/dashboard/summary", dashboard.Summary)
				rep.Get("/dashboard/sales-by-product", dashboard.SalesByProduct)
				rep.Get("/dashboard/sales-timeseries", dashboard.SalesTimeseries)

				rep.Get("/orders", adminOrders.List)
				rep.Get("/orders/{id}", adminOrders.Detail)

				rep.Get("/reports/sessions", adminReports.Sessions)
				rep.Get("/reports/products", adminReports.Products)
				rep.Get("/reports/activities", adminReports.Activities)
			})
		})

		if cfg.Env != "production" {
			devTools := handlers.NewDevToolsHandler(orderRepo, webhookAsaas, fakeAsaasClient, log)
			api.Route("/dev", func(dev chi.Router) {
				dev.Post("/simulate-payment/{id}", devTools.SimulatePaymentConfirmation)
			})
		}
	})

	return r
}
