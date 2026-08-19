package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("already exists")

type Student struct {
	ID           string
	FullName     string
	Email        string
	Phone        string
	PasswordHash string
	CPFLast4     string
}

type StudentRepository struct {
	pool *pgxpool.Pool
}

func NewStudentRepository(pool *pgxpool.Pool) *StudentRepository {
	return &StudentRepository{pool: pool}
}

type CreateStudentParams struct {
	FullName      string
	Email         string
	Phone         string
	PasswordHash  string
	CPF           string // raw digits, may be empty
	CPFHash       string // empty when CPF is empty
	CPFLast4      string
	EncryptionKey string
}

func (r *StudentRepository) Create(ctx context.Context, p CreateStudentParams) (*Student, error) {
	var id string
	var cpfEncrypted any
	var cpfHash any
	var cpfLast4 any
	if p.CPF != "" {
		cpfHash = p.CPFHash
		cpfLast4 = p.CPFLast4
	}

	row := r.pool.QueryRow(ctx, `
		INSERT INTO students (full_name, email, phone, password_hash, cpf_encrypted, cpf_hash, cpf_last4)
		VALUES ($1, $2, $3, $4,
			CASE WHEN $5 <> '' THEN pgp_sym_encrypt($5, $6) ELSE NULL END,
			$7, $8)
		RETURNING id
	`, p.FullName, p.Email, p.Phone, p.PasswordHash, p.CPF, p.EncryptionKey, cpfHash, cpfLast4)

	if err := row.Scan(&id); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	_ = cpfEncrypted

	return &Student{ID: id, FullName: p.FullName, Email: p.Email, Phone: p.Phone, CPFLast4: p.CPFLast4}, nil
}

func (r *StudentRepository) FindByEmail(ctx context.Context, email string) (*Student, error) {
	var s Student
	err := r.pool.QueryRow(ctx, `
		SELECT id, full_name, email, phone, password_hash, COALESCE(cpf_last4, '')
		FROM students WHERE email = $1 AND deleted_at IS NULL
	`, email).Scan(&s.ID, &s.FullName, &s.Email, &s.Phone, &s.PasswordHash, &s.CPFLast4)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *StudentRepository) FindByID(ctx context.Context, id string) (*Student, error) {
	var s Student
	err := r.pool.QueryRow(ctx, `
		SELECT id, full_name, email, phone, password_hash, COALESCE(cpf_last4, '')
		FROM students WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&s.ID, &s.FullName, &s.Email, &s.Phone, &s.PasswordHash, &s.CPFLast4)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

// DecryptedCPF returns the student's CPF (digits only) in the clear, for the one moment
// it's actually needed: sending it to Asaas so a charge can be generated (Asaas requires
// CPF/CNPJ on the customer record). Returns "" if the student never provided one.
func (r *StudentRepository) DecryptedCPF(ctx context.Context, id, encryptionKey string) (string, error) {
	var cpf *string
	err := r.pool.QueryRow(ctx, `
		SELECT CASE WHEN cpf_encrypted IS NOT NULL THEN pgp_sym_decrypt(cpf_encrypted, $2) ELSE NULL END
		FROM students WHERE id = $1
	`, id, encryptionKey).Scan(&cpf)
	if err != nil {
		return "", err
	}
	if cpf == nil {
		return "", nil
	}
	return *cpf, nil
}

func (r *StudentRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE students SET password_hash = $1, updated_at = now() WHERE id = $2`, passwordHash, id)
	return err
}

// UpdateCPF sets or replaces a student's CPF — used when it wasn't provided at
// registration (checkout requires one) or needs correcting. Returns ErrConflict if the
// CPF is already registered to a different account.
func (r *StudentRepository) UpdateCPF(ctx context.Context, id, cpf, cpfHash, cpfLast4, encryptionKey string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE students
		SET cpf_encrypted = pgp_sym_encrypt($1, $2), cpf_hash = $3, cpf_last4 = $4, updated_at = now()
		WHERE id = $5
	`, cpf, encryptionKey, cpfHash, cpfLast4, id)
	if isUniqueViolation(err) {
		return ErrConflict
	}
	return err
}

// Anonymize scrubs a student's personal data in place (LGPD art. 18 deletion request),
// keeping the row itself so historical orders/payments remain intact for fiscal
// retention. The account can no longer log in or be found by e-mail/CPF afterwards.
func (r *StudentRepository) Anonymize(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE students SET
			full_name = 'Cliente removido',
			email = ('removido-' || substr(id::text, 1, 8) || '@anonimizado.p5wellness')::citext,
			phone = NULL,
			cpf_encrypted = NULL,
			cpf_hash = NULL,
			cpf_last4 = NULL,
			password_hash = 'anonymized',
			asaas_customer_id = NULL,
			deleted_at = now(),
			updated_at = now()
		WHERE id = $1
	`, id)
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
