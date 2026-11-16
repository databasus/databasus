-- +goose Up
ALTER TABLE backups_physical_wal_history_files
    ADD COLUMN parent_timeline_id INT;

ALTER TABLE backups_physical_wal_history_files
    ADD COLUMN fork_lsn PG_LSN;

ALTER TABLE backups_physical_wal_history_files
    ADD COLUMN is_conflicting BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE backups_physical_wal_history_files
    ADD CONSTRAINT ck_backups_physical_wal_history_files_fork
    CHECK (num_nonnulls(parent_timeline_id, fork_lsn) IN (0, 2));

-- +goose Down
ALTER TABLE backups_physical_wal_history_files
    DROP CONSTRAINT ck_backups_physical_wal_history_files_fork;

ALTER TABLE backups_physical_wal_history_files
    DROP COLUMN is_conflicting;

ALTER TABLE backups_physical_wal_history_files
    DROP COLUMN fork_lsn;

ALTER TABLE backups_physical_wal_history_files
    DROP COLUMN parent_timeline_id;
