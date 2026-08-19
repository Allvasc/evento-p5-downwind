package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OwnerType string

const (
	OwnerStudent    OwnerType = "student"
	OwnerTeamMember OwnerType = "team_member"
)

type PasswordResetRepository struct {
	pool *pgxpool.Pool
}

func NewPasswordResetRepository(pool *pgxpool.Pool) *PasswordResetRepository {
	return &PasswordResetRepository{pool: pool}
}

// CountRecent returns how many reset tokens were requested for this owner in the
// given window, used to rate-limit repeated requests.
func (r *PasswordResetRepository) CountRecent(ctx context.Context, ownerType OwnerType, ownerID string, since time.Duration) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM password_reset_tokens
		WHERE owner_type = $1 AND owner_id = $2 AND created_at > now() - $3::interval
	`, ownerType, ownerID, since.String()).Scan(&count)
	return count, err
}

func (r *PasswordResetRepository) Create(ctx context.Context, ownerType OwnerType, ownerID, codeHash string, ttl time.Duration) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO password_reset_tokens (owner_type, owner_id, code_hash, expires_at)
		VALUES ($1, $2, $3, now() + $4::interval)
	`, ownerType, ownerID, codeHash, ttl.String())
	return err
}

type resetToken struct {
	ID       string
	OwnerID  string
	CodeHash string
	Attempts int
}

// FindActive returns the most recent, unconsumed, unexpired token for the owner.
func (r *PasswordResetRepository) FindActive(ctx context.Context, ownerType OwnerType, ownerID string) (*resetToken, error) {
	var t resetToken
	err := r.pool.QueryRow(ctx, `
		SELECT id, owner_id, code_hash, attempts FROM password_reset_tokens
		WHERE owner_type = $1 AND owner_id = $2 AND consumed_at IS NULL AND expires_at > now()
		ORDER BY created_at DESC LIMIT 1
	`, ownerType, ownerID).Scan(&t.ID, &t.OwnerID, &t.CodeHash, &t.Attempts)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *PasswordResetRepository) RegisterAttempt(ctx context.Context, tokenID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE password_reset_tokens SET attempts = attempts + 1 WHERE id = $1`, tokenID)
	return err
}

func (r *PasswordResetRepository) Consume(ctx context.Context, tokenID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE password_reset_tokens SET consumed_at = now() WHERE id = $1`, tokenID)
	return err
}
