-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════
-- Contato de emergência do cliente (nome + telefone). Obrigatório no cadastro
-- novo; quem já tem conta preenche antes da próxima compra ou resgate de
-- voucher — por isso a coluna é nullable (contas antigas ainda não têm).
-- ═══════════════════════════════════════════════════════════════════════════
-- +goose StatementBegin
ALTER TABLE students
  ADD COLUMN emergency_contact_name  text,
  ADD COLUMN emergency_contact_phone varchar(20);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE students
  DROP COLUMN emergency_contact_name,
  DROP COLUMN emergency_contact_phone;
-- +goose StatementEnd
