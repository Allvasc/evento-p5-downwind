package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WebhookRepository struct {
	pool *pgxpool.Pool
}

func NewWebhookRepository(pool *pgxpool.Pool) *WebhookRepository {
	return &WebhookRepository{pool: pool}
}

// RecordIfNew stores the dedupe hash for an inbound webhook delivery. It returns
// isNew=false when the same payload was already processed (e.g. Asaas retried the
// delivery), so callers can skip reprocessing without erroring.
func (r *WebhookRepository) RecordIfNew(ctx context.Context, provider, dedupeHash, eventType string, payload []byte) (isNew bool, err error) {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO webhook_events (provider, dedupe_hash, event_type, payload, processed_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (provider, dedupe_hash) DO NOTHING
	`, provider, dedupeHash, eventType, payload)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
