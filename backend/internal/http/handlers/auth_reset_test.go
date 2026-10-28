package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"p5wellness/backend/internal/domain/auth"
	"p5wellness/backend/internal/mailer"
	"p5wellness/backend/internal/repository/postgres"
)

type memoryMailer struct {
	lastTo      string
	lastSubject string
	lastBody    string
}

func (m *memoryMailer) Send(ctx context.Context, to, subject, template, body string) error {
	m.lastTo = to
	m.lastSubject = subject
	m.lastBody = body
	return nil
}

func (m *memoryMailer) SendOrderConfirmation(ctx context.Context, to, studentName, orderNumber string, tickets []mailer.TicketAttachment) error {
	return nil
}

func (m *memoryMailer) SendRescheduleNotice(ctx context.Context, to, studentName, orderNumber, newDate string, tickets []mailer.TicketAttachment) error {
	return nil
}

func (m *memoryMailer) SendWelcome(ctx context.Context, to, fullName string) error {
	return nil
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	_ = godotenv.Load("../../.env")
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestStudentPasswordResetFlow(t *testing.T) {
	pool := testPool(t)
	students := postgres.NewStudentRepository(pool)
	resets := postgres.NewPasswordResetRepository(pool)
	issuer := auth.NewTokenIssuer("test-jwt-secret-very-secure-32b!")
	mail := &memoryMailer{}
	pepper := "test-pepper-12345"
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	handler := NewAuthStudentHandler(students, resets, issuer, mail, pepper, logger)

	// Create student
	email := fmt.Sprintf("student-%s@test.com", uuid.NewString()[:8])
	oldPassword := "oldpassword123"
	oldHash, err := auth.HashPassword(oldPassword, pepper)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	student, err := students.Create(context.Background(), postgres.CreateStudentParams{
		FullName:     "Aluno Teste Reset",
		Email:        email,
		PasswordHash: oldHash,
	})
	if err != nil {
		t.Fatalf("create student: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM students WHERE id = $1", student.ID)
	})

	// 1. Request password reset
	body, _ := json.Marshal(map[string]string{"email": email})
	req := httptest.NewRequest("POST", "/auth/student/request-password-reset", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.RequestPasswordReset(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if mail.lastTo != email {
		t.Fatalf("expected email to %s, got %s", email, mail.lastTo)
	}

	// Extract code from email body: "Use o código 123456 para redefinir..."
	re := regexp.MustCompile(`\b\d{6}\b`)
	code := re.FindString(mail.lastBody)
	if code == "" {
		t.Fatalf("could not find 6-digit code in body: %s", mail.lastBody)
	}

	// 2. Try reset with wrong code
	wrongBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"code":     "000000",
		"password": "newpassword123",
	})
	req = httptest.NewRequest("POST", "/auth/student/reset-password", bytes.NewReader(wrongBody))
	w = httptest.NewRecorder()
	handler.ResetPassword(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for wrong code, got %d: %s", w.Code, w.Body.String())
	}

	// 3. Try reset with short password
	shortPassBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"code":     code,
		"password": "short",
	})
	req = httptest.NewRequest("POST", "/auth/student/reset-password", bytes.NewReader(shortPassBody))
	w = httptest.NewRecorder()
	handler.ResetPassword(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for short password, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Reset with valid code and new password
	newPassword := "newpassword123"
	validBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"code":     code,
		"password": newPassword,
	})
	req = httptest.NewRequest("POST", "/auth/student/reset-password", bytes.NewReader(validBody))
	w = httptest.NewRecorder()
	handler.ResetPassword(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 5. Try reusing the same code (should fail because token is consumed)
	req = httptest.NewRequest("POST", "/auth/student/reset-password", bytes.NewReader(validBody))
	w = httptest.NewRecorder()
	handler.ResetPassword(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for reused code, got %d: %s", w.Code, w.Body.String())
	}

	// 6. Verify login with old password fails
	loginOldBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": oldPassword,
	})
	req = httptest.NewRequest("POST", "/auth/student/login", bytes.NewReader(loginOldBody))
	w = httptest.NewRecorder()
	handler.Login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with old password, got %d", w.Code)
	}

	// 7. Verify login with new password succeeds
	loginNewBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": newPassword,
	})
	req = httptest.NewRequest("POST", "/auth/student/login", bytes.NewReader(loginNewBody))
	w = httptest.NewRecorder()
	handler.Login(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with new password, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTeamPasswordResetFlow(t *testing.T) {
	pool := testPool(t)
	team := postgres.NewTeamRepository(pool)
	resets := postgres.NewPasswordResetRepository(pool)
	issuer := auth.NewTokenIssuer("test-jwt-secret-very-secure-32b!")
	mail := &memoryMailer{}
	pepper := "test-pepper-12345"
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	handler := NewAuthTeamHandler(team, resets, issuer, mail, pepper, logger)

	// Create team member
	email := fmt.Sprintf("team-%s@p5kitehouse.com", uuid.NewString()[:8])
	oldPassword := "teamoldpass123"
	oldHash, err := auth.HashPassword(oldPassword, pepper)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	member, err := team.Create(context.Background(), postgres.CreateTeamMemberParams{
		Name:         "Staff Teste Reset",
		Email:        email,
		Role:         "admin",
		PasswordHash: oldHash,
	})
	if err != nil {
		t.Fatalf("create team member: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM team_members WHERE id = $1", member.ID)
	})

	// 1. Request reset
	body, _ := json.Marshal(map[string]string{"email": email})
	req := httptest.NewRequest("POST", "/auth/team/request-password-reset", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.RequestPasswordReset(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	re := regexp.MustCompile(`\b\d{6}\b`)
	code := re.FindString(mail.lastBody)
	if code == "" {
		t.Fatalf("could not find code in body: %s", mail.lastBody)
	}

	// 2. Reset password
	newPassword := "teamnewpass123"
	validBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"code":     code,
		"password": newPassword,
	})
	req = httptest.NewRequest("POST", "/auth/team/reset-password", bytes.NewReader(validBody))
	w = httptest.NewRecorder()
	handler.ResetPassword(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 3. Verify login with old password fails
	loginOldBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": oldPassword,
	})
	req = httptest.NewRequest("POST", "/auth/team/login", bytes.NewReader(loginOldBody))
	w = httptest.NewRecorder()
	handler.Login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with old password, got %d", w.Code)
	}

	// 4. Verify login with new password succeeds
	loginNewBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": newPassword,
	})
	req = httptest.NewRequest("POST", "/auth/team/login", bytes.NewReader(loginNewBody))
	w = httptest.NewRecorder()
	handler.Login(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with new password, got %d: %s", w.Code, w.Body.String())
	}
}
