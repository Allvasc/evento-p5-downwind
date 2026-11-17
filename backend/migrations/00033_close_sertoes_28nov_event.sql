-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Encerra a edição de 28/11 (Sertões Kitesurf Experience) no evento principal:
-- desativa produto, atividade e cancela a turma agendada, mantendo histórico
-- intacto no banco e demais eventos (como /test-session) ativos.
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
UPDATE products
   SET active = false, featured = false, updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000307';

UPDATE activities
   SET active = false, updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000205';

UPDATE class_sessions
   SET status = 'cancelled', updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000405'
   AND status = 'scheduled';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE activities
   SET active = true, updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000205';

UPDATE products
   SET active = true, featured = true, updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000307';

UPDATE class_sessions
   SET status = 'scheduled', updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000405'
   AND status = 'cancelled';
-- +goose StatementEnd
