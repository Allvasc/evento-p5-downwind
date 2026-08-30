package postgres

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrVoucherNotFound = errors.New("voucher não encontrado")
var ErrVoucherNotAvailable = errors.New("voucher já foi usado ou cancelado")
var ErrVoucherHasRedemption = errors.New("voucher já foi resgatado, não é possível cancelar")

type VoucherRepository struct {
	pool *pgxpool.Pool
}

func NewVoucherRepository(pool *pgxpool.Pool) *VoucherRepository {
	return &VoucherRepository{pool: pool}
}

const voucherCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no 0/O/1/I — avoids misreads when typed by hand

func generateVoucherCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := make([]byte, 6)
	for i, v := range b {
		code[i] = voucherCodeAlphabet[int(v)%len(voucherCodeAlphabet)]
	}
	return "P5-" + string(code), nil
}

// Create makes a new voucher with a freshly generated code, retrying the rare random
// collision instead of ever surfacing it to the caller.
func (r *VoucherRepository) Create(ctx context.Context, name, companyName, createdBy string) (id, code string, err error) {
	for attempt := 0; attempt < 5; attempt++ {
		candidate, genErr := generateVoucherCode()
		if genErr != nil {
			return "", "", genErr
		}
		var newID string
		err = r.pool.QueryRow(ctx, `
			INSERT INTO vouchers (code, name, company_name, created_by)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, candidate, name, companyName, createdBy).Scan(&newID)
		if err == nil {
			return newID, candidate, nil
		}
		if !isUniqueViolation(err) {
			return "", "", err
		}
	}
	return "", "", fmt.Errorf("could not generate a unique voucher code after several attempts: %w", err)
}

type CreatedVoucher struct {
	ID   string `json:"id"`
	Code string `json:"code"`
}

// CreateBatch makes `count` vouchers sharing the same name/company, each with its own
// code — for handing a partner company a batch of courtesy codes at once instead of
// creating them one by one. Every voucher is independently single-use.
func (r *VoucherRepository) CreateBatch(ctx context.Context, name, companyName, createdBy string, count int) ([]CreatedVoucher, error) {
	created := make([]CreatedVoucher, 0, count)
	for i := 0; i < count; i++ {
		id, code, err := r.Create(ctx, name, companyName, createdBy)
		if err != nil {
			return created, err
		}
		created = append(created, CreatedVoucher{ID: id, Code: code})
	}
	return created, nil
}

type VoucherRow struct {
	ID                    string  `json:"id"`
	Code                  string  `json:"code"`
	Name                  string  `json:"name"`
	CompanyName           string  `json:"companyName"`
	Status                string  `json:"status"`
	CreatedByName         string  `json:"createdByName"`
	RedeemedByName        string  `json:"redeemedByName,omitempty"`
	StudentEmail          string  `json:"studentEmail,omitempty"`
	StudentPhone          string  `json:"studentPhone,omitempty"`
	StudentCpfLast4       string  `json:"studentCpfLast4,omitempty"`
	OrderNumber           string  `json:"orderNumber,omitempty"`
	ActivityTitle         string  `json:"activityTitle,omitempty"`
	SessionStartTime      *string `json:"sessionStartTime,omitempty"`
	EntitlementStatus     string  `json:"entitlementStatus,omitempty"`
	EntitlementUsedAt     *string `json:"entitlementUsedAt,omitempty"`
	EntitlementUsedByName string  `json:"entitlementUsedByName,omitempty"`
	// ActivityDate is the day of the class the voucher was actually redeemed for (the
	// resulting entitlement's valid_from) — distinct from RedeemedAt, which is when the
	// redemption itself happened. Prestação de contas needs both: when the courtesy was
	// used, and which event date it covered.
	ActivityDate *string `json:"activityDate,omitempty"`
	RedeemedAt   *string `json:"redeemedAt,omitempty"`
	CreatedAt    string  `json:"createdAt"`
}

func (r *VoucherRepository) List(ctx context.Context) ([]VoucherRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT v.id, v.code, v.name, v.company_name, v.status,
		       tm.name,
		       COALESCE(s.full_name, ''),
		       COALESCE(s.email, ''),
		       COALESCE(s.phone, ''),
		       COALESCE(s.cpf_last4, ''),
		       COALESCE(o.order_number, ''),
		       COALESCE((SELECT string_agg(DISTINCT COALESCE(a.title, 'Café da Manhã'), ', ')
		                 FROM order_items oi
		                 LEFT JOIN activities a ON a.id = oi.activity_id
		                 WHERE oi.order_id = v.order_id), ''),
		       (SELECT to_char(MIN(cs.starts_at), 'YYYY-MM-DD"T"HH24:MI:SSZ')
		        FROM order_items oi
		        JOIN class_sessions cs ON cs.id = oi.class_session_id
		        WHERE oi.order_id = v.order_id),
		       COALESCE((SELECT e.status FROM entitlements e WHERE e.order_id = v.order_id ORDER BY e.issued_at DESC LIMIT 1), ''),
		       (SELECT to_char(e.used_at, 'YYYY-MM-DD"T"HH24:MI:SSZ') FROM entitlements e WHERE e.order_id = v.order_id AND e.used_at IS NOT NULL ORDER BY e.used_at DESC LIMIT 1),
		       COALESCE((SELECT utm.name FROM entitlements e JOIN team_members utm ON utm.id = e.used_by WHERE e.order_id = v.order_id AND e.used_by IS NOT NULL LIMIT 1), ''),
		       (SELECT MIN(e.valid_from)::text FROM entitlements e WHERE e.order_id = v.order_id),
		       to_char(v.redeemed_at, 'YYYY-MM-DD"T"HH24:MI:SSZ'),
		       to_char(v.created_at, 'YYYY-MM-DD"T"HH24:MI:SSZ')
		FROM vouchers v
		JOIN team_members tm ON tm.id = v.created_by
		LEFT JOIN students s ON s.id = v.redeemed_by
		LEFT JOIN orders o ON o.id = v.order_id
		ORDER BY v.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]VoucherRow, 0)
	for rows.Next() {
		var v VoucherRow
		var sessionStartTime, entitlementUsedAt, activityDate, redeemedAt *string
		if err := rows.Scan(
			&v.ID, &v.Code, &v.Name, &v.CompanyName, &v.Status,
			&v.CreatedByName,
			&v.RedeemedByName,
			&v.StudentEmail,
			&v.StudentPhone,
			&v.StudentCpfLast4,
			&v.OrderNumber,
			&v.ActivityTitle,
			&sessionStartTime,
			&v.EntitlementStatus,
			&entitlementUsedAt,
			&v.EntitlementUsedByName,
			&activityDate,
			&redeemedAt,
			&v.CreatedAt,
		); err != nil {
			return nil, err
		}
		v.SessionStartTime = sessionStartTime
		v.EntitlementUsedAt = entitlementUsedAt
		v.ActivityDate = activityDate
		v.RedeemedAt = redeemedAt
		list = append(list, v)
	}
	return list, rows.Err()
}

// Cancel voids a voucher that hasn't been redeemed yet — e.g. it was handed out by
// mistake, or the company partnership ended.
func (r *VoucherRepository) Cancel(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE vouchers SET status = 'cancelled' WHERE id = $1 AND status = 'available'
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var status string
		if scanErr := r.pool.QueryRow(ctx, `SELECT status FROM vouchers WHERE id = $1`, id).Scan(&status); scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return ErrVoucherNotFound
			}
			return scanErr
		}
		if status == "cancelled" {
			return nil // already in the state the caller wanted
		}
		return ErrVoucherHasRedemption
	}
	return nil
}

// FindAvailableByCode is the read-only check used before showing the product/date
// picker — lets the redemption page fail fast on a bad code without claiming anything.
func (r *VoucherRepository) FindAvailableByCode(ctx context.Context, code string) (*VoucherRow, error) {
	var v VoucherRow
	err := r.pool.QueryRow(ctx, `
		SELECT v.id, v.code, v.name, v.company_name, v.status, tm.name, to_char(v.created_at, 'YYYY-MM-DD"T"HH24:MI:SSZ')
		FROM vouchers v JOIN team_members tm ON tm.id = v.created_by
		WHERE v.code = $1
	`, code).Scan(&v.ID, &v.Code, &v.Name, &v.CompanyName, &v.Status, &v.CreatedByName, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrVoucherNotFound
	}
	if err != nil {
		return nil, err
	}
	if v.Status != "available" {
		return nil, ErrVoucherNotAvailable
	}
	return &v, nil
}

// Claim atomically locks a voucher by code and flips it to 'used' — the first caller to
// reach this for a given code wins; every later attempt (double submit, someone else
// trying the same code) gets ErrVoucherNotAvailable. Called before the order itself
// exists so a voucher is never left half-claimed; Release below undoes this if anything
// downstream (booking, payment recording) fails.
func (r *VoucherRepository) Claim(ctx context.Context, code, studentID string) (voucherID string, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var id, status string
	err = tx.QueryRow(ctx, `SELECT id, status FROM vouchers WHERE code = $1 FOR UPDATE`, code).Scan(&id, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrVoucherNotFound
	}
	if err != nil {
		return "", err
	}
	if status != "available" {
		return "", ErrVoucherNotAvailable
	}

	if _, err := tx.Exec(ctx, `
		UPDATE vouchers SET status = 'used', redeemed_by = $1, redeemed_at = now() WHERE id = $2
	`, studentID, id); err != nil {
		return "", err
	}

	return id, tx.Commit(ctx)
}

// Release reverts a claimed voucher back to available — used when redemption fails
// after the voucher was claimed (e.g. the chosen turma just lost its last seat), so the
// customer doesn't lose their voucher over something that wasn't their fault.
func (r *VoucherRepository) Release(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE vouchers SET status = 'available', redeemed_by = NULL, redeemed_at = NULL WHERE id = $1
	`, id)
	return err
}

// LinkOrder records which order a redeemed voucher produced — best-effort: the voucher
// is already correctly marked used by Claim either way, this is only for admin visibility.
func (r *VoucherRepository) LinkOrder(ctx context.Context, id, orderID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE vouchers SET order_id = $1 WHERE id = $2`, orderID, id)
	return err
}
