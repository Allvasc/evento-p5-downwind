-- +goose Up
-- +goose StatementBegin
-- Nova role "reports": enxerga todos os relatórios (por turma/produto/atividade) mas não
-- tem acesso a nenhuma config de admin (catálogo, equipe, pedidos, clientes, dashboard).
ALTER TABLE team_members DROP CONSTRAINT team_members_role_check;
ALTER TABLE team_members ADD CONSTRAINT team_members_role_check CHECK (role IN ('admin','staff','reports'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE team_members DROP CONSTRAINT team_members_role_check;
ALTER TABLE team_members ADD CONSTRAINT team_members_role_check CHECK (role IN ('admin','staff'));
-- +goose StatementEnd
