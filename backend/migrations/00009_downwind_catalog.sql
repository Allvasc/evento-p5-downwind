-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Catálogo real do P5 DownWind Day — um ingresso único, tudo incluso, no lugar
-- do catálogo de demonstração (Yoga/HYROX, seedado por 00002). Desativa em vez
-- de apagar: um pedido de teste pode ter referenciado os produtos antigos, e
-- order_items.product_id não tem ON DELETE CASCADE.
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
UPDATE products SET active = false WHERE id IN (
  '00000000-0000-0000-0000-000000000101',
  '00000000-0000-0000-0000-000000000102',
  '00000000-0000-0000-0000-000000000103'
);
UPDATE activities SET active = false WHERE id IN (
  '00000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000002'
);

-- vendor_id: '...f5' = P5 Kite House, seedado por 00003_vendor_scoped_entitlements.sql
-- (coluna NOT NULL desde aquela migration — não existia ainda quando 00002 rodou).
INSERT INTO activities (id, title, slug, instructor, duration_minutes, description, display_order, vendor_id) VALUES
  ('00000000-0000-0000-0000-000000000201', 'P5 DownWind Day', 'p5-downwind-day', 'Equipe P5', 330,
   'Percurso guiado da P5 Kite House até a Praia do Presídio, com apoio aquático e terrestre o tempo todo.', 1,
   '00000000-0000-0000-0000-0000000000f5');

INSERT INTO products (id, title, slug, description, type, includes_breakfast, price_cents, featured, active, display_order, choose_one_activity) VALUES
  ('00000000-0000-0000-0000-000000000301', 'P5 DownWind Day', 'p5-downwind-day',
   'Percurso Praia do Presídio, transporte, apoio aquático e terrestre, estrutura completa no ponto de saída e monitoramento pelo Wind Maps — tudo incluso.',
   'class', false, 10000, true, true, 1, false);

INSERT INTO product_activities (product_id, activity_id) VALUES
  ('00000000-0000-0000-0000-000000000301', '00000000-0000-0000-0000-000000000201');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM product_activities WHERE product_id = '00000000-0000-0000-0000-000000000301';
DELETE FROM products WHERE id = '00000000-0000-0000-0000-000000000301';
DELETE FROM activities WHERE id = '00000000-0000-0000-0000-000000000201';

UPDATE products SET active = true WHERE id IN (
  '00000000-0000-0000-0000-000000000101',
  '00000000-0000-0000-0000-000000000102',
  '00000000-0000-0000-0000-000000000103'
);
UPDATE activities SET active = true WHERE id IN (
  '00000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000002'
);
-- +goose StatementEnd
