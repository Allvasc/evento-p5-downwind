-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Downwind do Luau P5 tem apenas 10 vagas (a 00017 criou a turma com 50).
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
UPDATE class_sessions
   SET capacity = 10, updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000402';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE class_sessions
   SET capacity = 50, updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000402';
-- +goose StatementEnd
