package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func mustInsertFormEvent(t *testing.T, pool *pgxpool.Pool, capacity *int, waitlist bool) string {
	t.Helper()
	id := mustInsertEvent(t, pool, true)
	if _, err := pool.Exec(context.Background(), `
		UPDATE events SET registration_mode = 'form', form_capacity = $1, waitlist_enabled = $2 WHERE id = $3
	`, capacity, waitlist, id); err != nil {
		t.Fatalf("set form mode: %v", err)
	}
	// Runs before the event's own cleanup (LIFO), removing the rows that point at it.
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM event_registrations WHERE event_id = $1`, id)
		_, _ = pool.Exec(context.Background(), `DELETE FROM event_page_views WHERE event_id = $1`, id)
	})
	return id
}

func newReg(email string) NewRegistration {
	return NewRegistration{
		FullName: "Pessoa Teste", WhatsApp: "85999990000", Email: email,
		PracticeLevel: "intermediario", Interest: "ambos", ConsentText: "consentimento de teste",
	}
}

func testEmail() string { return fmt.Sprintf("inscricao-%s@p5.test", uuid.NewString()[:8]) }

func TestRegistrations_CapacityThenWaitlist(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewEventRegistrationRepository(pool)
	capacity := 2
	eventID := mustInsertFormEvent(t, pool, &capacity, true)

	for i := 0; i < 2; i++ {
		res, err := repo.Register(ctx, eventID, newReg(testEmail()))
		if err != nil || res.Status != RegistrationConfirmed {
			t.Fatalf("registration %d: %+v, %v — want confirmed", i+1, res, err)
		}
	}
	res, err := repo.Register(ctx, eventID, newReg(testEmail()))
	if err != nil || res.Status != RegistrationWaitlist || res.WaitlistPosition != 1 {
		t.Fatalf("3rd registration: %+v, %v — want waitlist position 1", res, err)
	}

	stats, err := repo.Stats(ctx, eventID)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Confirmed != 2 || stats.Waitlist != 1 {
		t.Fatalf("stats = %+v, want 2 confirmed / 1 waitlist", stats)
	}
}

func TestRegistrations_DuplicateEmailRefused(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewEventRegistrationRepository(pool)
	eventID := mustInsertFormEvent(t, pool, nil, true)

	email := testEmail()
	if _, err := repo.Register(ctx, eventID, newReg(email)); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := repo.Register(ctx, eventID, newReg(email)); !errors.Is(err, ErrRegistrationDuplicate) {
		t.Fatalf("second with the same e-mail: got %v, want ErrRegistrationDuplicate", err)
	}
}

func TestRegistrations_FullWithoutWaitlist(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewEventRegistrationRepository(pool)
	capacity := 1
	eventID := mustInsertFormEvent(t, pool, &capacity, false)

	if _, err := repo.Register(ctx, eventID, newReg(testEmail())); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := repo.Register(ctx, eventID, newReg(testEmail())); !errors.Is(err, ErrRegistrationFull) {
		t.Fatalf("over capacity without waitlist: got %v, want ErrRegistrationFull", err)
	}
}

// The main event sells tickets: its page never takes form sign-ups.
func TestRegistrations_TicketEventRefusesForm(t *testing.T) {
	pool := testPool(t)
	repo := NewEventRegistrationRepository(pool)
	if _, err := repo.Register(context.Background(), DefaultEventID, newReg(testEmail())); !errors.Is(err, ErrRegistrationClosed) {
		t.Fatalf("form sign-up on the main event: got %v, want ErrRegistrationClosed", err)
	}
}

// Many people racing for the last spots: exactly `capacity` are confirmed.
func TestRegistrations_ConcurrentLastSpots(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewEventRegistrationRepository(pool)
	capacity := 3
	eventID := mustInsertFormEvent(t, pool, &capacity, true)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := repo.Register(ctx, eventID, newReg(testEmail())); err != nil {
				t.Errorf("Register: %v", err)
			}
		}()
	}
	wg.Wait()

	stats, err := repo.Stats(ctx, eventID)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Confirmed != capacity || stats.Waitlist != 10-capacity {
		t.Fatalf("stats = %+v, want %d confirmed / %d waitlist", stats, capacity, 10-capacity)
	}
}
