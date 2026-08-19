package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// testPool connects to a real Postgres for integration tests that need transactional/
// concurrency guarantees no mock could exercise (e.g. two goroutines racing on the same
// row). It's skipped, not failed, when no database is configured — so `go test ./...`
// stays runnable without a live Postgres, while `go test ./...` run locally with
// backend/.env present (or DATABASE_URL exported) actually exercises them.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	_ = godotenv.Load("../../../.env")
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set — skipping integration test (see backend/.env)")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
