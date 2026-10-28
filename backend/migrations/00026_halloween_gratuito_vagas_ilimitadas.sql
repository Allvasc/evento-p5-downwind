-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Downwind Halloween P5 (31/10) vira evento gratuito com vagas ilimitadas.
-- Gratuidade = produto com preço R$ 0: o checkout pula a Asaas e emite o
-- ingresso na hora, com payment_method = 'free' (serve como inscrição e para
-- contar quantas pessoas se inscreveram). Para voltar a cobrar, basta editar o
-- preço do produto no Admin.
-- Vagas ilimitadas = capacidade 9999 (o Admin e o site exibem "ilimitadas").
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
ALTER TABLE orders DROP CONSTRAINT orders_payment_method_check;
ALTER TABLE orders ADD CONSTRAINT orders_payment_method_check
  CHECK (payment_method IN ('pix','credit_card','boleto','voucher','free'));

UPDATE products
   SET price_cents = 0, updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000306';

UPDATE class_sessions
   SET capacity = 9999, updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000404';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE class_sessions
   SET capacity = 16, updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000404';

UPDATE products
   SET price_cents = 10000, updated_at = now()
 WHERE id = '00000000-0000-0000-0000-000000000306';

UPDATE orders SET payment_method = NULL WHERE payment_method = 'free';
ALTER TABLE orders DROP CONSTRAINT orders_payment_method_check;
ALTER TABLE orders ADD CONSTRAINT orders_payment_method_check
  CHECK (payment_method IN ('pix','credit_card','boleto','voucher'));
-- +goose StatementEnd
