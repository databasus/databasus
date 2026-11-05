-- +goose Up
ALTER TABLE users
    ADD COLUMN password_version UUID NOT NULL DEFAULT gen_random_uuid();

ALTER TABLE users_pending_sign_ins
    ADD COLUMN password_version UUID NOT NULL DEFAULT gen_random_uuid();

ALTER TABLE users_pending_sign_ins
    ALTER COLUMN password_version DROP DEFAULT;

ALTER TABLE users_pending_sign_ins
    DROP COLUMN password_creation_time;

-- +goose Down
ALTER TABLE users_pending_sign_ins
    ADD COLUMN password_creation_time TIMESTAMPTZ NOT NULL DEFAULT '-infinity';

ALTER TABLE users_pending_sign_ins
    ALTER COLUMN password_creation_time DROP DEFAULT;

ALTER TABLE users_pending_sign_ins
    DROP COLUMN password_version;

ALTER TABLE users
    DROP COLUMN password_version;
