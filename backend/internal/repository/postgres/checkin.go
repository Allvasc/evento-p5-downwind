package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CheckinRepository struct {
	pool *pgxpool.Pool
}

func NewCheckinRepository(pool *pgxpool.Pool) *CheckinRepository {
	return &CheckinRepository{pool: pool}
}

type ValidationResult string

const (
	ResultSuccess          ValidationResult = "success"
	ResultAlreadyUsed      ValidationResult = "already_used"
	ResultExpired          ValidationResult = "expired"
	ResultNotFound         ValidationResult = "not_found"
	ResultInvalidSignature ValidationResult = "invalid_signature"
	ResultWrongSector      ValidationResult = "wrong_sector"
)

type ValidatedTicket struct {
	Result        ValidationResult
	EntitlementID string
	Label         string
	VendorName    string
	StudentName   string
	OrderNumber   string
	UsedAt        *time.Time
}

// ValidateToken performs an atomic conditional update: only a ticket that is currently
// `available`, not expired, and belonging to the scanning staff's own sector flips to
// `used`. Two staff devices racing on the same ticket: exactly one UPDATE affects a row,
// the other sees zero rows affected and is reported as already_used.
//
// scannerVendorID is the team member's own sector; empty means unrestricted (P5 admin).
func (r *CheckinRepository) ValidateToken(ctx context.Context, token, scannerVendorID, scannedBy, clientScanID string, deviceScannedAt time.Time) (*ValidatedTicket, error) {
	return r.validateEntitlement(ctx, "qr_token", token, scannerVendorID, scannedBy, clientScanID, deviceScannedAt)
}

// ValidateByID is the list-based fallback to ValidateToken: staff pick the guest by name
// off the paid roster (ListRoster) instead of scanning their QR code, and this validates
// the same entitlement row by its own id. Same atomicity, sector, and expiry rules apply —
// it's just a different way to find the row.
func (r *CheckinRepository) ValidateByID(ctx context.Context, entitlementID, scannerVendorID, scannedBy, clientScanID string, deviceScannedAt time.Time) (*ValidatedTicket, error) {
	return r.validateEntitlement(ctx, "id", entitlementID, scannerVendorID, scannedBy, clientScanID, deviceScannedAt)
}

// lookupColumn is always one of the literal strings above — never caller input — so
// building the query with it isn't a SQL-injection risk.
func (r *CheckinRepository) validateEntitlement(ctx context.Context, lookupColumn, lookupValue, scannerVendorID, scannedBy, clientScanID string, deviceScannedAt time.Time) (*ValidatedTicket, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var entitlementID, studentID, orderNumber, currentStatus, vendorID, vendorName string
	var validUntil time.Time
	var existingUsedAt *time.Time
	err = tx.QueryRow(ctx, `
		SELECT e.id, e.student_id, o.order_number, e.status, e.vendor_id, v.name, e.valid_until, e.used_at
		FROM entitlements e
		JOIN orders o ON o.id = e.order_id
		JOIN vendors v ON v.id = e.vendor_id
		WHERE e.`+lookupColumn+` = $1
	`, lookupValue).Scan(&entitlementID, &studentID, &orderNumber, &currentStatus, &vendorID, &vendorName, &validUntil, &existingUsedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		_ = r.logAttempt(ctx, tx, nil, scannedBy, clientScanID, ResultNotFound, deviceScannedAt)
		_ = tx.Commit(ctx)
		return &ValidatedTicket{Result: ResultNotFound}, nil
	}
	if err != nil {
		return nil, err
	}

	label := ticketLabel(ctx, tx, entitlementID)
	studentName := studentNameFor(ctx, tx, studentID)

	result := func() ValidationResult {
		if scannerVendorID != "" && scannerVendorID != vendorID {
			return ResultWrongSector
		}
		if currentStatus != "available" {
			return ResultAlreadyUsed
		}
		if validUntil.Before(truncateToDate(deviceScannedAt)) {
			return ResultExpired
		}
		return ResultSuccess
	}()

	usedAt := existingUsedAt
	if result == ResultSuccess {
		var newUsedAt time.Time
		err := tx.QueryRow(ctx, `
			UPDATE entitlements SET status = 'used', used_at = now(), used_by = $1
			WHERE id = $2 AND status = 'available'
			RETURNING used_at
		`, scannedBy, entitlementID).Scan(&newUsedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			result = ResultAlreadyUsed
			usedAt = existingUsedAt
		} else if err != nil {
			return nil, err
		} else {
			usedAt = &newUsedAt
		}
	}

	if err := r.logAttempt(ctx, tx, &entitlementID, scannedBy, clientScanID, result, deviceScannedAt); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &ValidatedTicket{
		Result: result, EntitlementID: entitlementID, Label: label, VendorName: vendorName,
		StudentName: studentName, OrderNumber: orderNumber, UsedAt: usedAt,
	}, nil
}

func (r *CheckinRepository) logAttempt(ctx context.Context, tx pgx.Tx, entitlementID *string, scannedBy, clientScanID string, result ValidationResult, deviceScannedAt time.Time) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO validation_log (entitlement_id, scanned_by, result, client_scan_id, device_scanned_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (client_scan_id) DO NOTHING
	`, entitlementID, scannedBy, string(result), clientScanID, deviceScannedAt)
	return err
}

func ticketLabel(ctx context.Context, tx pgx.Tx, entitlementID string) string {
	rows, err := tx.Query(ctx, `
		SELECT COALESCE(a.title, 'Café da Manhã')
		FROM entitlement_items ei
		JOIN order_items oi ON oi.id = ei.order_item_id
		LEFT JOIN activities a ON a.id = oi.activity_id
		WHERE ei.entitlement_id = $1
	`, entitlementID)
	if err != nil {
		return "Benefício"
	}
	defer rows.Close()

	var labels []string
	for rows.Next() {
		var l string
		if err := rows.Scan(&l); err == nil {
			labels = append(labels, l)
		}
	}
	if len(labels) == 0 {
		return "Benefício"
	}
	return strings.Join(labels, " + ")
}

func studentNameFor(ctx context.Context, tx pgx.Tx, studentID string) string {
	var name string
	_ = tx.QueryRow(ctx, `SELECT full_name FROM students WHERE id = $1`, studentID).Scan(&name)
	return name
}

type RosterEntry struct {
	EntitlementID string     `json:"entitlementId"`
	StudentName   string     `json:"studentName"`
	OrderNumber   string     `json:"orderNumber"`
	Status        string     `json:"status"`
	UsedAt        *time.Time `json:"usedAt,omitempty"`
}

type RosterGroup struct {
	Key        string        `json:"key"`
	Label      string        `json:"label"`
	VendorName string        `json:"vendorName"`
	Entries    []RosterEntry `json:"entries"`
}

// ListRoster is the data behind list-based check-in: everyone who paid for a class or
// breakfast on the given day, grouped by turma (or by "Café da Manhã" for breakfast-only
// items, which have no session), for staff to tap through by name instead of scanning a
// QR code. scannerVendorID scopes the list to the caller's own sector, same as
// ValidateToken; empty sees every sector (P5 admin).
//
// A combo entitlement covering two classes from the same vendor (one ticket, two
// entitlement_items) appears once in each class's group — validating it from either one
// marks the whole ticket used, matching what scanning its one QR code would do.
func (r *CheckinRepository) ListRoster(ctx context.Context, scannerVendorID, date string) ([]RosterGroup, error) {
	// nil (not "") signals "unrestricted" to Postgres — an empty string can't be compared
	// against the uuid column without an explicit, always-evaluated cast.
	var vendorFilter *string
	if scannerVendorID != "" {
		vendorFilter = &scannerVendorID
	}
	rows, err := r.pool.Query(ctx, `
		SELECT e.id, e.status, e.used_at, v.name,
		       s.full_name, o.order_number,
		       cs.id, cs.starts_at, COALESCE(a.title, 'Café da Manhã')
		FROM entitlements e
		JOIN orders o ON o.id = e.order_id AND o.status = 'paid'
		JOIN vendors v ON v.id = e.vendor_id
		JOIN students s ON s.id = e.student_id
		JOIN entitlement_items ei ON ei.entitlement_id = e.id
		JOIN order_items oi ON oi.id = ei.order_item_id
		LEFT JOIN class_sessions cs ON cs.id = oi.class_session_id
		LEFT JOIN activities a ON a.id = oi.activity_id
		WHERE e.valid_until = $1::date
		  AND ($2::uuid IS NULL OR e.vendor_id = $2::uuid)
		ORDER BY cs.starts_at NULLS LAST, s.full_name
	`, date, vendorFilter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var order []string
	groups := map[string]*RosterGroup{}
	for rows.Next() {
		var entitlementID, status, vendorName, studentName, orderNumber, itemTitle string
		var usedAt *time.Time
		var sessionID *string
		var startsAt *time.Time
		if err := rows.Scan(&entitlementID, &status, &usedAt, &vendorName, &studentName, &orderNumber, &sessionID, &startsAt, &itemTitle); err != nil {
			return nil, err
		}

		var key, label string
		if sessionID != nil {
			key = "session:" + *sessionID
			label = itemTitle + " — " + startsAt.In(checkinLocation).Format("02/01 15:04")
		} else {
			key = "breakfast:" + vendorName
			label = itemTitle
		}

		g, ok := groups[key]
		if !ok {
			g = &RosterGroup{Key: key, Label: label, VendorName: vendorName}
			groups[key] = g
			order = append(order, key)
		}
		g.Entries = append(g.Entries, RosterEntry{
			EntitlementID: entitlementID, StudentName: studentName, OrderNumber: orderNumber,
			Status: status, UsedAt: usedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	list := make([]RosterGroup, 0, len(order))
	for _, k := range order {
		list = append(list, *groups[k])
	}
	return list, nil
}

// ListRosterDates is what feeds the check-in list's date picker: every calendar day
// (Fortaleza) that actually has a registered turma, taken straight from class_sessions
// instead of hardcoded to "today" — so staff can pull up a different event's roster (the
// next Saturday, one that ran late, etc.) without guessing a date. Café da Manhã has no
// session of its own, but it always rides along with a class purchase (see order.go), so
// every one of its dates already shows up here too. scannerVendorID scopes it the same
// way as ListRoster; empty sees every sector's dates.
func (r *CheckinRepository) ListRosterDates(ctx context.Context, scannerVendorID string) ([]string, error) {
	var vendorFilter *string
	if scannerVendorID != "" {
		vendorFilter = &scannerVendorID
	}
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT to_char(cs.starts_at AT TIME ZONE 'America/Fortaleza', 'YYYY-MM-DD') AS d
		FROM class_sessions cs
		JOIN activities a ON a.id = cs.activity_id
		WHERE cs.status = 'scheduled'
		  AND ($1::uuid IS NULL OR a.vendor_id = $1::uuid)
		ORDER BY d
	`, vendorFilter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dates := make([]string, 0)
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		dates = append(dates, d)
	}
	return dates, rows.Err()
}

// checkinLocation is the calendar day the check-in happens in — the event is always in
// Fortaleza, so "expires at the end of the day" must mean the end of the day there, not
// in UTC. Devices report scan times in UTC (JS toISOString()), which without this would
// roll entitlements over to "expired" at 21:00 local time instead of midnight.
var checkinLocation = mustLoadLocation("America/Fortaleza")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

// TodayInEventTZ is the current calendar date in Fortaleza, formatted for the `date`
// columns/params used throughout check-in (valid_until, ListRoster) — the default day
// the check-in roster shows when the caller doesn't ask for a specific one.
func TodayInEventTZ() string {
	return time.Now().In(checkinLocation).Format("2006-01-02")
}

func truncateToDate(t time.Time) time.Time {
	// Find the calendar day in Fortaleza (where the check-in actually happens), but build
	// the result in UTC — matching how Postgres/pgx represent a plain `date` column (no
	// timezone, read back as UTC midnight) — so this stays comparable to validUntil.
	y, m, d := t.In(checkinLocation).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
