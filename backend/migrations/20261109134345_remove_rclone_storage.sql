-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM storages WHERE type = 'RCLONE') THEN
        RAISE EXCEPTION 'Databasus 3.67.1 supports rclone storages. Roll back to Databasus 3.67.1 and remove all rclone storages before upgrading, or continue using 3.67.1.';
    END IF;
END;
$$;
-- +goose StatementEnd

DROP TABLE IF EXISTS storages_rclone;

DROP TABLE IF EXISTS rclone_storages;

-- +goose Down
CREATE TABLE storages_rclone (
    storage_id     UUID PRIMARY KEY,
    type           TEXT NOT NULL DEFAULT 'RCLONE',
    config_content TEXT NOT NULL,
    remote_path    TEXT
);

ALTER TABLE storages_rclone
    ADD CONSTRAINT fk_storages_rclone_storage
    FOREIGN KEY (storage_id, type)
    REFERENCES storages (id, type)
    ON DELETE CASCADE;
