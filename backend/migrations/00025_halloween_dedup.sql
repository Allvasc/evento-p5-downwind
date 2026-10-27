-- +goose Up
-- Garante que apenas o produto Halloween oficial (306) fique ativo.
-- Desativa qualquer outro produto ativo que não seja o 306.
-- +goose StatementBegin
UPDATE products
   SET active = false, featured = false, updated_at = now()
 WHERE id <> '00000000-0000-0000-0000-000000000306'
   AND (active = true OR featured = true);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1; -- irreversível por design: não sabemos quais estavam ativos antes
-- +goose StatementEnd
