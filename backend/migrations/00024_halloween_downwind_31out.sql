-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Downwind Halloween P5 — sexta, 31/10/2026.
-- Concentração às 9h30 na P5 Kite House. Saída às 10h rumo ao Sababa.
-- Percurso: Sababa → P5 Kite House. Evento à fantasia.
-- Ingresso único: R$ 100 (sem lycra, sem café da manhã).
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
UPDATE products
   SET active = false, featured = false, updated_at = now()
 WHERE id IN ('00000000-0000-0000-0000-000000000304', '00000000-0000-0000-0000-000000000305');

UPDATE activities
   SET active = false, updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000203';

INSERT INTO activities (id, title, slug, instructor, duration_minutes, description, display_order, vendor_id) VALUES
  ('00000000-0000-0000-0000-000000000204', 'Downwind Halloween P5', 'downwind-halloween-p5-31out', 'Equipe P5', 210,
   'Downwind Sababa → P5 Kite House. Evento à fantasia com suporte aquático e terrestre da equipe P5.', 1,
   '00000000-0000-0000-0000-0000000000f5')
ON CONFLICT (id) DO NOTHING;

INSERT INTO products (id, title, slug, description, type, includes_breakfast, price_cents, featured, active, display_order, choose_one_activity) VALUES
  ('00000000-0000-0000-0000-000000000306', 'Downwind Halloween P5', 'downwind-halloween-p5-31out',
   'Percurso Sababa → P5 Kite House com suporte aquático e terrestre. Evento à fantasia — venha caracterizado!',
   'class', false, 10000, true, true, 1, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO product_activities (product_id, activity_id) VALUES
  ('00000000-0000-0000-0000-000000000306', '00000000-0000-0000-0000-000000000204')
ON CONFLICT DO NOTHING;

INSERT INTO class_sessions (id, activity_id, starts_at, ends_at, capacity, status)
VALUES (
  '00000000-0000-0000-0000-000000000404',
  '00000000-0000-0000-0000-000000000204',
  '2026-10-31 09:30:00-03',
  '2026-10-31 13:00:00-03',
  16,
  'scheduled'
)
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM class_sessions     WHERE id = '00000000-0000-0000-0000-000000000404';
DELETE FROM product_activities WHERE product_id = '00000000-0000-0000-0000-000000000306';
DELETE FROM products           WHERE id = '00000000-0000-0000-0000-000000000306';
DELETE FROM activities         WHERE id = '00000000-0000-0000-0000-000000000204';

UPDATE activities SET active = true, updated_at = now() WHERE id = '00000000-0000-0000-0000-000000000203';
UPDATE products   SET active = true, featured = false, updated_at = now() WHERE id = '00000000-0000-0000-0000-000000000304';
UPDATE products   SET active = true, featured = true,  updated_at = now() WHERE id = '00000000-0000-0000-0000-000000000305';
-- +goose StatementEnd
