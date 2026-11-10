package postgres

import (
	"context"
	"testing"
	"time"
)

func TestPasswordResetRepository(t *testing.T) {
	pool := testPool(t)
	studentID := mustInsertStudent(t, pool)

	repo := NewPasswordResetRepository(pool)
	ctx := context.Background()

	// 1. Initially, count recent is 0
	count, err := repo.CountRecent(ctx, OwnerStudent, studentID, 15*time.Minute)
	if err != nil {
		t.Fatalf("CountRecent error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected count 0, got %d", count)
	}

	// 2. Initially, no active token
	token, err := repo.FindActive(ctx, OwnerStudent, studentID)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v (token: %v)", err, token)
	}

	// 3. Create a reset token
	codeHash := "test_code_hash_123"
	err = repo.Create(ctx, OwnerStudent, studentID, codeHash, 15*time.Minute)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	// 4. Count recent should now be 1
	count, err = repo.CountRecent(ctx, OwnerStudent, studentID, 15*time.Minute)
	if err != nil {
		t.Fatalf("CountRecent error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}

	// 5. FindActive should return the token
	token, err = repo.FindActive(ctx, OwnerStudent, studentID)
	if err != nil {
		t.Fatalf("FindActive error: %v", err)
	}
	if token == nil {
		t.Fatal("expected token, got nil")
	}
	if token.CodeHash != codeHash {
		t.Fatalf("expected codeHash %s, got %s", codeHash, token.CodeHash)
	}
	if token.Attempts != 0 {
		t.Fatalf("expected 0 attempts, got %d", token.Attempts)
	}

	// 6. Register attempt
	err = repo.RegisterAttempt(ctx, token.ID)
	if err != nil {
		t.Fatalf("RegisterAttempt error: %v", err)
	}
	token, err = repo.FindActive(ctx, OwnerStudent, studentID)
	if err != nil {
		t.Fatalf("FindActive error: %v", err)
	}
	if token.Attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", token.Attempts)
	}

	// 7. Consume token
	err = repo.Consume(ctx, token.ID)
	if err != nil {
		t.Fatalf("Consume error: %v", err)
	}

	// 8. FindActive should now return ErrNotFound since consumed_at is not null
	token, err = repo.FindActive(ctx, OwnerStudent, studentID)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after consume, got %v (token: %v)", err, token)
	}
}
