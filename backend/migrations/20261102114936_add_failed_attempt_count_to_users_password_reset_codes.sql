-- +goose Up
-- +goose StatementBegin

ALTER TABLE users_password_reset_codes
    ADD COLUMN failed_attempt_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE users_password_reset_codes
    ADD CONSTRAINT ck_users_password_reset_codes_failed_attempt_count_not_negative
    CHECK (failed_attempt_count >= 0);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE users_password_reset_codes
    DROP CONSTRAINT IF EXISTS ck_users_password_reset_codes_failed_attempt_count_not_negative;

ALTER TABLE users_password_reset_codes
    DROP COLUMN IF EXISTS failed_attempt_count;

-- +goose StatementEnd
