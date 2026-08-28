package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminReportsRepository struct {
	pool *pgxpool.Pool
}

func NewAdminReportsRepository(pool *pgxpool.Pool) *AdminReportsRepository {
	return &AdminReportsRepository{pool: pool}
}

type Attendee struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	OrderNumber string `json:"orderNumber"`
}

type SessionRoster struct {
	SessionID     string     `json:"sessionId"`
	ActivityTitle string     `json:"activityTitle"`
	StartsAt      string     `json:"startsAt"`
	Capacity      int        `json:"capacity"`
	Attendees     []Attendee `json:"attendees"`
}

// SessionsWithAttendees lists every upcoming scheduled turma with the names of
// everyone who paid for a seat in it — the roster staff needs to know who to expect at
// each specific date/time, independent of which product they bought it through. from/to
// bound cs.starts_at (the turma's own date) — either may be nil to leave that side open.
func (r *AdminReportsRepository) SessionsWithAttendees(ctx context.Context, from, to *time.Time) ([]SessionRoster, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT cs.id, a.title, to_char(cs.starts_at, 'YYYY-MM-DD"T"HH24:MI:SSZ'), cs.capacity,
		       COALESCE(s.full_name, ''), COALESCE(s.email, ''), COALESCE(o.order_number, '')
		FROM class_sessions cs
		JOIN activities a ON a.id = cs.activity_id
		LEFT JOIN order_items oi ON oi.class_session_id = cs.id
		LEFT JOIN orders o ON o.id = oi.order_id AND o.status = 'paid'
		LEFT JOIN students s ON s.id = o.student_id
		WHERE cs.status = 'scheduled'
		  AND ($1::timestamptz IS NULL OR cs.starts_at >= $1)
		  AND ($2::timestamptz IS NULL OR cs.starts_at < $2::timestamptz + INTERVAL '1 day')
		ORDER BY cs.starts_at, a.title, s.full_name
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]SessionRoster, 0)
	index := map[string]int{}
	for rows.Next() {
		var sessionID, activityTitle, startsAt, name, email, orderNumber string
		var capacity int
		if err := rows.Scan(&sessionID, &activityTitle, &startsAt, &capacity, &name, &email, &orderNumber); err != nil {
			return nil, err
		}
		i, ok := index[sessionID]
		if !ok {
			i = len(list)
			index[sessionID] = i
			list = append(list, SessionRoster{
				SessionID: sessionID, ActivityTitle: activityTitle, StartsAt: startsAt,
				Capacity: capacity, Attendees: []Attendee{},
			})
		}
		if orderNumber != "" {
			list[i].Attendees = append(list[i].Attendees, Attendee{Name: name, Email: email, OrderNumber: orderNumber})
		}
	}
	return list, rows.Err()
}

type ProductRoster struct {
	ProductID string     `json:"productId"`
	Title     string     `json:"title"`
	Buyers    []Attendee `json:"buyers"`
}

// ProductsWithBuyers lists every active product with the names of everyone who paid
// for it, independent of session/date — e.g. every "Aula Individual — Yoga" buyer in
// one place, across every turma. Products with no buyers yet still appear (empty list),
// and that stays true even when from/to narrow the buyer list to zero for a product —
// the date bound lives in the orders JOIN condition, not a WHERE clause, so it only
// prunes which purchases count as a "buyer" rather than hiding products outright.
func (r *AdminReportsRepository) ProductsWithBuyers(ctx context.Context, from, to *time.Time) ([]ProductRoster, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT p.display_order, p.id, p.title, COALESCE(o.id::text, ''), COALESCE(s.full_name, ''), COALESCE(s.email, ''), COALESCE(o.order_number, ''), o.created_at
		FROM products p
		LEFT JOIN order_items oi ON oi.product_id = p.id
		LEFT JOIN orders o ON o.id = oi.order_id AND o.status = 'paid'
		  AND ($1::timestamptz IS NULL OR o.created_at >= $1)
		  AND ($2::timestamptz IS NULL OR o.created_at < $2::timestamptz + INTERVAL '1 day')
		LEFT JOIN students s ON s.id = o.student_id
		WHERE p.active = true
		ORDER BY p.display_order, o.created_at
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type row struct {
		productID, title, orderID, name, email, orderNumber string
	}
	var scanned []row
	for rows.Next() {
		var r row
		var displayOrder int
		var createdAt any
		if err := rows.Scan(&displayOrder, &r.productID, &r.title, &r.orderID, &r.name, &r.email, &r.orderNumber, &createdAt); err != nil {
			return nil, err
		}
		scanned = append(scanned, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Preserve product display order (from the products table's own ORDER BY, applied
	// via a second pass keyed on first-seen order) while deduping nothing further —
	// SELECT DISTINCT already collapsed the one-row-per-order_item duplication.
	list := make([]ProductRoster, 0)
	index := map[string]int{}
	for _, r := range scanned {
		i, ok := index[r.productID]
		if !ok {
			i = len(list)
			index[r.productID] = i
			list = append(list, ProductRoster{ProductID: r.productID, Title: r.title, Buyers: []Attendee{}})
		}
		if r.orderNumber != "" {
			list[i].Buyers = append(list[i].Buyers, Attendee{Name: r.name, Email: r.email, OrderNumber: r.orderNumber})
		}
	}
	return list, nil
}

type ActivityRoster struct {
	Label     string     `json:"label"`
	Attendees []Attendee `json:"attendees"`
}

// EventAttendee is one paid benefit due on the event day, joined to the buyer's full
// contact record — phone, e-mail and CPF in the clear. It's the data a staff member
// needs on a printed clipboard to check people in by hand, so unlike the name+e-mail
// rosters above this is the complete personal record and is only ever exported (CSV) or
// shown on the single "Lista de presença" screen, behind the same admin/reports auth.
type EventAttendee struct {
	FullName    string `json:"fullName"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	CPF         string `json:"cpf"`         // 000.000.000-00, or "" when the buyer never provided one
	PurchasedAt string `json:"purchasedAt"` // dd/mm/aaaa hh:mm, Fortaleza — pagamento (cai pra criação se não houver)
	OrderNumber string `json:"orderNumber"`
	Benefit     string `json:"benefit"`   // "P5 DownWind Day", "Café da Manhã", "Yoga + HYROX"...
	SessionAt   string `json:"sessionAt"` // dd/mm/aaaa hh:mm da turma, ou "" (Café da Manhã não tem turma)
	EventDate   string `json:"eventDate"` // dd/mm/aaaa — dia do evento (entitlements.valid_until)
	VendorName  string `json:"vendorName"`
	CheckedIn   bool   `json:"checkedIn"`   // já validado no sistema (QR ou lista)
	CheckedInAt string `json:"checkedInAt"` // dd/mm/aaaa hh:mm, ou ""
}

// EventAttendees lists every paid, non-cancelled benefit whose event day (valid_until)
// falls in [from, to] — one row per entitlement, so a guest with a class + Café da Manhã
// shows up twice, once per QR. from/to are plain dates (either may be nil to leave that
// side open); with both nil it returns every paid benefit ever, which for a one-day event
// is exactly the full list. encryptionKey is the same pepper used to encrypt the CPF at
// registration (see student.go) — a wrong key would make pgp_sym_decrypt raise.
func (r *AdminReportsRepository) EventAttendees(ctx context.Context, from, to *time.Time, encryptionKey string) ([]EventAttendee, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.full_name,
		       COALESCE(s.phone, ''),
		       s.email,
		       CASE WHEN s.cpf_encrypted IS NOT NULL
		            THEN pgp_sym_decrypt(s.cpf_encrypted, $3)
		            ELSE '' END,
		       to_char(COALESCE(o.paid_at, o.created_at) AT TIME ZONE 'America/Fortaleza', 'DD/MM/YYYY HH24:MI'),
		       o.order_number,
		       string_agg(DISTINCT COALESCE(a.title, 'Café da Manhã'), ' + ' ORDER BY COALESCE(a.title, 'Café da Manhã')),
		       COALESCE(to_char(MIN(cs.starts_at) AT TIME ZONE 'America/Fortaleza', 'DD/MM/YYYY HH24:MI'), ''),
		       to_char(e.valid_until, 'DD/MM/YYYY'),
		       v.name,
		       e.status = 'used',
		       COALESCE(to_char(e.used_at AT TIME ZONE 'America/Fortaleza', 'DD/MM/YYYY HH24:MI'), '')
		FROM entitlements e
		JOIN orders o ON o.id = e.order_id AND o.status = 'paid'
		JOIN students s ON s.id = e.student_id
		JOIN vendors v ON v.id = e.vendor_id
		JOIN entitlement_items ei ON ei.entitlement_id = e.id
		JOIN order_items oi ON oi.id = ei.order_item_id
		LEFT JOIN activities a ON a.id = oi.activity_id
		LEFT JOIN class_sessions cs ON cs.id = oi.class_session_id
		WHERE e.status <> 'cancelled'
		  AND ($1::date IS NULL OR e.valid_until >= $1::date)
		  AND ($2::date IS NULL OR e.valid_until <= $2::date)
		GROUP BY e.id, s.full_name, s.phone, s.email, s.cpf_encrypted,
		         o.paid_at, o.created_at, o.order_number, e.valid_until, v.name, e.status, e.used_at
		ORDER BY e.valid_until, MIN(cs.starts_at) NULLS FIRST, s.full_name
	`, from, to, encryptionKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]EventAttendee, 0)
	for rows.Next() {
		var a EventAttendee
		var cpfDigits string
		if err := rows.Scan(&a.FullName, &a.Phone, &a.Email, &cpfDigits, &a.PurchasedAt, &a.OrderNumber,
			&a.Benefit, &a.SessionAt, &a.EventDate, &a.VendorName, &a.CheckedIn, &a.CheckedInAt); err != nil {
			return nil, err
		}
		a.CPF = formatCPF(cpfDigits)
		list = append(list, a)
	}
	return list, rows.Err()
}

// formatCPF turns the 11 stored digits into 000.000.000-00; anything else (empty, or an
// unexpected length) is returned untouched.
func formatCPF(digits string) string {
	if len(digits) != 11 {
		return digits
	}
	return digits[0:3] + "." + digits[3:6] + "." + digits[6:9] + "-" + digits[9:11]
}

// ByActivity collapses attendance to the underlying benefit itself — every Yoga booking
// in one place, every HYROX booking in another, every Café da Manhã in a third —
// independent of which product or specific turma/date it was bought through. A single
// combo purchase (e.g. "Aulas" = Yoga + HYROX) contributes one order_item per activity,
// so the same buyer can appear once under Yoga and once under HYROX. from/to bound
// o.created_at (the purchase date, since activities/order_items have no date of their own).
func (r *AdminReportsRepository) ByActivity(ctx context.Context, from, to *time.Time) ([]ActivityRoster, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT COALESCE(a.title, 'Café da Manhã') AS label,
		       COALESCE(s.full_name, ''), COALESCE(s.email, ''), COALESCE(o.order_number, '')
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id AND o.status = 'paid'
		JOIN students s ON s.id = o.student_id
		LEFT JOIN activities a ON a.id = oi.activity_id
		WHERE ($1::timestamptz IS NULL OR o.created_at >= $1)
		  AND ($2::timestamptz IS NULL OR o.created_at < $2::timestamptz + INTERVAL '1 day')
		ORDER BY label, s.full_name
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]ActivityRoster, 0)
	index := map[string]int{}
	for rows.Next() {
		var label, name, email, orderNumber string
		if err := rows.Scan(&label, &name, &email, &orderNumber); err != nil {
			return nil, err
		}
		i, ok := index[label]
		if !ok {
			i = len(list)
			index[label] = i
			list = append(list, ActivityRoster{Label: label, Attendees: []Attendee{}})
		}
		list[i].Attendees = append(list[i].Attendees, Attendee{Name: name, Email: email, OrderNumber: orderNumber})
	}
	return list, rows.Err()
}
