-- +goose Up
ALTER TABLE healthcheck_configs RENAME COLUMN is_sent_notification_when_unavailable TO is_notification_enabled;

ALTER TABLE healthcheck_configs RENAME COLUMN attempts_before_considered_as_down TO failed_attempts_before_unavailable;

ALTER TABLE healthcheck_configs RENAME COLUMN store_attempts_days TO attempt_retention_days;

UPDATE healthcheck_configs SET interval_minutes = 1 WHERE interval_minutes < 1;

UPDATE healthcheck_configs SET failed_attempts_before_unavailable = 3 WHERE failed_attempts_before_unavailable < 1;

UPDATE healthcheck_configs SET attempt_retention_days = 7 WHERE attempt_retention_days < 1;

DELETE FROM healthcheck_attempts WHERE status NOT IN ('AVAILABLE', 'UNAVAILABLE');

DELETE FROM healthcheck_configs WHERE database_id NOT IN (SELECT id FROM databases);

DELETE FROM healthcheck_attempts WHERE database_id NOT IN (SELECT id FROM databases);

INSERT INTO healthcheck_configs (database_id, is_healthcheck_enabled, is_notification_enabled, interval_minutes, failed_attempts_before_unavailable, attempt_retention_days)
SELECT id, TRUE, TRUE, 1, 3, 7 FROM databases
WHERE id NOT IN (SELECT database_id FROM healthcheck_configs);

SET CONSTRAINTS ALL IMMEDIATE;

ALTER TABLE healthcheck_configs
    ADD CONSTRAINT ck_healthcheck_configs_interval_minutes
    CHECK (interval_minutes >= 1);

ALTER TABLE healthcheck_configs
    ADD CONSTRAINT ck_healthcheck_configs_failed_attempts_before_unavailable
    CHECK (failed_attempts_before_unavailable >= 1);

ALTER TABLE healthcheck_configs
    ADD CONSTRAINT ck_healthcheck_configs_attempt_retention_days
    CHECK (attempt_retention_days >= 1);

ALTER TABLE healthcheck_configs DROP CONSTRAINT fk_healthcheck_configs_database_id;

ALTER TABLE healthcheck_configs
    ADD CONSTRAINT fk_healthcheck_configs_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE RESTRICT;

ALTER TABLE healthcheck_attempts DROP CONSTRAINT fk_healthcheck_attempts_database_id;

ALTER TABLE healthcheck_attempts
    ADD CONSTRAINT fk_healthcheck_attempts_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE RESTRICT;

DROP INDEX idx_healthcheck_configs_database_id;

-- +goose Down
ALTER TABLE healthcheck_configs DROP CONSTRAINT fk_healthcheck_configs_database_id;

ALTER TABLE healthcheck_configs
    ADD CONSTRAINT fk_healthcheck_configs_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE healthcheck_attempts DROP CONSTRAINT fk_healthcheck_attempts_database_id;

ALTER TABLE healthcheck_attempts
    ADD CONSTRAINT fk_healthcheck_attempts_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE healthcheck_configs DROP CONSTRAINT ck_healthcheck_configs_interval_minutes;

ALTER TABLE healthcheck_configs DROP CONSTRAINT ck_healthcheck_configs_failed_attempts_before_unavailable;

ALTER TABLE healthcheck_configs DROP CONSTRAINT ck_healthcheck_configs_attempt_retention_days;

ALTER TABLE healthcheck_configs RENAME COLUMN is_notification_enabled TO is_sent_notification_when_unavailable;

ALTER TABLE healthcheck_configs RENAME COLUMN failed_attempts_before_unavailable TO attempts_before_considered_as_down;

ALTER TABLE healthcheck_configs RENAME COLUMN attempt_retention_days TO store_attempts_days;

CREATE INDEX idx_healthcheck_configs_database_id ON healthcheck_configs (database_id);
