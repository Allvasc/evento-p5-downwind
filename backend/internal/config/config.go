package config

import (
	"fmt"
	"os"
)

type Config struct {
	Env         string
	Port        string
	DatabaseURL string

	JWTSecret      string
	QRHMACSecret   string
	PasswordPepper string

	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	AsaasAPIKey       string
	AsaasBaseURL      string
	AsaasWebhookToken string

	PublicSiteURL  string
	PublicAdminURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		Env:               getEnv("APP_ENV", "development"),
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		QRHMACSecret:      os.Getenv("QR_HMAC_SECRET"),
		PasswordPepper:    os.Getenv("PASSWORD_PEPPER"),
		SMTPHost:          os.Getenv("SMTP_HOST"),
		SMTPPort:          getEnv("SMTP_PORT", "587"),
		SMTPUser:          os.Getenv("SMTP_USER"),
		SMTPPass:          os.Getenv("SMTP_PASS"),
		SMTPFrom:          getEnv("SMTP_FROM", "P5 Wellness Club <wellness@p5beachclub.com.br>"),
		AsaasAPIKey:       os.Getenv("ASAAS_API_KEY"),
		AsaasBaseURL:      getEnv("ASAAS_BASE_URL", "https://sandbox.asaas.com/api/v3"),
		AsaasWebhookToken: os.Getenv("ASAAS_WEBHOOK_TOKEN"),
		PublicSiteURL:     getEnv("PUBLIC_SITE_URL", "http://localhost:5173"),
		PublicAdminURL:    getEnv("PUBLIC_ADMIN_URL", "http://localhost:5174"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.Env == "production" {
		if cfg.JWTSecret == "" || cfg.QRHMACSecret == "" || cfg.PasswordPepper == "" {
			return nil, fmt.Errorf("JWT_SECRET, QR_HMAC_SECRET and PASSWORD_PEPPER are required in production")
		}
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "dev-insecure-jwt-secret"
	}
	if cfg.QRHMACSecret == "" {
		cfg.QRHMACSecret = "dev-insecure-qr-secret"
	}
	if cfg.PasswordPepper == "" {
		cfg.PasswordPepper = "dev-insecure-pepper"
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
