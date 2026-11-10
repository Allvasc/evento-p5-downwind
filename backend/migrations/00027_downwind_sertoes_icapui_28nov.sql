-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- P5 DownWind Day — Sertões Experience, sábado 28/11/2026.
-- Abertura da P5 às 5h (café da manhã), saída de ônibus às 6h30 rumo a Ponta
-- Grossa (Icapuí); downwind de ~40 km até Fortim; retorno noturno à P5.
-- Ingresso único: R$ 1.000,00 · 40 vagas. Encerra a edição de Halloween.
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
UPDATE products
   SET active = false, featured = false, updated_at = now()
 WHERE active = true OR featured = true;

UPDATE activities
   SET active = false, updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000204';

INSERT INTO activities (id, title, slug, instructor, duration_minutes, description, display_order, vendor_id) VALUES
  ('00000000-0000-0000-0000-000000000205', 'DownWind Icapuí → Fortim', 'downwind-sertoes-icapui-28nov', 'Equipe P5', 960,
   'Sertões Experience: ônibus da P5 até Ponta Grossa (Icapuí) e downwind de aproximadamente 40 km até Fortim, com equipe de acompanhamento.', 1,
   '00000000-0000-0000-0000-0000000000f5')
ON CONFLICT (id) DO NOTHING;

INSERT INTO products (id, title, slug, description, type, includes_breakfast, price_cents, featured, active, display_order, choose_one_activity) VALUES
  ('00000000-0000-0000-0000-000000000307', 'P5 DownWind Day — Sertões Experience', 'downwind-sertoes-icapui-28nov',
   'Icapuí → Fortim (~40 km). Inclui inscrição Sertões Experience, café da manhã na P5, kit do evento, ônibus executivo ida e volta, equipe de acompanhamento com hidratação e snacks, equipamentos de kite e entretenimento.',
   'class', false, 100000, true, true, 1, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO product_activities (product_id, activity_id) VALUES
  ('00000000-0000-0000-0000-000000000307', '00000000-0000-0000-0000-000000000205')
ON CONFLICT DO NOTHING;

INSERT INTO class_sessions (id, activity_id, starts_at, ends_at, capacity, status)
VALUES (
  '00000000-0000-0000-0000-000000000405',
  '00000000-0000-0000-0000-000000000205',
  '2026-11-28 05:00:00-03',
  '2026-11-28 21:00:00-03',
  40,
  'scheduled'
)
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM class_sessions     WHERE id = '00000000-0000-0000-0000-000000000405';
DELETE FROM product_activities WHERE product_id = '00000000-0000-0000-0000-000000000307';
DELETE FROM products           WHERE id = '00000000-0000-0000-0000-000000000307';
DELETE FROM activities         WHERE id = '00000000-0000-0000-0000-000000000205';

UPDATE activities SET active = true, updated_at = now() WHERE id = '00000000-0000-0000-0000-000000000204';
UPDATE products   SET active = true, featured = true, updated_at = now() WHERE id = '00000000-0000-0000-0000-000000000306';
-- +goose StatementEnd
