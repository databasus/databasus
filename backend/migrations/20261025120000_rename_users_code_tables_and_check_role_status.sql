-- +goose Up
-- +goose StatementBegin
ALTER TABLE password_reset_codes
    RENAME TO users_password_reset_codes;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE users_password_reset_codes
    RENAME CONSTRAINT password_reset_codes_pkey TO users_password_reset_codes_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE users_password_reset_codes
    RENAME CONSTRAINT fk_password_reset_codes_user_id TO fk_users_password_reset_codes_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_password_reset_codes_user_id
    RENAME TO idx_users_password_reset_codes_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_password_reset_codes_expires_at
    RENAME TO idx_users_password_reset_codes_expires_at;
-- +goose StatementEnd

-- A row is a pending sign-in: its ID is the pendingSignInId of the API.
-- +goose StatementBegin
ALTER TABLE two_factor_codes
    RENAME TO users_pending_sign_ins;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE users_pending_sign_ins
    RENAME CONSTRAINT two_factor_codes_pkey TO users_pending_sign_ins_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE users_pending_sign_ins
    RENAME CONSTRAINT fk_two_factor_codes_user_id TO fk_users_pending_sign_ins_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE users_pending_sign_ins
    RENAME CONSTRAINT ck_two_factor_codes_failed_attempt_count_not_negative
    TO ck_users_pending_sign_ins_failed_attempt_count_not_negative;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_two_factor_codes_user_id
    RENAME TO idx_users_pending_sign_ins_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_two_factor_codes_created_at
    RENAME TO idx_users_pending_sign_ins_created_at;
-- +goose StatementEnd

-- A damaged row loses its rights instead of stopping the upgrade.
-- +goose StatementBegin
UPDATE users
SET role = 'MEMBER'
WHERE role NOT IN ('ADMIN', 'MEMBER');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE users
SET status = 'INACTIVE'
WHERE status NOT IN ('INVITED', 'ACTIVE', 'INACTIVE');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER INDEX idx_users_pending_sign_ins_created_at
    RENAME TO idx_two_factor_codes_created_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_users_pending_sign_ins_user_id
    RENAME TO idx_two_factor_codes_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE users_pending_sign_ins
    RENAME CONSTRAINT ck_users_pending_sign_ins_failed_attempt_count_not_negative
    TO ck_two_factor_codes_failed_attempt_count_not_negative;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE users_pending_sign_ins
    RENAME CONSTRAINT fk_users_pending_sign_ins_user_id TO fk_two_factor_codes_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE users_pending_sign_ins
    RENAME CONSTRAINT users_pending_sign_ins_pkey TO two_factor_codes_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE users_pending_sign_ins
    RENAME TO two_factor_codes;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_users_password_reset_codes_expires_at
    RENAME TO idx_password_reset_codes_expires_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER INDEX idx_users_password_reset_codes_user_id
    RENAME TO idx_password_reset_codes_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE users_password_reset_codes
    RENAME CONSTRAINT fk_users_password_reset_codes_user_id TO fk_password_reset_codes_user_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE users_password_reset_codes
    RENAME CONSTRAINT users_password_reset_codes_pkey TO password_reset_codes_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE users_password_reset_codes
    RENAME TO password_reset_codes;
-- +goose StatementEnd
