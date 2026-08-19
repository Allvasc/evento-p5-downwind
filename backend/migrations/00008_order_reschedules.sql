-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Remarcação de pedido — quando um cliente não comparece e pede pra trocar de
-- data, a equipe move o pedido inteiro (todas as turmas + o café) pra um novo
-- dia e registra quem fez, quando e por quê.
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
CREATE TABLE order_reschedules (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id       uuid NOT NULL REFERENCES orders(id),
  changed_by     uuid NOT NULL REFERENCES team_members(id),
  reason         text NOT NULL,
  previous_date  date NOT NULL,
  new_date       date NOT NULL,
  created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_order_reschedules_order ON order_reschedules(order_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS order_reschedules;
-- +goose StatementEnd
