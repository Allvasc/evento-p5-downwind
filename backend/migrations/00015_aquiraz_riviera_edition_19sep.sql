-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Atualiza o percurso para Aquiraz Riviera → P5 e cadastra a edição do dia 19/09/2026.
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
UPDATE activities
   SET description = 'Percurso guiado do Aquiraz Riviera até a P5 Kite House, com apoio aquático e terrestre o tempo todo.'
 WHERE id = '00000000-0000-0000-0000-000000000201';

UPDATE products
   SET description = 'Percurso guiado Aquiraz Riviera → P5 com transporte, apoio aquático/terrestre e estrutura inclusa.'
 WHERE id = '00000000-0000-0000-0000-000000000301';

INSERT INTO class_sessions (id, activity_id, starts_at, ends_at, capacity, status)
VALUES (
  '00000000-0000-0000-0000-000000000401',
  '00000000-0000-0000-0000-000000000201',
  '2026-09-19 06:00:00-03',
  '2026-09-19 11:30:00-03',
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
DELETE FROM class_sessions WHERE id = '00000000-0000-0000-0000-000000000401';

UPDATE products
   SET description = 'Percurso guiado Prainha → P5 com transporte, apoio aquático/terrestre e estrutura inclusa.'
 WHERE id = '00000000-0000-0000-0000-000000000301';

UPDATE activities
   SET description = 'Percurso guiado da Prainha até a P5 Kite House, com apoio aquático e terrestre o tempo todo.'
 WHERE id = '00000000-0000-0000-0000-000000000201';
-- +goose StatementEnd
