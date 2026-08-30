-- +goose Up
-- +goose StatementBegin
-- Remarcação por falta agora pode ser feita pelo próprio cliente (autoatendimento na área
-- logada), não só por staff — order_reschedules precisa registrar um autor OU outro, nunca
-- os dois nem nenhum.
ALTER TABLE order_reschedules ALTER COLUMN changed_by DROP NOT NULL;
ALTER TABLE order_reschedules ADD COLUMN changed_by_student_id uuid REFERENCES students(id);
ALTER TABLE order_reschedules ADD CONSTRAINT order_reschedules_changed_by_xor CHECK (
  (changed_by IS NOT NULL AND changed_by_student_id IS NULL) OR
  (changed_by IS NULL AND changed_by_student_id IS NOT NULL)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM order_reschedules WHERE changed_by IS NULL;
ALTER TABLE order_reschedules DROP CONSTRAINT order_reschedules_changed_by_xor;
ALTER TABLE order_reschedules DROP COLUMN changed_by_student_id;
ALTER TABLE order_reschedules ALTER COLUMN changed_by SET NOT NULL;
-- +goose StatementEnd
