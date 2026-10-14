-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- DownWind do Luau P5 — sábado, 24/10/2026.
-- Rota: Sababa → P5 Kite House. Saída às 15h, chegada até 18h30.
-- Dois ingressos: R$ 100 (sem lycra) · R$ 160 (com lycra exclusiva P5).
-- Corrige os títulos criados em 00021 (que haviam usado "Presídio" por engano).
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
UPDATE activities
   SET title       = 'DownWind do Luau P5',
       slug        = 'downwind-luau-p5-24out',
       description = 'Downwind Sababa → P5 Kite House com apoio aquático e terrestre da equipe P5 do início ao fim.',
       updated_at  = now()
 WHERE id = '00000000-0000-0000-0000-000000000203';

UPDATE products
   SET title      = 'DownWind do Luau P5 · Sem lycra',
       slug       = 'downwind-luau-p5-24out-sem-lycra',
       description= 'Sababa → P5 Kite House com transporte, apoio aquático e terrestre, estrutura na saída e receptivo na chegada.',
       updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000304';

UPDATE products
   SET title      = 'DownWind do Luau P5 · Com lycra exclusiva P5',
       slug       = 'downwind-luau-p5-24out-com-lycra',
       description= 'Tudo do ingresso sem lycra + a lycra exclusiva P5.',
       updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000305';

UPDATE class_sessions
   SET starts_at = '2026-10-24 15:00:00-03',
       ends_at   = '2026-10-24 18:30:00-03'
 WHERE id = '00000000-0000-0000-0000-000000000403';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE activities
   SET title       = 'P5 DownWind Day',
       slug        = 'p5-downwind-day-presidio',
       description = 'Percurso Praia do Presídio → P5 Kite House, com apoio aquático e terrestre da equipe P5 do início ao fim.',
       updated_at  = now()
 WHERE id = '00000000-0000-0000-0000-000000000203';

UPDATE products
   SET title      = 'P5 DownWind Day · Sem lycra',
       slug       = 'p5-downwind-day-presidio-sem-lycra',
       description= 'Praia do Presídio → P5 Kite House com transporte, apoio aquático e terrestre, estrutura na saída e receptivo na chegada.',
       updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000304';

UPDATE products
   SET title      = 'P5 DownWind Day · Com lycra exclusiva P5',
       slug       = 'p5-downwind-day-presidio-com-lycra',
       description= 'Tudo do ingresso sem lycra + a lycra exclusiva P5.',
       updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000305';

UPDATE class_sessions
   SET starts_at = '2026-10-24 07:30:00-03',
       ends_at   = '2026-10-24 13:00:00-03'
 WHERE id = '00000000-0000-0000-0000-000000000403';
-- +goose StatementEnd
