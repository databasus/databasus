-- +goose Up
-- Backup rows are the only record of the files in a storage, so the database refuses to delete a
-- storage they or a backup config still reference. RESTRICT, unlike NO ACTION, cannot be deferred,
-- so the refusal comes at the DELETE, before the deletion removes any file.
-- +goose StatementBegin
ALTER TABLE logical_backup_configs
    DROP CONSTRAINT fk_backup_config_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE logical_backup_configs
    ADD CONSTRAINT fk_backup_config_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE logical_backups
    DROP CONSTRAINT fk_backups_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE logical_backups
    ADD CONSTRAINT fk_backups_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_backup_configs
    DROP CONSTRAINT fk_physical_backup_configs_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_backup_configs
    ADD CONSTRAINT fk_physical_backup_configs_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_full_backups
    DROP CONSTRAINT fk_physical_full_backups_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_full_backups
    ADD CONSTRAINT fk_physical_full_backups_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_incremental_backups
    DROP CONSTRAINT fk_physical_incremental_backups_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_incremental_backups
    ADD CONSTRAINT fk_physical_incremental_backups_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_wal_segments
    DROP CONSTRAINT fk_physical_wal_segments_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_wal_segments
    ADD CONSTRAINT fk_physical_wal_segments_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_wal_history_files
    DROP CONSTRAINT fk_physical_wal_history_files_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_wal_history_files
    ADD CONSTRAINT fk_physical_wal_history_files_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE RESTRICT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE logical_backup_configs
    DROP CONSTRAINT fk_backup_config_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE logical_backup_configs
    ADD CONSTRAINT fk_backup_config_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE logical_backups
    DROP CONSTRAINT fk_backups_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE logical_backups
    ADD CONSTRAINT fk_backups_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_backup_configs
    DROP CONSTRAINT fk_physical_backup_configs_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_backup_configs
    ADD CONSTRAINT fk_physical_backup_configs_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_full_backups
    DROP CONSTRAINT fk_physical_full_backups_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_full_backups
    ADD CONSTRAINT fk_physical_full_backups_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_incremental_backups
    DROP CONSTRAINT fk_physical_incremental_backups_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_incremental_backups
    ADD CONSTRAINT fk_physical_incremental_backups_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_wal_segments
    DROP CONSTRAINT fk_physical_wal_segments_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_wal_segments
    ADD CONSTRAINT fk_physical_wal_segments_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_wal_history_files
    DROP CONSTRAINT fk_physical_wal_history_files_storage_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE physical_wal_history_files
    ADD CONSTRAINT fk_physical_wal_history_files_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id);
-- +goose StatementEnd
