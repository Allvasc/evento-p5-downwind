-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Atualiza o percurso para Riviera Wind → P5 (títulos e descrições)
-- e cadastra a turma do dia 19/09/2026 para todas as atividades ativas.
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
UPDATE activities
   SET title = REPLACE(REPLACE(title, 'Prainha', 'Riviera Wind'), 'Aquiraz Riviera', 'Riviera Wind'),
       description = REPLACE(REPLACE(description, 'Prainha', 'Riviera Wind'), 'Aquiraz Riviera', 'Riviera Wind')
 WHERE title LIKE '%Prainha%' OR description LIKE '%Prainha%'
    OR title LIKE '%Aquiraz Riviera%' OR description LIKE '%Aquiraz Riviera%';

UPDATE products
   SET title = REPLACE(REPLACE(title, 'Prainha', 'Riviera Wind'), 'Aquiraz Riviera', 'Riviera Wind'),
       description = REPLACE(REPLACE(description, 'Prainha', 'Riviera Wind'), 'Aquiraz Riviera', 'Riviera Wind')
 WHERE title LIKE '%Prainha%' OR description LIKE '%Prainha%'
    OR title LIKE '%Aquiraz Riviera%' OR description LIKE '%Aquiraz Riviera%';

UPDATE activities
   SET description = 'Percurso guiado do Riviera Wind até a P5 Kite House, com apoio aquático e terrestre o tempo todo.'
 WHERE id = '00000000-0000-0000-0000-000000000201';

UPDATE products
   SET description = 'Percurso guiado Riviera Wind → P5 com transporte, apoio aquático/terrestre e estrutura inclusa.'
 WHERE id = '00000000-0000-0000-0000-000000000301';

-- Garante sessão do dia 19/09/2026 para a atividade principal do P5 DownWind Day
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

-- Garante sessão do dia 19/09/2026 para qualquer outra atividade ativa sem turma nessa data
INSERT INTO class_sessions (activity_id, starts_at, ends_at, capacity, status)
SELECT id, '2026-09-19 06:00:00-03', '2026-09-19 11:30:00-03', 50, 'scheduled'
FROM activities
WHERE active = true
  AND id NOT IN (
    SELECT activity_id FROM class_sessions WHERE starts_at::date = '2026-09-19'::date
  );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM class_sessions WHERE id = '00000000-0000-0000-0000-000000000401' OR starts_at::date = '2026-09-19'::date;

UPDATE products
   SET title = REPLACE(title, 'Riviera Wind', 'Prainha'),
       description = REPLACE(description, 'Riviera Wind', 'Prainha')
 WHERE title LIKE '%Riviera Wind%' OR description LIKE '%Riviera Wind%';

UPDATE activities
   SET title = REPLACE(title, 'Riviera Wind', 'Prainha'),
       description = REPLACE(description, 'Riviera Wind', 'Prainha')
 WHERE title LIKE '%Riviera Wind%' OR description LIKE '%Riviera Wind%';
-- +goose StatementEnd
