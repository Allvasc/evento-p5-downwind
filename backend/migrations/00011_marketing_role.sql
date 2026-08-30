-- +goose Up
-- +goose StatementBegin
-- Nova role "marketing": enxerga só a tela de vouchers, em modo somente leitura (não cria
-- nem cancela) — sem acesso a nenhuma outra config de admin.
ALTER TABLE team_members DROP CONSTRAINT team_members_role_check;
ALTER TABLE team_members ADD CONSTRAINT team_members_role_check CHECK (role IN ('admin','staff','reports','marketing'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE team_members DROP CONSTRAINT team_members_role_check;
ALTER TABLE team_members ADD CONSTRAINT team_members_role_check CHECK (role IN ('admin','staff','reports'));
-- +goose StatementEnd
