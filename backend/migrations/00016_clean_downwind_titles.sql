-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Remove sufixos após traço dos títulos de produtos e atividades P5 DownWind
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
UPDATE activities
   SET title = 'P5 DownWind Day'
 WHERE title LIKE 'P5 DownWind%';

UPDATE products
   SET title = 'P5 DownWind Day'
 WHERE (title LIKE 'P5 DownWind%')
   AND title NOT LIKE '%Café%' AND title NOT LIKE '%café%';

UPDATE products
   SET title = 'P5 DownWind + Café da Manhã'
 WHERE (title LIKE 'P5 DownWind%')
   AND (title LIKE '%Café%' OR title LIKE '%café%');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Sem ação de reversão necessária
-- +goose StatementEnd
