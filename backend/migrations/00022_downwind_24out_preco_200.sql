-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- P5 DownWind Day — atualização:
--   • Data do evento: 10/10/2026 → 24/10/2026
--   • Preços mantidos: R$ 100 (sem lycra) · R$ 160 (com lycra)
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
UPDATE class_sessions
   SET starts_at = '2026-10-24 07:30:00-03',
       ends_at   = '2026-10-24 13:00:00-03'
 WHERE id = '00000000-0000-0000-0000-000000000403';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE class_sessions
   SET starts_at = '2026-10-10 07:30:00-03',
       ends_at   = '2026-10-10 13:00:00-03'
 WHERE id = '00000000-0000-0000-0000-000000000403';
-- +goose StatementEnd
