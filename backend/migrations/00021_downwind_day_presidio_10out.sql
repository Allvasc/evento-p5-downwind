-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- P5 DownWind Day — sábado, 10/10/2026. Percurso Praia do Presídio → P5 Kite
-- House. Saída às 7h30 da P5 (transporte até o ponto de partida), chegada até
-- as 13h. Apenas 16 vagas. Dois ingressos para a mesma atividade/turma:
--   • R$ 100 — sem lycra
--   • R$ 160 — com lycra exclusiva P5 (destaque)
--
-- Tudo que estava à venda (Downwind do Luau P5 e eventuais produtos criados
-- pelo Admin com outros IDs) sai de venda: desativa em vez de apagar, pois há
-- pedidos reais referenciando produto/atividade/turma.
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
UPDATE products
   SET active = false, featured = false, updated_at = now()
 WHERE id NOT IN ('00000000-0000-0000-0000-000000000304', '00000000-0000-0000-0000-000000000305')
   AND (active = true OR featured = true);

UPDATE activities
   SET active = false, updated_at = now()
 WHERE id <> '00000000-0000-0000-0000-000000000203'
   AND active = true;

-- vendor_id: '...f5' = P5 Kite House
INSERT INTO activities (id, title, slug, instructor, duration_minutes, description, display_order, vendor_id) VALUES
  ('00000000-0000-0000-0000-000000000203', 'P5 DownWind Day', 'p5-downwind-day-presidio', 'Equipe P5', 330,
   'Percurso Praia do Presídio → P5 Kite House, com apoio aquático e terrestre da equipe P5 do início ao fim.', 1,
   '00000000-0000-0000-0000-0000000000f5')
ON CONFLICT (id) DO NOTHING;

INSERT INTO products (id, title, slug, description, type, includes_breakfast, price_cents, featured, active, display_order, choose_one_activity) VALUES
  ('00000000-0000-0000-0000-000000000304', 'P5 DownWind Day · Sem lycra', 'p5-downwind-day-presidio-sem-lycra',
   'Praia do Presídio → P5 Kite House com transporte, apoio aquático e terrestre, estrutura na saída e receptivo na chegada.',
   'class', false, 10000, false, true, 1, false),
  ('00000000-0000-0000-0000-000000000305', 'P5 DownWind Day · Com lycra exclusiva P5', 'p5-downwind-day-presidio-com-lycra',
   'Tudo do ingresso sem lycra + a lycra exclusiva P5.',
   'class', false, 16000, true, true, 2, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO product_activities (product_id, activity_id) VALUES
  ('00000000-0000-0000-0000-000000000304', '00000000-0000-0000-0000-000000000203'),
  ('00000000-0000-0000-0000-000000000305', '00000000-0000-0000-0000-000000000203')
ON CONFLICT DO NOTHING;

INSERT INTO class_sessions (id, activity_id, starts_at, ends_at, capacity, status)
VALUES (
  '00000000-0000-0000-0000-000000000403',
  '00000000-0000-0000-0000-000000000203',
  '2026-10-10 07:30:00-03',
  '2026-10-10 13:00:00-03',
  16,
  'scheduled'
)
ON CONFLICT (id) DO UPDATE
   SET starts_at = EXCLUDED.starts_at,
       ends_at = EXCLUDED.ends_at,
       capacity = EXCLUDED.capacity,
       status = 'scheduled';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM class_sessions     WHERE id = '00000000-0000-0000-0000-000000000403';
DELETE FROM product_activities WHERE product_id IN ('00000000-0000-0000-0000-000000000304', '00000000-0000-0000-0000-000000000305');
DELETE FROM products           WHERE id IN ('00000000-0000-0000-0000-000000000304', '00000000-0000-0000-0000-000000000305');
DELETE FROM activities         WHERE id = '00000000-0000-0000-0000-000000000203';

UPDATE activities SET active = true WHERE id = '00000000-0000-0000-0000-000000000202';
UPDATE products   SET active = true, featured = true WHERE id = '00000000-0000-0000-0000-000000000303';
-- +goose StatementEnd
