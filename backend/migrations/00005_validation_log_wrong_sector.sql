-- +goose Up
-- +goose StatementBegin
ALTER TABLE validation_log DROP CONSTRAINT validation_log_result_check;
ALTER TABLE validation_log ADD CONSTRAINT validation_log_result_check
  CHECK (result IN ('success','already_used','expired','not_found','invalid_signature','override','wrong_sector'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE validation_log DROP CONSTRAINT validation_log_result_check;
ALTER TABLE validation_log ADD CONSTRAINT validation_log_result_check
  CHECK (result IN ('success','already_used','expired','not_found','invalid_signature','override'));
-- +goose StatementEnd
