-- +goose Up
-- +goose StatementBegin
UPDATE events
   SET name = 'Test Session',
       updated_at = now()
 WHERE id = '00000000-0000-0000-0000-0000000000e2';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE events
   SET name = 'CORE Demo Day',
       updated_at = now()
 WHERE id = '00000000-0000-0000-0000-0000000000e2';
-- +goose StatementEnd
