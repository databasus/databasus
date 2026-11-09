-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM storages WHERE type = 'GOOGLE_DRIVE') THEN
        RAISE EXCEPTION 'Google Drive support ended after Databasus 3.67.0. Roll back to Databasus 3.67.0 and remove all Google Drive storages before upgrading, or continue using 3.67.0.';
    END IF;
END;
$$;
-- +goose StatementEnd

DROP TABLE IF EXISTS storages_google_drive;

DROP TABLE IF EXISTS google_drive_storages;

-- +goose Down
CREATE TABLE storages_google_drive (
    storage_id    UUID PRIMARY KEY,
    type          TEXT NOT NULL DEFAULT 'GOOGLE_DRIVE',
    client_id     TEXT NOT NULL,
    client_secret TEXT NOT NULL,
    token_json    TEXT NOT NULL
);

ALTER TABLE storages_google_drive
    ADD CONSTRAINT fk_storages_google_drive_storage
    FOREIGN KEY (storage_id, type)
    REFERENCES storages (id, type)
    ON DELETE CASCADE;
