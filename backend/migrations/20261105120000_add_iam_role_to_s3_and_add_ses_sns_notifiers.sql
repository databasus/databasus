-- +goose Up
-- +goose StatementBegin

ALTER TABLE s3_storages
    ADD COLUMN auth_type TEXT NOT NULL DEFAULT 'KEYS';

ALTER TABLE s3_storages
    ADD COLUMN role_arn TEXT NOT NULL DEFAULT '';

ALTER TABLE s3_storages
    ADD COLUMN external_id TEXT NOT NULL DEFAULT '';

ALTER TABLE s3_storages
    ALTER COLUMN s3_access_key DROP NOT NULL;

ALTER TABLE s3_storages
    ALTER COLUMN s3_access_key SET DEFAULT '';

ALTER TABLE s3_storages
    ALTER COLUMN s3_secret_key DROP NOT NULL;

ALTER TABLE s3_storages
    ALTER COLUMN s3_secret_key SET DEFAULT '';

CREATE TABLE ses_notifiers (
    notifier_id  UUID PRIMARY KEY,
    from_email   TEXT NOT NULL,
    target_email TEXT NOT NULL,
    region       TEXT NOT NULL,
    auth_type    TEXT NOT NULL DEFAULT 'IAM_ROLE',
    access_key   TEXT NOT NULL DEFAULT '',
    secret_key   TEXT NOT NULL DEFAULT '',
    role_arn     TEXT NOT NULL DEFAULT '',
    external_id  TEXT NOT NULL DEFAULT ''
);

ALTER TABLE ses_notifiers
    ADD CONSTRAINT fk_ses_notifiers_notifier
    FOREIGN KEY (notifier_id)
    REFERENCES notifiers (id)
    ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE sns_notifiers (
    notifier_id UUID PRIMARY KEY,
    topic_arn   TEXT NOT NULL,
    region      TEXT NOT NULL,
    auth_type   TEXT NOT NULL DEFAULT 'KEYS',
    access_key  TEXT NOT NULL DEFAULT '',
    secret_key  TEXT NOT NULL DEFAULT '',
    role_arn    TEXT NOT NULL DEFAULT '',
    external_id TEXT NOT NULL DEFAULT ''
);

ALTER TABLE sns_notifiers
    ADD CONSTRAINT fk_sns_notifiers_notifier
    FOREIGN KEY (notifier_id)
    REFERENCES notifiers (id)
    ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS sns_notifiers;

DROP TABLE IF EXISTS ses_notifiers;

ALTER TABLE s3_storages
    DROP COLUMN IF EXISTS external_id;

ALTER TABLE s3_storages
    DROP COLUMN IF EXISTS role_arn;

ALTER TABLE s3_storages
    DROP COLUMN IF EXISTS auth_type;

UPDATE s3_storages
    SET s3_access_key = ''
    WHERE s3_access_key IS NULL;

UPDATE s3_storages
    SET s3_secret_key = ''
    WHERE s3_secret_key IS NULL;

ALTER TABLE s3_storages
    ALTER COLUMN s3_access_key SET NOT NULL;

ALTER TABLE s3_storages
    ALTER COLUMN s3_secret_key SET NOT NULL;

-- +goose StatementEnd
