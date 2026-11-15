-- +goose Up
CREATE TABLE users_identity_codes (
    id                   UUID        NOT NULL DEFAULT gen_random_uuid(),
    user_id              UUID        NOT NULL,
    session_sha256       TEXT        NOT NULL,
    purpose              TEXT        NOT NULL,
    subject              TEXT        NOT NULL,
    recipient_email      TEXT        NOT NULL,
    hashed_code          TEXT        NOT NULL,
    expires_at           TIMESTAMPTZ NOT NULL,
    is_used              BOOLEAN     NOT NULL DEFAULT FALSE,
    failed_attempt_count INTEGER     NOT NULL DEFAULT 0,
    created_at           TIMESTAMPTZ NOT NULL
);

ALTER TABLE users_identity_codes
    ADD CONSTRAINT pk_users_identity_codes
    PRIMARY KEY (id);

ALTER TABLE users_identity_codes
    ADD CONSTRAINT fk_users_identity_codes_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE CASCADE;

ALTER TABLE users_identity_codes
    ADD CONSTRAINT chk_users_identity_codes_failed_attempt_count
    CHECK (failed_attempt_count >= 0);

CREATE INDEX idx_users_identity_codes_user_id
    ON users_identity_codes (user_id, purpose, created_at DESC);

-- +goose Down
DROP TABLE users_identity_codes;
