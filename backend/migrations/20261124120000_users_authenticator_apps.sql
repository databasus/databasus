-- +goose Up
ALTER TABLE users_settings
    ADD COLUMN two_factor_requirement TEXT NOT NULL DEFAULT 'OFF';

UPDATE users_settings
SET two_factor_requirement = 'EMAIL'
WHERE is_two_factor_auth_required;

ALTER TABLE users_settings
    DROP COLUMN is_two_factor_auth_required;

CREATE TABLE users_authenticator_apps (
    id               UUID        NOT NULL,
    user_id          UUID        NOT NULL,
    encrypted_secret TEXT        NOT NULL,
    last_used_step   BIGINT      NOT NULL DEFAULT 0,
    connected_at     TIMESTAMPTZ NOT NULL
);

ALTER TABLE users_authenticator_apps
    ADD CONSTRAINT pk_users_authenticator_apps
    PRIMARY KEY (id);

ALTER TABLE users_authenticator_apps
    ADD CONSTRAINT uq_users_authenticator_apps_user_id
    UNIQUE (user_id);

ALTER TABLE users_authenticator_apps
    ADD CONSTRAINT fk_users_authenticator_apps_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE CASCADE;

-- At most one unconfirmed setup per user: starting another replaces it.
CREATE TABLE users_authenticator_setups (
    id                   UUID        NOT NULL,
    user_id              UUID        NOT NULL,
    encrypted_secret     TEXT        NOT NULL,
    failed_attempt_count INTEGER     NOT NULL DEFAULT 0,
    expires_at           TIMESTAMPTZ NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL
);

ALTER TABLE users_authenticator_setups
    ADD CONSTRAINT pk_users_authenticator_setups
    PRIMARY KEY (id);

ALTER TABLE users_authenticator_setups
    ADD CONSTRAINT uq_users_authenticator_setups_user_id
    UNIQUE (user_id);

ALTER TABLE users_authenticator_setups
    ADD CONSTRAINT fk_users_authenticator_setups_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE CASCADE;

ALTER TABLE users_authenticator_setups
    ADD CONSTRAINT chk_users_authenticator_setups_failed_attempt_count
    CHECK (failed_attempt_count >= 0);

CREATE INDEX idx_users_authenticator_setups_expires_at
    ON users_authenticator_setups (expires_at);

CREATE TABLE users_recovery_codes (
    id                   UUID        NOT NULL,
    authenticator_app_id UUID        NOT NULL,
    hashed_code          TEXT        NOT NULL,
    used_at              TIMESTAMPTZ NULL
);

ALTER TABLE users_recovery_codes
    ADD CONSTRAINT pk_users_recovery_codes
    PRIMARY KEY (id);

ALTER TABLE users_recovery_codes
    ADD CONSTRAINT fk_users_recovery_codes_authenticator_app_id
    FOREIGN KEY (authenticator_app_id)
    REFERENCES users_authenticator_apps (id)
    ON DELETE CASCADE;

CREATE INDEX idx_users_recovery_codes_authenticator_app_id
    ON users_recovery_codes (authenticator_app_id);

-- One row is one wrong app or recovery code; the per-account limit counts the last hour of them.
CREATE TABLE users_second_factor_failures (
    id        UUID        NOT NULL,
    user_id   UUID        NOT NULL,
    failed_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE users_second_factor_failures
    ADD CONSTRAINT pk_users_second_factor_failures
    PRIMARY KEY (id);

ALTER TABLE users_second_factor_failures
    ADD CONSTRAINT fk_users_second_factor_failures_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE CASCADE;

CREATE INDEX idx_users_second_factor_failures_user_id
    ON users_second_factor_failures (user_id, failed_at);

CREATE INDEX idx_users_second_factor_failures_failed_at
    ON users_second_factor_failures (failed_at);

ALTER TABLE users_pending_sign_ins
    ADD COLUMN kind TEXT NOT NULL DEFAULT 'EMAIL_CODE';

ALTER TABLE users_pending_sign_ins
    ADD COLUMN authenticator_app_id UUID NULL;

ALTER TABLE users_pending_sign_ins
    ADD COLUMN authenticator_setup_id UUID NULL;

ALTER TABLE users_pending_sign_ins
    ALTER COLUMN hashed_code DROP NOT NULL;

-- A pending sign-in waits for exactly one thing: an emailed code, a connected app or a setup.
-- Binding it to the app or setup row is what makes a reset or replacement end it.
ALTER TABLE users_pending_sign_ins
    ADD CONSTRAINT chk_users_pending_sign_ins_one_second_factor
    CHECK (num_nonnulls(hashed_code, authenticator_app_id, authenticator_setup_id) = 1);

ALTER TABLE users_pending_sign_ins
    ADD CONSTRAINT fk_users_pending_sign_ins_authenticator_app_id
    FOREIGN KEY (authenticator_app_id)
    REFERENCES users_authenticator_apps (id)
    ON DELETE CASCADE;

ALTER TABLE users_pending_sign_ins
    ADD CONSTRAINT fk_users_pending_sign_ins_authenticator_setup_id
    FOREIGN KEY (authenticator_setup_id)
    REFERENCES users_authenticator_setups (id)
    ON DELETE CASCADE;

CREATE INDEX idx_users_pending_sign_ins_authenticator_app_id
    ON users_pending_sign_ins (authenticator_app_id);

CREATE INDEX idx_users_pending_sign_ins_authenticator_setup_id
    ON users_pending_sign_ins (authenticator_setup_id);

-- +goose Down
DELETE FROM users_pending_sign_ins
WHERE kind <> 'EMAIL_CODE';

DROP INDEX idx_users_pending_sign_ins_authenticator_setup_id;

DROP INDEX idx_users_pending_sign_ins_authenticator_app_id;

ALTER TABLE users_pending_sign_ins
    DROP CONSTRAINT fk_users_pending_sign_ins_authenticator_setup_id;

ALTER TABLE users_pending_sign_ins
    DROP CONSTRAINT fk_users_pending_sign_ins_authenticator_app_id;

ALTER TABLE users_pending_sign_ins
    DROP CONSTRAINT chk_users_pending_sign_ins_one_second_factor;

ALTER TABLE users_pending_sign_ins
    ALTER COLUMN hashed_code SET NOT NULL;

ALTER TABLE users_pending_sign_ins
    DROP COLUMN authenticator_setup_id;

ALTER TABLE users_pending_sign_ins
    DROP COLUMN authenticator_app_id;

ALTER TABLE users_pending_sign_ins
    DROP COLUMN kind;

DROP TABLE users_second_factor_failures;

DROP TABLE users_recovery_codes;

DROP TABLE users_authenticator_setups;

DROP TABLE users_authenticator_apps;

ALTER TABLE users_settings
    ADD COLUMN is_two_factor_auth_required BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE users_settings
SET is_two_factor_auth_required = two_factor_requirement <> 'OFF';

ALTER TABLE users_settings
    DROP COLUMN two_factor_requirement;
