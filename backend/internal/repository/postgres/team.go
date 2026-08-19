package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TeamMember struct {
	ID           string
	Name         string
	Email        string
	Phone        string
	PasswordHash string
	Role         string
	Active       bool
	VendorID     *string
	VendorName   string
}

type TeamRepository struct {
	pool *pgxpool.Pool
}

func NewTeamRepository(pool *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{pool: pool}
}

func (r *TeamRepository) FindByEmail(ctx context.Context, email string) (*TeamMember, error) {
	var t TeamMember
	err := r.pool.QueryRow(ctx, `
		SELECT t.id, t.name, t.email, COALESCE(t.phone, ''), t.password_hash, t.role, t.active, t.vendor_id, COALESCE(v.name, '')
		FROM team_members t
		LEFT JOIN vendors v ON v.id = t.vendor_id
		WHERE t.email = $1
	`, email).Scan(&t.ID, &t.Name, &t.Email, &t.Phone, &t.PasswordHash, &t.Role, &t.Active, &t.VendorID, &t.VendorName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *TeamRepository) FindByID(ctx context.Context, id string) (*TeamMember, error) {
	var t TeamMember
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, email, COALESCE(phone, ''), password_hash, role, active, vendor_id
		FROM team_members WHERE id = $1
	`, id).Scan(&t.ID, &t.Name, &t.Email, &t.Phone, &t.PasswordHash, &t.Role, &t.Active, &t.VendorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

type CreateTeamMemberParams struct {
	Name         string
	Email        string
	Phone        string
	PasswordHash string
	Role         string
	VendorID     *string
}

func (r *TeamRepository) Create(ctx context.Context, p CreateTeamMemberParams) (*TeamMember, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO team_members (name, email, phone, password_hash, role, vendor_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, p.Name, p.Email, p.Phone, p.PasswordHash, p.Role, p.VendorID).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &TeamMember{ID: id, Name: p.Name, Email: p.Email, Phone: p.Phone, Role: p.Role, Active: true, VendorID: p.VendorID}, nil
}

func (r *TeamRepository) ListAll(ctx context.Context) ([]TeamMember, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.name, t.email, COALESCE(t.phone, ''), t.role, t.active, t.vendor_id, COALESCE(v.name, '')
		FROM team_members t
		LEFT JOIN vendors v ON v.id = t.vendor_id
		ORDER BY t.created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]TeamMember, 0)
	for rows.Next() {
		var t TeamMember
		if err := rows.Scan(&t.ID, &t.Name, &t.Email, &t.Phone, &t.Role, &t.Active, &t.VendorID, &t.VendorName); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (r *TeamRepository) SetActive(ctx context.Context, id string, active bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE team_members SET active = $1 WHERE id = $2`, active, id)
	return err
}

func (r *TeamRepository) UpdateLastLogin(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE team_members SET last_login_at = now() WHERE id = $1`, id)
	return err
}

func (r *TeamRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE team_members SET password_hash = $1, updated_at = now() WHERE id = $2`, passwordHash, id)
	return err
}
