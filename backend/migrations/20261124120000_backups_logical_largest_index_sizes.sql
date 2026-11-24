-- +goose Up
ALTER TABLE backups_logical
    ADD COLUMN backup_largest_index_sizes_mb DOUBLE PRECISION[];

ALTER TABLE backups_logical
    ADD CONSTRAINT chk_backups_logical_largest_index_sizes_mb
    CHECK (
        cardinality(backup_largest_index_sizes_mb) <= 8
        AND array_position(backup_largest_index_sizes_mb, NULL) IS NULL
        AND 0 <= ALL(backup_largest_index_sizes_mb)
    );

-- +goose Down
ALTER TABLE backups_logical
    DROP COLUMN backup_largest_index_sizes_mb;
