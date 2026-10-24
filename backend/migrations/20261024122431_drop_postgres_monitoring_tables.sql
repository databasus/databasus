-- +goose Up
-- +goose StatementBegin
DROP TABLE postgres_monitoring_metrics;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE postgres_monitoring_settings;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE postgres_monitoring_settings (
    database_id                        UUID PRIMARY KEY,
    is_db_resources_monitoring_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    monitoring_interval_seconds        BIGINT  NOT NULL DEFAULT 60,
    installed_extensions_raw           TEXT
);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE postgres_monitoring_settings
    ADD CONSTRAINT fk_postgres_monitoring_settings_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE postgres_monitoring_metrics (
    id          UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id UUID             NOT NULL,
    metric      TEXT             NOT NULL,
    value_type  TEXT             NOT NULL,
    value       DOUBLE PRECISION NOT NULL,
    created_at  TIMESTAMPTZ      NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE postgres_monitoring_metrics
    ADD CONSTRAINT fk_postgres_monitoring_metrics_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_postgres_monitoring_metrics_database_id
    ON postgres_monitoring_metrics (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_postgres_monitoring_metrics_created_at
    ON postgres_monitoring_metrics (created_at);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_postgres_monitoring_metrics_database_metric_created_at
    ON postgres_monitoring_metrics (database_id, metric, created_at);
-- +goose StatementEnd
