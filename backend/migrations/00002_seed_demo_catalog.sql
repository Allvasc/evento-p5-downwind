-- +goose Up
-- +goose StatementBegin
INSERT INTO activities (id, title, slug, instructor, duration_minutes, description, display_order) VALUES
  ('00000000-0000-0000-0000-000000000001', 'Yoga',  'yoga',  'Equipe AYO', 60, 'Práticas guiadas para todos os níveis, em uma atmosfera acolhedora.', 1),
  ('00000000-0000-0000-0000-000000000002', 'HYROX', 'hyrox', 'Equipe AYO', 60, 'Yoga e HYROX para acordar o corpo com intenção, presença e energia.', 2);

INSERT INTO products (id, title, slug, description, type, includes_breakfast, price_cents, featured, display_order) VALUES
  ('00000000-0000-0000-0000-000000000101', 'Aula individual', 'aula-individual', 'Yoga ou HYROX — uma prática, um novo ritmo.', 'class', false, 2500, false, 1),
  ('00000000-0000-0000-0000-000000000102', 'Aulas', 'aulas-yoga-hyrox', 'Yoga + HYROX — movimento em dobro para viver a manhã.', 'combo', false, 4000, false, 2),
  ('00000000-0000-0000-0000-000000000103', 'Aulas + Café', 'aulas-cafe-manha', 'Yoga + HYROX + Café da Manhã — a experiência completa, da energia à mesa.', 'combo', true, 9000, true, 3);

INSERT INTO product_activities (product_id, activity_id) VALUES
  ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000001'),
  ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000002'),
  ('00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000001'),
  ('00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000002'),
  ('00000000-0000-0000-0000-000000000103', '00000000-0000-0000-0000-000000000001'),
  ('00000000-0000-0000-0000-000000000103', '00000000-0000-0000-0000-000000000002');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM product_activities WHERE product_id IN (
  '00000000-0000-0000-0000-000000000101',
  '00000000-0000-0000-0000-000000000102',
  '00000000-0000-0000-0000-000000000103'
);
DELETE FROM products WHERE id IN (
  '00000000-0000-0000-0000-000000000101',
  '00000000-0000-0000-0000-000000000102',
  '00000000-0000-0000-0000-000000000103'
);
DELETE FROM activities WHERE id IN (
  '00000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000002'
);
-- +goose StatementEnd
