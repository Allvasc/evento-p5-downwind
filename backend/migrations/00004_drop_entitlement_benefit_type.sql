-- +goose Up
-- +goose StatementBegin
-- benefit_type era 1:1 com order_item; agora um entitlement pode cobrir vários
-- order_items do mesmo setor (via entitlement_items), então deixou de fazer sentido
-- como coluna única aqui — o tipo de cada item já vive em order_items.benefit_type.
ALTER TABLE entitlements DROP COLUMN IF EXISTS benefit_type;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE entitlements ADD COLUMN benefit_type text;
-- +goose StatementEnd
