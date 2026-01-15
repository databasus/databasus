-- +goose Up
-- +goose StatementBegin
CREATE TABLE mongodb_databases (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id       UUID REFERENCES databases(id) ON DELETE CASCADE,
    version           TEXT NOT NULL,
    host              TEXT NOT NULL,
    port              INT NOT NULL,
    username          TEXT NOT NULL,
    password          TEXT NOT NULL,
    tls_ca_file       TEXT NOT NULL,
    tls_cert_file     TEXT NOT NULL,  
    tls_cert_key_file TEXT NOT NULL,
    database          TEXT NOT NULL,
    auth_database     TEXT NOT NULL DEFAULT 'admin',
    is_https          BOOLEAN NOT NULL DEFAULT FALSE
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_mongodb_databases_database_id ON mongodb_databases(database_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_mongodb_databases_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS mongodb_databases;
-- +goose StatementEnd
