-- +goose Up
-- +goose StatementBegin
-- Controla o direito de remarcação por falta: 1 remarcação grátis por ingresso (QR), perdida
-- se a próxima data do evento também passar sem o cliente usá-la. NULL = direito ainda não
-- usado; preenchido quando a remarcação automática por falta é aplicada.
ALTER TABLE entitlements ADD COLUMN no_show_reschedule_used_at timestamptz;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE entitlements DROP COLUMN no_show_reschedule_used_at;
-- +goose StatementEnd
