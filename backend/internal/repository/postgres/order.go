package postgres

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"p5wellness/backend/internal/domain/product"
	"p5wellness/backend/internal/qrcode"
)

var ErrOrderNotPending = errors.New("order is not pending")
var ErrOrderNotPaid = errors.New("order is not paid")
var ErrSessionRequired = errors.New("session selection required for activity")
var ErrSessionFull = errors.New("session has no open seats")
var ErrActivityChoiceRequired = errors.New("choose one of the product's linked activities")
var ErrTicketAlreadyUsed = errors.New("um dos benefícios deste pedido já foi utilizado")
var ErrRescheduleDateInPast = errors.New("a nova data não pode ser no passado")
var ErrNoSessionOnDate = errors.New("no matching session for activity on the requested date")
var ErrNotNoShow = errors.New("este benefício não está com falta registrada")
var ErrNoShowRescheduleUsed = errors.New("a remarcação por falta já foi utilizada para este benefício")
var ErrNoNextSession = errors.New("não há próxima data cadastrada para esta atividade")
var ErrNoShowWindowExpired = errors.New("o prazo para remarcar por falta expirou")

type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

type CreatedOrder struct {
	ID          string
	OrderNumber string
	TotalCents  int
}

// CreateOrder creates the order and expands the product into order_items, one per
// activity plus one for breakfast when included. Every class-type activity requires a
// chosen session (sessionSelections: activityID -> class_session_id); capacity is locked
// and re-checked inside this same transaction so two customers racing for the last seat
// can't both succeed.
func (r *OrderRepository) CreateOrder(ctx context.Context, studentID string, p product.Product, sessionSelections map[string]string) (*CreatedOrder, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	orderNumber, err := generateOrderNumber()
	if err != nil {
		return nil, err
	}

	var orderID string
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (order_number, student_id, status, total_cents)
		VALUES ($1, $2, 'pending', $3)
		RETURNING id
	`, orderNumber, studentID, p.PriceCents).Scan(&orderID)
	if err != nil {
		return nil, err
	}

	if len(p.Activities) == 0 && !p.IncludesBreakfast {
		return nil, fmt.Errorf("product %s has no activities or breakfast benefit configured", p.ID)
	}

	bookActivity := func(act product.Activity, sessionID string) error {
		if sessionID == "" {
			return fmt.Errorf("%w: %s", ErrSessionRequired, act.Title)
		}

		var capacity, booked int
		err := tx.QueryRow(ctx, `
			SELECT capacity, (
				SELECT COUNT(*) FROM order_items oi
				JOIN orders o ON o.id = oi.order_id
				WHERE oi.class_session_id = cs.id AND o.status IN ('paid', 'pending')
			)
			FROM class_sessions cs
			WHERE cs.id = $1 AND cs.activity_id = $2 AND cs.status = 'scheduled'
			FOR UPDATE
		`, sessionID, act.ID).Scan(&capacity, &booked)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: turma inválida para %s", ErrSessionRequired, act.Title)
		}
		if err != nil {
			return err
		}
		if booked >= capacity {
			return fmt.Errorf("%w: %s", ErrSessionFull, act.Title)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO order_items (order_id, product_id, activity_id, class_session_id, benefit_type, unit_price_cents)
			VALUES ($1, $2, $3, $4, 'class', 0)
		`, orderID, p.ID, act.ID, sessionID)
		return err
	}

	if p.ChooseOneActivity {
		// The customer picks exactly one of the linked activities (e.g. "Yoga ou
		// HYROX") — take the first one with a matching, non-empty selection and
		// ignore the rest, rather than requiring a session for every activity.
		var chosen *product.Activity
		var chosenSessionID string
		for i := range p.Activities {
			if sessionID, ok := sessionSelections[p.Activities[i].ID]; ok && sessionID != "" {
				chosen = &p.Activities[i]
				chosenSessionID = sessionID
				break
			}
		}
		if chosen == nil {
			return nil, ErrActivityChoiceRequired
		}
		if err := bookActivity(*chosen, chosenSessionID); err != nil {
			return nil, err
		}
	} else {
		for _, act := range p.Activities {
			if err := bookActivity(act, sessionSelections[act.ID]); err != nil {
				return nil, err
			}
		}
	}
	if p.IncludesBreakfast {
		if _, err := tx.Exec(ctx, `
			INSERT INTO order_items (order_id, product_id, activity_id, benefit_type, unit_price_cents)
			VALUES ($1, $2, NULL, 'breakfast', 0)
		`, orderID, p.ID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &CreatedOrder{ID: orderID, OrderNumber: orderNumber, TotalCents: p.PriceCents}, nil
}

func (r *OrderRepository) SetAsaasPayment(ctx context.Context, orderID, paymentMethod, asaasPaymentID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE orders SET payment_method = $1, asaas_payment_id = $2, updated_at = now() WHERE id = $3
	`, paymentMethod, asaasPaymentID, orderID)
	return err
}

type IssuedEntitlement struct {
	ID         string
	QRToken    string
	Label      string // ex: "Yoga + HYROX" ou "Café da Manhã"
	VendorID   string
	VendorName string
}

type PaidOrderResult struct {
	OrderID      string
	OrderNumber  string
	StudentID    string
	StudentEmail string
	StudentName  string
	Entitlements []IssuedEntitlement
	AlreadyPaid  bool
}

// MarkPaidAndIssueEntitlements transitions a pending order to paid and issues one
// entitlement (QR ticket) per order_item, all inside a single transaction. It is safe to
// call more than once for the same order (e.g. a duplicate webhook delivery): if the
// order is no longer pending, it returns AlreadyPaid=true and does nothing further.
func (r *OrderRepository) MarkPaidAndIssueEntitlements(ctx context.Context, orderID, asaasPaymentID, billingType string, amountCents int, qrSecret string) (*PaidOrderResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var status, studentID, orderNumber string
	err = tx.QueryRow(ctx, `
		SELECT status, student_id, order_number FROM orders WHERE id = $1 FOR UPDATE
	`, orderID).Scan(&status, &studentID, &orderNumber)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if status != "pending" {
		return &PaidOrderResult{OrderID: orderID, AlreadyPaid: true}, tx.Commit(ctx)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE orders SET status = 'paid', paid_at = now(), asaas_payment_id = $1, updated_at = now() WHERE id = $2
	`, asaasPaymentID, orderID); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO payments (order_id, asaas_payment_id, asaas_customer_id, status, billing_type, amount_cents)
		VALUES ($1, $2, '', 'CONFIRMED', $3, $4)
		ON CONFLICT (asaas_payment_id) DO NOTHING
	`, orderID, asaasPaymentID, billingType, amountCents); err != nil {
		return nil, err
	}

	// Café da manhã não tem "atividade" própria — pertence sempre ao setor da P5.
	// A validade do ticket segue a data da turma escolhida (não mais "sempre hoje").
	rows, err := tx.Query(ctx, `
		SELECT oi.id, oi.benefit_type, COALESCE(a.title, 'Café da Manhã'),
		       COALESCE(a.vendor_id, (SELECT id FROM vendors WHERE slug = 'p5')) AS vendor_id,
		       COALESCE(v.name, (SELECT name FROM vendors WHERE slug = 'p5')) AS vendor_name,
		       cs.starts_at
		FROM order_items oi
		LEFT JOIN activities a ON a.id = oi.activity_id
		LEFT JOIN vendors v ON v.id = a.vendor_id
		LEFT JOIN class_sessions cs ON cs.id = oi.class_session_id
		WHERE oi.order_id = $1
	`, orderID)
	if err != nil {
		return nil, err
	}
	type item struct {
		id, benefitType, label, vendorID, vendorName string
		startsAt                                     *time.Time
	}
	var items []item
	var orderDate *time.Time
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.benefitType, &it.label, &it.vendorID, &it.vendorName, &it.startsAt); err != nil {
			rows.Close()
			return nil, err
		}
		if it.startsAt != nil && (orderDate == nil || it.startsAt.Before(*orderDate)) {
			orderDate = it.startsAt
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if orderDate == nil {
		now := time.Now()
		orderDate = &now
	}

	// Um QR por setor: se o pedido tem duas aulas da mesma empresa (ex: Yoga + HYROX,
	// ambas da AYO), elas viram um único ticket — validado de uma vez pela equipe daquele
	// setor. O café da manhã, por pertencer à P5, sempre sai como ticket separado.
	type group struct {
		vendorID, vendorName string
		labels               []string
		itemIDs              []string
		date                 time.Time
	}
	order := make([]string, 0, 2)
	groups := map[string]*group{}
	for _, it := range items {
		g, ok := groups[it.vendorID]
		if !ok {
			date := *orderDate
			if it.startsAt != nil {
				date = *it.startsAt
			}
			g = &group{vendorID: it.vendorID, vendorName: it.vendorName, date: date}
			groups[it.vendorID] = g
			order = append(order, it.vendorID)
		}
		g.labels = append(g.labels, it.label)
		g.itemIDs = append(g.itemIDs, it.id)
	}

	var issued []IssuedEntitlement
	for _, vendorID := range order {
		g := groups[vendorID]
		validDate := g.date.Format("2006-01-02")
		token, _, err := qrcode.GenerateToken(qrSecret)
		if err != nil {
			return nil, err
		}
		var entitlementID string
		err = tx.QueryRow(ctx, `
			INSERT INTO entitlements (order_id, student_id, vendor_id, qr_token, status, valid_from, valid_until)
			VALUES ($1, $2, $3, $4, 'available', $5::date, $5::date)
			RETURNING id
		`, orderID, studentID, g.vendorID, token, validDate).Scan(&entitlementID)
		if err != nil {
			return nil, err
		}
		for _, itemID := range g.itemIDs {
			if _, err := tx.Exec(ctx, `
				INSERT INTO entitlement_items (entitlement_id, order_item_id) VALUES ($1, $2)
			`, entitlementID, itemID); err != nil {
				return nil, err
			}
		}
		issued = append(issued, IssuedEntitlement{
			ID: entitlementID, QRToken: token,
			Label: strings.Join(g.labels, " + "), VendorID: g.vendorID, VendorName: g.vendorName,
		})
	}

	var studentEmail, studentName string
	if err := tx.QueryRow(ctx, `SELECT email, full_name FROM students WHERE id = $1`, studentID).Scan(&studentEmail, &studentName); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &PaidOrderResult{
		OrderID: orderID, OrderNumber: orderNumber, StudentID: studentID,
		StudentEmail: studentEmail, StudentName: studentName, Entitlements: issued,
	}, nil
}

// MarkFailed transitions a pending order straight to failed — used when creating the
// order succeeds but the Asaas customer/charge call right after it doesn't. Without
// this, an order stuck at "pending" keeps counting against its session's capacity
// forever (the booked-seats query counts paid AND pending orders), permanently
// squatting on a seat nobody actually paid for.
func (r *OrderRepository) MarkFailed(ctx context.Context, orderID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE orders SET status = 'failed', updated_at = now()
		WHERE id = $1 AND status = 'pending'
	`, orderID)
	return err
}

// MarkStatusByAsaasPaymentID updates an order's status by external payment id, but only
// when it is still pending — a terminal/paid order is never regressed by a late event.
func (r *OrderRepository) MarkStatusByAsaasPaymentID(ctx context.Context, asaasPaymentID, status string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE orders SET status = $1, updated_at = now()
		WHERE asaas_payment_id = $2 AND status = 'pending'
	`, status, asaasPaymentID)
	return err
}

func (r *OrderRepository) FindByID(ctx context.Context, orderID string) (asaasPaymentID string, status string, err error) {
	err = r.pool.QueryRow(ctx, `SELECT COALESCE(asaas_payment_id, ''), status FROM orders WHERE id = $1`, orderID).Scan(&asaasPaymentID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return
}

func (r *OrderRepository) FindOrderIDByAsaasPaymentID(ctx context.Context, asaasPaymentID string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `SELECT id FROM orders WHERE asaas_payment_id = $1`, asaasPaymentID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return id, err
}

// GetPaidOrderDetail re-reads the entitlements already issued for a paid order (does
// NOT issue new ones) — used to resend the confirmation e-mail on demand, either by the
// student themselves or by an admin on their behalf.
func (r *OrderRepository) GetPaidOrderDetail(ctx context.Context, orderID string) (*PaidOrderResult, error) {
	var status, studentID, orderNumber, studentEmail, studentName string
	err := r.pool.QueryRow(ctx, `
		SELECT o.status, o.student_id, o.order_number, s.email, s.full_name
		FROM orders o JOIN students s ON s.id = o.student_id
		WHERE o.id = $1
	`, orderID).Scan(&status, &studentID, &orderNumber, &studentEmail, &studentName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status != "paid" {
		return nil, ErrOrderNotPaid
	}

	rows, err := r.pool.Query(ctx, `
		SELECT e.id, e.qr_token, e.vendor_id, v.name
		FROM entitlements e JOIN vendors v ON v.id = e.vendor_id
		WHERE e.order_id = $1
		ORDER BY e.issued_at
	`, orderID)
	if err != nil {
		return nil, err
	}
	var entitlements []IssuedEntitlement
	ids := make([]string, 0)
	index := map[string]int{}
	for rows.Next() {
		var e IssuedEntitlement
		if err := rows.Scan(&e.ID, &e.QRToken, &e.VendorID, &e.VendorName); err != nil {
			rows.Close()
			return nil, err
		}
		index[e.ID] = len(entitlements)
		ids = append(ids, e.ID)
		entitlements = append(entitlements, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(ids) > 0 {
		labelRows, err := r.pool.Query(ctx, `
			SELECT ei.entitlement_id, COALESCE(a.title, 'Café da Manhã')
			FROM entitlement_items ei
			JOIN order_items oi ON oi.id = ei.order_item_id
			LEFT JOIN activities a ON a.id = oi.activity_id
			WHERE ei.entitlement_id = ANY($1)
		`, ids)
		if err != nil {
			return nil, err
		}
		labels := map[string][]string{}
		for labelRows.Next() {
			var entitlementID, label string
			if err := labelRows.Scan(&entitlementID, &label); err != nil {
				labelRows.Close()
				return nil, err
			}
			labels[entitlementID] = append(labels[entitlementID], label)
		}
		labelRows.Close()
		if err := labelRows.Err(); err != nil {
			return nil, err
		}
		for id, parts := range labels {
			entitlements[index[id]].Label = strings.Join(parts, " + ")
		}
	}

	return &PaidOrderResult{
		OrderID: orderID, OrderNumber: orderNumber, StudentID: studentID,
		StudentEmail: studentEmail, StudentName: studentName, Entitlements: entitlements,
	}, nil
}

// RefundOrder flips a paid order to refunded and cancels every entitlement that hasn't
// been used yet (an already-used ticket stays used — the visit happened, only the
// remaining, unused benefits of the order are voided). Safe to call once; the handler
// is responsible for not calling it on a non-paid order.
func (r *OrderRepository) RefundOrder(ctx context.Context, orderID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE orders SET status = 'refunded', updated_at = now() WHERE id = $1
	`, orderID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE entitlements SET status = 'cancelled' WHERE order_id = $1 AND status = 'available'
	`, orderID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

type RescheduleResult struct {
	OrderID      string
	PreviousDate string
	NewDate      string
}

// Reschedule moves every benefit of a paid order to a new Fortaleza calendar day — used
// when a customer misses the event and asks to attend on another date. The whole order
// moves together (matching the single-day-event model the public checkout already
// assumes): every class-type order_item gets rebooked into that activity's session on the
// new day, and every entitlement (including a breakfast one, which has no session of its
// own) has its valid_from/valid_until pushed to match. Blocked if any of the order's
// entitlements has already been used — the visit already happened, so there is nothing
// left to move.
func (r *OrderRepository) Reschedule(ctx context.Context, orderID, newDate, reason, changedBy string) (*RescheduleResult, error) {
	if newDate < TodayInEventTZ() {
		return nil, ErrRescheduleDateInPast
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1 FOR UPDATE`, orderID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if status != "paid" {
		return nil, ErrOrderNotPaid
	}

	var blocked int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM entitlements WHERE order_id = $1 AND status != 'available'
	`, orderID).Scan(&blocked); err != nil {
		return nil, err
	}
	if blocked > 0 {
		return nil, ErrTicketAlreadyUsed
	}

	var previousDate string
	if err := tx.QueryRow(ctx, `
		SELECT to_char(MIN(valid_from), 'YYYY-MM-DD') FROM entitlements WHERE order_id = $1
	`, orderID).Scan(&previousDate); err != nil {
		return nil, err
	}

	activityRows, err := tx.Query(ctx, `
		SELECT DISTINCT activity_id FROM order_items WHERE order_id = $1 AND benefit_type = 'class'
	`, orderID)
	if err != nil {
		return nil, err
	}
	var activityIDs []string
	for activityRows.Next() {
		var activityID string
		if err := activityRows.Scan(&activityID); err != nil {
			activityRows.Close()
			return nil, err
		}
		activityIDs = append(activityIDs, activityID)
	}
	activityRows.Close()
	if err := activityRows.Err(); err != nil {
		return nil, err
	}

	for _, activityID := range activityIDs {
		var sessionID string
		var capacity, booked int
		err := tx.QueryRow(ctx, `
			SELECT cs.id, cs.capacity, (
				SELECT COUNT(*) FROM order_items oi
				JOIN orders o ON o.id = oi.order_id
				WHERE oi.class_session_id = cs.id AND o.status IN ('paid', 'pending')
			)
			FROM class_sessions cs
			WHERE cs.activity_id = $1 AND cs.status = 'scheduled'
			  AND to_char(cs.starts_at AT TIME ZONE 'America/Fortaleza', 'YYYY-MM-DD') = $2
			ORDER BY cs.starts_at ASC
			LIMIT 1
			FOR UPDATE
		`, activityID, newDate).Scan(&sessionID, &capacity, &booked)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoSessionOnDate
		}
		if err != nil {
			return nil, err
		}
		if booked >= capacity {
			return nil, ErrSessionFull
		}

		if _, err := tx.Exec(ctx, `
			UPDATE order_items SET class_session_id = $1 WHERE order_id = $2 AND activity_id = $3 AND benefit_type = 'class'
		`, sessionID, orderID, activityID); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE entitlements SET valid_from = $1::date, valid_until = $1::date WHERE order_id = $2
	`, newDate, orderID); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO order_reschedules (order_id, changed_by, reason, previous_date, new_date)
		VALUES ($1, $2, $3, $4::date, $5::date)
	`, orderID, changedBy, reason, previousDate, newDate); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &RescheduleResult{OrderID: orderID, PreviousDate: previousDate, NewDate: newDate}, nil
}

type NoShowRescheduleResult struct {
	OrderID       string
	EntitlementID string
	PreviousDate  string
	NewDate       string
}

// noShowRescheduleWindowDays is how long, from the missed class date, the customer's one
// free no-show reschedule stays claimable — by staff in the admin panel or by the customer
// themselves in the student area. Past this many calendar days the right is gone, no matter
// how many sessions of the activity happened (or didn't) in between.
const noShowRescheduleWindowDays = 60

// pgxQuerier is the subset of pgxpool.Pool/pgx.Tx that entitlementClassActivityIDs needs —
// lets the same helper run either inside a transaction (FOR UPDATE flows) or as a plain
// pool query (read-only listing).
type pgxQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// entitlementClassActivityIDs returns the distinct activities (excluding breakfast, which
// has no session to reschedule) bundled into one entitlement's QR.
func entitlementClassActivityIDs(ctx context.Context, q pgxQuerier, entitlementID string) ([]string, error) {
	rows, err := q.Query(ctx, `
		SELECT DISTINCT oi.activity_id
		FROM entitlement_items ei
		JOIN order_items oi ON oi.id = ei.order_item_id
		WHERE ei.entitlement_id = $1 AND oi.benefit_type = 'class'
	`, entitlementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// addDaysISO adds days to a "YYYY-MM-DD" date and returns it in the same format.
func addDaysISO(dateStr string, days int) (string, error) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "", err
	}
	return t.AddDate(0, 0, days).Format("2006-01-02"), nil
}

// noShowEligibility loads and locks (FOR UPDATE — caller must be inside a transaction) the
// shared "1 remarcação grátis por falta" rule for an entitlement: it must still be
// 'available' past its own valid_until (nobody scanned the QR and the date has passed),
// must not have used this right before, and must be within noShowRescheduleWindowDays of
// the missed date. Shared by the staff-triggered flow (auto-picks the next session) and the
// student-triggered one (student picks a session) so both enforce identical eligibility. A
// non-empty studentID additionally requires the entitlement to belong to that student —
// used by the self-service flow so a customer can't act on someone else's ticket.
func noShowEligibility(ctx context.Context, tx pgx.Tx, entitlementID, studentID string) (orderID, missedDate string, err error) {
	var status, entStudentID string
	var noShowUsedAt *time.Time
	if err := tx.QueryRow(ctx, `
		SELECT order_id, student_id, status, valid_until::text, no_show_reschedule_used_at
		FROM entitlements WHERE id = $1 FOR UPDATE
	`, entitlementID).Scan(&orderID, &entStudentID, &status, &missedDate, &noShowUsedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrNotFound
		}
		return "", "", err
	}
	if studentID != "" && entStudentID != studentID {
		return "", "", ErrNotFound
	}
	today := TodayInEventTZ()
	if status != "available" || missedDate >= today {
		return "", "", ErrNotNoShow
	}
	if noShowUsedAt != nil {
		return "", "", ErrNoShowRescheduleUsed
	}
	deadline, err := addDaysISO(missedDate, noShowRescheduleWindowDays)
	if err != nil {
		return "", "", err
	}
	if today > deadline {
		return "", "", ErrNoShowWindowExpired
	}
	return orderID, missedDate, nil
}

// RescheduleNoShow is the staff-triggered flow (admin panel): it auto-picks the makeup date
// (the activity's next scheduled session) instead of letting staff choose one. Like
// Reschedule, every activity bundled into this entitlement's QR moves together onto that
// single makeup date.
func (r *OrderRepository) RescheduleNoShow(ctx context.Context, entitlementID, changedBy string) (*NoShowRescheduleResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	orderID, validUntil, err := noShowEligibility(ctx, tx, entitlementID, "")
	if err != nil {
		return nil, err
	}

	activityIDs, err := entitlementClassActivityIDs(ctx, tx, entitlementID)
	if err != nil {
		return nil, err
	}
	if len(activityIDs) == 0 {
		return nil, ErrNoNextSession
	}

	// A data da vaga de reposição é sempre a próxima sessão agendada da primeira atividade
	// do ingresso — condiz com o modelo de "evento de um dia" já assumido pelo Reschedule
	// manual: as demais atividades do mesmo QR precisam ter turma nessa mesma data.
	var newDate string
	if err := tx.QueryRow(ctx, `
		SELECT to_char(cs.starts_at AT TIME ZONE 'America/Fortaleza', 'YYYY-MM-DD')
		FROM class_sessions cs
		WHERE cs.activity_id = $1 AND cs.status = 'scheduled' AND cs.starts_at::date > $2::date
		ORDER BY cs.starts_at ASC
		LIMIT 1
	`, activityIDs[0], validUntil).Scan(&newDate); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoNextSession
		}
		return nil, err
	}

	for _, activityID := range activityIDs {
		var sessionID string
		var capacity, booked int
		err := tx.QueryRow(ctx, `
			SELECT cs.id, cs.capacity, (
				SELECT COUNT(*) FROM order_items oi
				JOIN orders o ON o.id = oi.order_id
				WHERE oi.class_session_id = cs.id AND o.status IN ('paid', 'pending')
			)
			FROM class_sessions cs
			WHERE cs.activity_id = $1 AND cs.status = 'scheduled'
			  AND to_char(cs.starts_at AT TIME ZONE 'America/Fortaleza', 'YYYY-MM-DD') = $2
			ORDER BY cs.starts_at ASC
			LIMIT 1
			FOR UPDATE
		`, activityID, newDate).Scan(&sessionID, &capacity, &booked)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoSessionOnDate
		}
		if err != nil {
			return nil, err
		}
		if booked >= capacity {
			return nil, ErrSessionFull
		}

		if _, err := tx.Exec(ctx, `
			UPDATE order_items SET class_session_id = $1
			WHERE benefit_type = 'class' AND activity_id = $2
			  AND id IN (SELECT order_item_id FROM entitlement_items WHERE entitlement_id = $3)
		`, sessionID, activityID, entitlementID); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE entitlements SET valid_from = $1::date, valid_until = $1::date, no_show_reschedule_used_at = now()
		WHERE id = $2
	`, newDate, entitlementID); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO order_reschedules (order_id, changed_by, reason, previous_date, new_date)
		VALUES ($1, $2, 'Remarcação automática por falta (não compareceu)', $3::date, $4::date)
	`, orderID, changedBy, validUntil, newDate); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &NoShowRescheduleResult{OrderID: orderID, EntitlementID: entitlementID, PreviousDate: validUntil, NewDate: newDate}, nil
}

type NoShowRescheduleOptions struct {
	Eligible        bool
	Reason          string
	MissedDate      string
	WindowExpiresAt string
	Dates           []string
}

// ListNoShowRescheduleOptions is the read side of the student self-service flow: it reports
// whether this entitlement (owned by studentID) currently qualifies for the free no-show
// reschedule and, if so, every calendar day — within the 60-day window — that has an open
// session for every activity bundled into the entitlement's QR (same "single-day-event"
// requirement RescheduleNoShow/Reschedule already enforce). Read-only, no row lock: the
// actual reschedule re-validates and locks for real.
func (r *OrderRepository) ListNoShowRescheduleOptions(ctx context.Context, entitlementID, studentID string) (*NoShowRescheduleOptions, error) {
	var status, missedDate string
	var noShowUsedAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT status, valid_until::text, no_show_reschedule_used_at
		FROM entitlements WHERE id = $1 AND student_id = $2
	`, entitlementID, studentID).Scan(&status, &missedDate, &noShowUsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	deadline, err := addDaysISO(missedDate, noShowRescheduleWindowDays)
	if err != nil {
		return nil, err
	}
	opts := &NoShowRescheduleOptions{MissedDate: missedDate, WindowExpiresAt: deadline}

	today := TodayInEventTZ()
	switch {
	case status != "available" || missedDate >= today:
		opts.Reason = "Este benefício não está com falta registrada."
		return opts, nil
	case noShowUsedAt != nil:
		opts.Reason = "Você já usou a remarcação por falta deste benefício."
		return opts, nil
	case today > deadline:
		opts.Reason = "O prazo de 60 dias para remarcar por falta expirou."
		return opts, nil
	}

	activityIDs, err := entitlementClassActivityIDs(ctx, r.pool, entitlementID)
	if err != nil {
		return nil, err
	}
	if len(activityIDs) == 0 {
		opts.Reason = "Não há atividade para remarcar neste benefício."
		return opts, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT to_char(cs.starts_at AT TIME ZONE 'America/Fortaleza', 'YYYY-MM-DD'), cs.activity_id
		FROM class_sessions cs
		WHERE cs.activity_id = ANY($1) AND cs.status = 'scheduled'
		  AND cs.starts_at > now() AND cs.starts_at::date <= $2::date
		  AND cs.capacity > (
		        SELECT COUNT(*) FROM order_items oi
		        JOIN orders o ON o.id = oi.order_id
		        WHERE oi.class_session_id = cs.id AND o.status IN ('paid', 'pending')
		      )
	`, activityIDs, deadline)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dayActivities := map[string]map[string]bool{}
	for rows.Next() {
		var day, activityID string
		if err := rows.Scan(&day, &activityID); err != nil {
			return nil, err
		}
		if dayActivities[day] == nil {
			dayActivities[day] = map[string]bool{}
		}
		dayActivities[day][activityID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var dates []string
	for day, set := range dayActivities {
		if len(set) == len(activityIDs) {
			dates = append(dates, day)
		}
	}
	sort.Strings(dates)
	opts.Dates = dates
	opts.Eligible = len(dates) > 0
	if !opts.Eligible {
		opts.Reason = "Não há turma disponível dentro do prazo — fale com a equipe."
	}
	return opts, nil
}

// RescheduleNoShowSelfService is the student-triggered flow (área do aluno): unlike
// RescheduleNoShow (staff, auto-picks the next session), here the student picks the makeup
// date themselves from ListNoShowRescheduleOptions's result — this re-validates that choice
// server-side (open seats can vanish between listing and confirming) before committing it.
func (r *OrderRepository) RescheduleNoShowSelfService(ctx context.Context, entitlementID, studentID, newDate string) (*NoShowRescheduleResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	orderID, missedDate, err := noShowEligibility(ctx, tx, entitlementID, studentID)
	if err != nil {
		return nil, err
	}
	deadline, err := addDaysISO(missedDate, noShowRescheduleWindowDays)
	if err != nil {
		return nil, err
	}
	if newDate <= missedDate || newDate > deadline {
		return nil, ErrNoShowWindowExpired
	}

	activityIDs, err := entitlementClassActivityIDs(ctx, tx, entitlementID)
	if err != nil {
		return nil, err
	}
	if len(activityIDs) == 0 {
		return nil, ErrNoNextSession
	}

	for _, activityID := range activityIDs {
		var sessionID string
		var capacity, booked int
		err := tx.QueryRow(ctx, `
			SELECT cs.id, cs.capacity, (
				SELECT COUNT(*) FROM order_items oi
				JOIN orders o ON o.id = oi.order_id
				WHERE oi.class_session_id = cs.id AND o.status IN ('paid', 'pending')
			)
			FROM class_sessions cs
			WHERE cs.activity_id = $1 AND cs.status = 'scheduled'
			  AND to_char(cs.starts_at AT TIME ZONE 'America/Fortaleza', 'YYYY-MM-DD') = $2
			ORDER BY cs.starts_at ASC
			LIMIT 1
			FOR UPDATE
		`, activityID, newDate).Scan(&sessionID, &capacity, &booked)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoSessionOnDate
		}
		if err != nil {
			return nil, err
		}
		if booked >= capacity {
			return nil, ErrSessionFull
		}

		if _, err := tx.Exec(ctx, `
			UPDATE order_items SET class_session_id = $1
			WHERE benefit_type = 'class' AND activity_id = $2
			  AND id IN (SELECT order_item_id FROM entitlement_items WHERE entitlement_id = $3)
		`, sessionID, activityID, entitlementID); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE entitlements SET valid_from = $1::date, valid_until = $1::date, no_show_reschedule_used_at = now()
		WHERE id = $2
	`, newDate, entitlementID); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO order_reschedules (order_id, changed_by_student_id, reason, previous_date, new_date)
		VALUES ($1, $2, 'Remarcação por falta (autoatendimento do aluno)', $3::date, $4::date)
	`, orderID, studentID, missedDate, newDate); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &NoShowRescheduleResult{OrderID: orderID, EntitlementID: entitlementID, PreviousDate: missedDate, NewDate: newDate}, nil
}

func generateOrderNumber() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("P5-%d-%X", time.Now().Year(), b), nil
}
