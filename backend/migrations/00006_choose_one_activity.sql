-- +goose Up
-- +goose StatementBegin
-- Some products list two or more activities as alternatives (e.g. "Aula Individual:
-- Yoga ou HYROX") rather than requiring all of them (e.g. a combo needing both). This
-- flag switches CreateOrder's behavior for the product: pick exactly one linked
-- activity's session instead of requiring a session for every linked activity.
ALTER TABLE products ADD COLUMN choose_one_activity boolean NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE products DROP COLUMN choose_one_activity;
-- +goose StatementEnd
