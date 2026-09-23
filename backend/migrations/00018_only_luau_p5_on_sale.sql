-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Deixa à venda apenas o Downwind do Luau P5. A 00017 desativou os ingressos
-- do DownWind Day pelos IDs do seed (...301/...302), mas em produção eles
-- existem com outros IDs (cadastrados pelo Admin) e continuaram na landing.
-- Desativa tudo que não é o produto/atividade do Luau, sem depender de ID.
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
UPDATE products
   SET active = false, featured = false, updated_at = now()
 WHERE id <> '00000000-0000-0000-0000-000000000303'
   AND (active = true OR featured = true);

UPDATE activities
   SET active = false, updated_at = now()
 WHERE id <> '00000000-0000-0000-0000-000000000202'
   AND active = true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Sem reversão automática: os ingressos antigos são reativados pelo Admin.
-- +goose StatementEnd
