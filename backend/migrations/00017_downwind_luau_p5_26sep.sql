-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Downwind do Luau P5 — 26/09/2026. Concentração às 15h na P5, ida ao
-- Sababa (preparação + LEDs nos kites), saída do Sababa por volta das 17h30
-- (nascer da lua) e chegada na P5 Kite House entre 18h e 18h30, com Luau P5
-- na chegada. Ingresso único de R$ 200 com lycra exclusiva P5 Temporada 2026.
--
-- Os ingressos do DownWind Day (Riviera Wind → P5) saem de venda: desativa em
-- vez de apagar, pois há pedidos reais referenciando produto/atividade/turma.
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
UPDATE products   SET active = false, featured = false
 WHERE id IN ('00000000-0000-0000-0000-000000000301', '00000000-0000-0000-0000-000000000302');
UPDATE activities SET active = false
 WHERE id = '00000000-0000-0000-0000-000000000201';

-- vendor_id: '...f5' = P5 Kite House
INSERT INTO activities (id, title, slug, instructor, duration_minutes, description, display_order, vendor_id) VALUES
  ('00000000-0000-0000-0000-000000000202', 'Downwind do Luau P5', 'downwind-luau-p5', 'Equipe P5', 210,
   'Downwind Sababa → P5 Kite House na transição do pôr do sol para a noite, com kites iluminados por LEDs e Luau P5 na chegada.', 1,
   '00000000-0000-0000-0000-0000000000f5')
ON CONFLICT (id) DO NOTHING;

INSERT INTO products (id, title, slug, description, type, includes_breakfast, price_cents, featured, active, display_order, choose_one_activity) VALUES
  ('00000000-0000-0000-0000-000000000303', 'Downwind do Luau P5', 'downwind-luau-p5',
   'Downwind Sababa → P5 Kite House do pôr do sol à lua, com kites iluminados por LEDs, apoio na água, lycra exclusiva P5 Temporada 2026 e Luau P5 na chegada.',
   'class', false, 20000, true, true, 1, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO product_activities (product_id, activity_id) VALUES
  ('00000000-0000-0000-0000-000000000303', '00000000-0000-0000-0000-000000000202')
ON CONFLICT DO NOTHING;

INSERT INTO class_sessions (id, activity_id, starts_at, ends_at, capacity, status)
VALUES (
  '00000000-0000-0000-0000-000000000402',
  '00000000-0000-0000-0000-000000000202',
  '2026-09-26 15:00:00-03',
  '2026-09-26 18:30:00-03',
  50,
  'scheduled'
)
ON CONFLICT (id) DO UPDATE
   SET starts_at = EXCLUDED.starts_at,
       ends_at = EXCLUDED.ends_at,
       status = 'scheduled';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM class_sessions     WHERE id = '00000000-0000-0000-0000-000000000402';
DELETE FROM product_activities WHERE product_id = '00000000-0000-0000-0000-000000000303';
DELETE FROM products           WHERE id = '00000000-0000-0000-0000-000000000303';
DELETE FROM activities         WHERE id = '00000000-0000-0000-0000-000000000202';

UPDATE activities SET active = true WHERE id = '00000000-0000-0000-0000-000000000201';
UPDATE products   SET active = true WHERE id IN ('00000000-0000-0000-0000-000000000301', '00000000-0000-0000-0000-000000000302');
UPDATE products   SET featured = true WHERE id = '00000000-0000-0000-0000-000000000302';
-- +goose StatementEnd
