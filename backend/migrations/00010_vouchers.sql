-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Vouchers — cortesia dada a uma empresa parceira (ex: funcionários da Coca-
-- Cola): quem tem o código resgata, dentro da área logada, uma aula avulsa ou
-- o combo de aulas (sem café da manhã) sem passar pelo pagamento. Vira um
-- pedido igual a uma compra normal, só que com payment_method = 'voucher'.
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
CREATE TABLE vouchers (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  code          text NOT NULL UNIQUE,
  name          text NOT NULL,
  company_name  text NOT NULL,
  status        text NOT NULL DEFAULT 'available' CHECK (status IN ('available','used','cancelled')),
  created_by    uuid NOT NULL REFERENCES team_members(id),
  redeemed_by   uuid REFERENCES students(id),
  order_id      uuid REFERENCES orders(id),
  redeemed_at   timestamptz,
  created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_vouchers_status ON vouchers(status);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE orders DROP CONSTRAINT orders_payment_method_check;
ALTER TABLE orders ADD CONSTRAINT orders_payment_method_check
  CHECK (payment_method IN ('pix','credit_card','boleto','voucher'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders DROP CONSTRAINT orders_payment_method_check;
ALTER TABLE orders ADD CONSTRAINT orders_payment_method_check
  CHECK (payment_method IN ('pix','credit_card','boleto'));
DROP TABLE IF EXISTS vouchers;
-- +goose StatementEnd
