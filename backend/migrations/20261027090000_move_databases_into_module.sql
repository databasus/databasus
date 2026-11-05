-- +goose Up
-- +goose StatementBegin
ALTER TABLE postgresql_logical_databases
    RENAME TO databases_postgresql_logical;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    RENAME CONSTRAINT postgresql_logical_databases_pkey TO databases_postgresql_logical_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    DROP CONSTRAINT uk_postgresql_logical_databases_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_postgresql_logical_databases_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE postgresql_physical_databases
    RENAME TO databases_postgresql_physical;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    RENAME CONSTRAINT postgresql_physical_databases_pkey TO databases_postgresql_physical_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    DROP CONSTRAINT fk_postgresql_physical_databases_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    DROP CONSTRAINT uk_postgresql_physical_databases_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_postgresql_physical_databases_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE mysql_databases
    RENAME TO databases_mysql;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    RENAME CONSTRAINT mysql_databases_pkey TO databases_mysql_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    DROP CONSTRAINT mysql_databases_database_id_fkey;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_mysql_databases_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE mariadb_databases
    RENAME TO databases_mariadb;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    RENAME CONSTRAINT mariadb_databases_pkey TO databases_mariadb_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    DROP CONSTRAINT mariadb_databases_database_id_fkey;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_mariadb_databases_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE mongodb_databases
    RENAME TO databases_mongodb;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    RENAME CONSTRAINT mongodb_databases_pkey TO databases_mongodb_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    DROP CONSTRAINT mongodb_databases_database_id_fkey;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_mongodb_databases_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE database_notifiers
    RENAME TO databases_notifiers;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_notifiers
    RENAME CONSTRAINT database_notifiers_pkey TO databases_notifiers_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_notifiers
    RENAME CONSTRAINT fk_database_notifiers_database_id TO fk_databases_notifiers_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_notifiers
    RENAME CONSTRAINT fk_database_notifiers_notifier_id TO fk_databases_notifiers_notifier_id;
-- +goose StatementEnd

-- The primary key starts with database_id, so it serves the lookups by database.
-- +goose StatementBegin
DROP INDEX idx_database_notifiers_database_id;
-- +goose StatementEnd

-- No release has written a type outside the five. Deleting a referenced database of another type
-- would leave its backup files without an owner, because SQL cannot ask a storage to delete them.
-- +goose StatementBegin
DO $$
DECLARE
    unknown_database_id UUID;
BEGIN
    SELECT d.id
    INTO unknown_database_id
    FROM databases d
    WHERE d.type NOT IN ('POSTGRES_LOGICAL', 'POSTGRES_PHYSICAL', 'MYSQL', 'MARIADB', 'MONGODB')
        AND (
        EXISTS (SELECT 1 FROM logical_backup_configs r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM logical_backups r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM physical_backup_configs r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM physical_full_backups r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM physical_incremental_backups r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM physical_wal_segments r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM physical_wal_history_files r WHERE r.database_id = d.id)
    )
    ORDER BY d.id
    LIMIT 1;

    IF unknown_database_id IS NOT NULL THEN
        RAISE EXCEPTION 'database % has an unknown type and backups or a backup config reference it; remove it by hand before upgrading', unknown_database_id;
    END IF;
END $$;
-- +goose StatementEnd

-- Nothing references such a database any more, and nothing could work with it.
-- +goose StatementBegin
DELETE FROM databases d
WHERE d.type NOT IN ('POSTGRES_LOGICAL', 'POSTGRES_PHYSICAL', 'MYSQL', 'MARIADB', 'MONGODB');
-- +goose StatementEnd

-- A provider row whose database has another type is never read.
-- +goose StatementBegin
DELETE FROM databases_postgresql_logical p
USING databases d
WHERE d.id = p.database_id
    AND d.type <> 'POSTGRES_LOGICAL';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM databases_postgresql_physical p
USING databases d
WHERE d.id = p.database_id
    AND d.type <> 'POSTGRES_PHYSICAL';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM databases_mysql p
USING databases d
WHERE d.id = p.database_id
    AND d.type <> 'MYSQL';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM databases_mariadb p
USING databases d
WHERE d.id = p.database_id
    AND d.type <> 'MARIADB';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM databases_mongodb p
USING databases d
WHERE d.id = p.database_id
    AND d.type <> 'MONGODB';
-- +goose StatementEnd

-- Only a create that reused the ID of another database wrote a second provider row; the old read
-- picked one of them at random, and the row with the lowest ID stays.
-- +goose StatementBegin
DELETE FROM databases_postgresql_logical p
USING databases_postgresql_logical kept
WHERE kept.database_id = p.database_id
    AND kept.id < p.id;
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM databases_postgresql_physical p
USING databases_postgresql_physical kept
WHERE kept.database_id = p.database_id
    AND kept.id < p.id;
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM databases_mysql p
USING databases_mysql kept
WHERE kept.database_id = p.database_id
    AND kept.id < p.id;
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM databases_mariadb p
USING databases_mariadb kept
WHERE kept.database_id = p.database_id
    AND kept.id < p.id;
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM databases_mongodb p
USING databases_mongodb kept
WHERE kept.database_id = p.database_id
    AND kept.id < p.id;
-- +goose StatementEnd

-- A database without its provider row cannot connect. It goes unless backups or a backup config
-- reference it; such a database keeps its parent row, and the module answers a read of it as an internal
-- error.
-- +goose StatementBegin
DELETE FROM databases d
WHERE NOT (
        (d.type = 'POSTGRES_LOGICAL' AND EXISTS (SELECT 1 FROM databases_postgresql_logical p WHERE p.database_id = d.id))
        OR (d.type = 'POSTGRES_PHYSICAL' AND EXISTS (SELECT 1 FROM databases_postgresql_physical p WHERE p.database_id = d.id))
        OR (d.type = 'MYSQL' AND EXISTS (SELECT 1 FROM databases_mysql p WHERE p.database_id = d.id))
        OR (d.type = 'MARIADB' AND EXISTS (SELECT 1 FROM databases_mariadb p WHERE p.database_id = d.id))
        OR (d.type = 'MONGODB' AND EXISTS (SELECT 1 FROM databases_mongodb p WHERE p.database_id = d.id))
    )
    AND NOT (
        EXISTS (SELECT 1 FROM logical_backup_configs r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM logical_backups r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM physical_backup_configs r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM physical_full_backups r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM physical_incremental_backups r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM physical_wal_segments r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM physical_wal_history_files r WHERE r.database_id = d.id)
    );
-- +goose StatementEnd

-- A database that restores created before workspaces existed joins the oldest workspace when it has
-- backups or a backup config.
-- +goose StatementBegin
UPDATE databases d
SET workspace_id = (SELECT id FROM workspaces ORDER BY created_at, id LIMIT 1)
WHERE d.workspace_id IS NULL
    AND (
        EXISTS (SELECT 1 FROM logical_backup_configs r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM logical_backups r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM physical_backup_configs r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM physical_full_backups r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM physical_incremental_backups r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM physical_wal_segments r WHERE r.database_id = d.id)
        OR EXISTS (SELECT 1 FROM physical_wal_history_files r WHERE r.database_id = d.id)
    );
-- +goose StatementEnd

-- Any other database without a workspace has not been shown since workspaces appeared.
-- +goose StatementBegin
DELETE FROM databases
WHERE workspace_id IS NULL;
-- +goose StatementEnd

-- A provider row without its database is never read: the logical table had no foreign key, and the
-- deletions above leave its rows behind.
-- +goose StatementBegin
DELETE FROM databases_postgresql_logical p
WHERE p.database_id IS NULL
    OR NOT EXISTS (SELECT 1 FROM databases d WHERE d.id = p.database_id);
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM databases_postgresql_physical p
WHERE p.database_id IS NULL
    OR NOT EXISTS (SELECT 1 FROM databases d WHERE d.id = p.database_id);
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM databases_mysql p
WHERE p.database_id IS NULL
    OR NOT EXISTS (SELECT 1 FROM databases d WHERE d.id = p.database_id);
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM databases_mariadb p
WHERE p.database_id IS NULL
    OR NOT EXISTS (SELECT 1 FROM databases d WHERE d.id = p.database_id);
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM databases_mongodb p
WHERE p.database_id IS NULL
    OR NOT EXISTS (SELECT 1 FROM databases d WHERE d.id = p.database_id);
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE databases
SET health_status = NULL
WHERE health_status NOT IN ('AVAILABLE', 'UNAVAILABLE');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE databases_postgresql_logical
SET ssl_mode = 'disable'
WHERE ssl_mode NOT IN ('disable', 'require', 'verify-ca', 'verify-full');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE databases_postgresql_physical
SET ssl_mode = 'disable'
WHERE ssl_mode NOT IN ('disable', 'require', 'verify-ca', 'verify-full');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE databases_postgresql_logical
SET ssh_auth_type = 'PASSWORD'
WHERE ssh_auth_type NOT IN ('PASSWORD', 'PRIVATE_KEY');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE databases_postgresql_physical
SET ssh_auth_type = 'PASSWORD'
WHERE ssh_auth_type NOT IN ('PASSWORD', 'PRIVATE_KEY');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE databases_mysql
SET ssh_auth_type = 'PASSWORD'
WHERE ssh_auth_type NOT IN ('PASSWORD', 'PRIVATE_KEY');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE databases_mariadb
SET ssh_auth_type = 'PASSWORD'
WHERE ssh_auth_type NOT IN ('PASSWORD', 'PRIVATE_KEY');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE databases_mongodb
SET ssh_auth_type = 'PASSWORD'
WHERE ssh_auth_type NOT IN ('PASSWORD', 'PRIVATE_KEY');
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE databases_postgresql_physical
SET backup_type = 'FULL'
WHERE backup_type NOT IN ('FULL', 'FULL_INCREMENTAL', 'FULL_INCREMENTAL_WAL_STREAM');
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases
    ALTER COLUMN workspace_id SET NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases
    ADD CONSTRAINT uq_databases_id_type
    UNIQUE (id, type);
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_databases_workspace_id;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_databases_workspace_id_name_id
    ON databases (workspace_id, name, id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN database_id SET NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN id SET DEFAULT gen_random_uuid();
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ADD CONSTRAINT uq_databases_postgresql_logical_database_id
    UNIQUE (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ADD COLUMN type TEXT NOT NULL DEFAULT 'POSTGRES_LOGICAL';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ADD CONSTRAINT fk_databases_postgresql_logical_database
    FOREIGN KEY (database_id, type)
    REFERENCES databases (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    ALTER COLUMN database_id SET NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    ALTER COLUMN id SET DEFAULT gen_random_uuid();
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    ADD CONSTRAINT uq_databases_postgresql_physical_database_id
    UNIQUE (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    ADD COLUMN type TEXT NOT NULL DEFAULT 'POSTGRES_PHYSICAL';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    ADD CONSTRAINT fk_databases_postgresql_physical_database
    FOREIGN KEY (database_id, type)
    REFERENCES databases (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ALTER COLUMN database_id SET NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ALTER COLUMN id SET DEFAULT gen_random_uuid();
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ADD CONSTRAINT uq_databases_mysql_database_id
    UNIQUE (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ADD COLUMN type TEXT NOT NULL DEFAULT 'MYSQL';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ADD CONSTRAINT fk_databases_mysql_database
    FOREIGN KEY (database_id, type)
    REFERENCES databases (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ALTER COLUMN database_id SET NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ALTER COLUMN id SET DEFAULT gen_random_uuid();
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ADD CONSTRAINT uq_databases_mariadb_database_id
    UNIQUE (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ADD COLUMN type TEXT NOT NULL DEFAULT 'MARIADB';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ADD CONSTRAINT fk_databases_mariadb_database
    FOREIGN KEY (database_id, type)
    REFERENCES databases (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    ALTER COLUMN database_id SET NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    ALTER COLUMN id SET DEFAULT gen_random_uuid();
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    ADD CONSTRAINT uq_databases_mongodb_database_id
    UNIQUE (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    ADD COLUMN type TEXT NOT NULL DEFAULT 'MONGODB';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    ADD CONSTRAINT fk_databases_mongodb_database
    FOREIGN KEY (database_id, type)
    REFERENCES databases (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- The per-backup replication slot was named after the row's own id, which a later migration drops.
-- The column keeps every existing name, so a slot left on a source server by an interrupted backup
-- is still found and dropped by the next backup.
-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    ADD COLUMN backup_slot_name TEXT;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE databases_postgresql_physical
SET backup_slot_name = 'databasus_basebackup_' || replace(id::text, '-', '');
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    ALTER COLUMN backup_slot_name SET NOT NULL;
-- +goose StatementEnd

-- The lists were comma-separated text that the old code parsed on commas, newlines, carriage returns
-- and tabs, trimmed, and stripped of empty entries and later duplicates. The arrays keep that reading.
-- +goose StatementBegin
CREATE FUNCTION databases_split_names(raw_names TEXT) RETURNS TEXT[]
    LANGUAGE SQL
    IMMUTABLE
AS $function$
    SELECT COALESCE(array_agg(name ORDER BY first_position), '{}')
    FROM (
        SELECT btrim(part, E' \t\n\r\v\f') AS name, min(position) AS first_position
        FROM regexp_split_to_table(raw_names, E'[,\n\r\t]') WITH ORDINALITY AS parts (part, position)
        GROUP BY btrim(part, E' \t\n\r\v\f')
    ) names
    WHERE name <> ''
$function$;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN include_schemas DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN include_schemas TYPE TEXT[] USING databases_split_names(include_schemas);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN include_schemas SET DEFAULT '{}';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN exclude_tables DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN exclude_tables TYPE TEXT[] USING databases_split_names(exclude_tables);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN exclude_tables SET DEFAULT '{}';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN exclude_table_data DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN exclude_table_data TYPE TEXT[] USING databases_split_names(exclude_table_data);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN exclude_table_data SET DEFAULT '{}';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ALTER COLUMN exclude_tables DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ALTER COLUMN exclude_tables TYPE TEXT[] USING databases_split_names(exclude_tables);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ALTER COLUMN exclude_tables SET DEFAULT '{}';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ALTER COLUMN privileges DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ALTER COLUMN privileges TYPE TEXT[] USING databases_split_names(privileges);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ALTER COLUMN privileges SET DEFAULT '{}';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ALTER COLUMN exclude_tables DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ALTER COLUMN exclude_tables TYPE TEXT[] USING databases_split_names(exclude_tables);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ALTER COLUMN exclude_tables SET DEFAULT '{}';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ALTER COLUMN privileges DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ALTER COLUMN privileges TYPE TEXT[] USING databases_split_names(privileges);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ALTER COLUMN privileges SET DEFAULT '{}';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    ALTER COLUMN exclude_collections DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    ALTER COLUMN exclude_collections TYPE TEXT[] USING databases_split_names(exclude_collections);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    ALTER COLUMN exclude_collections SET DEFAULT '{}';
-- +goose StatementEnd

-- +goose StatementBegin
DROP FUNCTION databases_split_names(TEXT);
-- +goose StatementEnd

-- Physical backups write the last backup time from now on; until now the list computed it from the
-- newest completed full or incremental backup and the newest committed WAL segment.
-- +goose StatementBegin
UPDATE databases d
SET last_backup_time = latest.backup_time
FROM (
    SELECT database_id, MAX(backup_time) AS backup_time
    FROM (
        SELECT database_id, created_at AS backup_time
        FROM physical_full_backups
        WHERE status = 'COMPLETED'
        UNION ALL
        SELECT database_id, created_at
        FROM physical_incremental_backups
        WHERE status = 'COMPLETED'
        UNION ALL
        SELECT database_id, received_at
        FROM physical_wal_segments
        WHERE file_name IS NOT NULL
    ) backups
    GROUP BY database_id
) latest
WHERE d.id = latest.database_id
    AND d.type = 'POSTGRES_PHYSICAL';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN include_schemas DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN include_schemas TYPE TEXT USING array_to_string(include_schemas, ',');
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN include_schemas SET DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN exclude_tables DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN exclude_tables TYPE TEXT USING array_to_string(exclude_tables, ',');
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN exclude_tables SET DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN exclude_table_data DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN exclude_table_data TYPE TEXT USING array_to_string(exclude_table_data, ',');
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN exclude_table_data SET DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ALTER COLUMN exclude_tables DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ALTER COLUMN exclude_tables TYPE TEXT USING array_to_string(exclude_tables, ',');
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ALTER COLUMN exclude_tables SET DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ALTER COLUMN privileges DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ALTER COLUMN privileges TYPE TEXT USING array_to_string(privileges, ',');
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ALTER COLUMN privileges SET DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ALTER COLUMN exclude_tables DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ALTER COLUMN exclude_tables TYPE TEXT USING array_to_string(exclude_tables, ',');
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ALTER COLUMN exclude_tables SET DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ALTER COLUMN privileges DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ALTER COLUMN privileges TYPE TEXT USING array_to_string(privileges, ',');
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ALTER COLUMN privileges SET DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    ALTER COLUMN exclude_collections DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    ALTER COLUMN exclude_collections TYPE TEXT USING array_to_string(exclude_collections, ',');
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    ALTER COLUMN exclude_collections SET DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    DROP COLUMN backup_slot_name;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    DROP CONSTRAINT fk_databases_postgresql_logical_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    DROP CONSTRAINT uq_databases_postgresql_logical_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ALTER COLUMN database_id DROP NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    DROP CONSTRAINT fk_databases_postgresql_physical_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    DROP CONSTRAINT uq_databases_postgresql_physical_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    ALTER COLUMN database_id DROP NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    DROP CONSTRAINT fk_databases_mysql_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    DROP CONSTRAINT uq_databases_mysql_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ALTER COLUMN database_id DROP NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    DROP CONSTRAINT fk_databases_mariadb_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    DROP CONSTRAINT uq_databases_mariadb_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ALTER COLUMN database_id DROP NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    DROP CONSTRAINT fk_databases_mongodb_database;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    DROP CONSTRAINT uq_databases_mongodb_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    ALTER COLUMN database_id DROP NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_databases_workspace_id_name_id;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_databases_workspace_id
    ON databases (workspace_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases
    DROP CONSTRAINT uq_databases_id_type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases
    ALTER COLUMN workspace_id DROP NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_database_notifiers_database_id
    ON databases_notifiers (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_notifiers
    RENAME CONSTRAINT fk_databases_notifiers_notifier_id TO fk_database_notifiers_notifier_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_notifiers
    RENAME CONSTRAINT fk_databases_notifiers_database_id TO fk_database_notifiers_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_notifiers
    RENAME CONSTRAINT databases_notifiers_pkey TO database_notifiers_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_notifiers
    RENAME TO database_notifiers;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_mongodb_databases_database_id
    ON databases_mongodb (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    ADD CONSTRAINT mongodb_databases_database_id_fkey
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    RENAME CONSTRAINT databases_mongodb_pkey TO mongodb_databases_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    RENAME TO mongodb_databases;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_mariadb_databases_database_id
    ON databases_mariadb (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ADD CONSTRAINT mariadb_databases_database_id_fkey
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    RENAME CONSTRAINT databases_mariadb_pkey TO mariadb_databases_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    RENAME TO mariadb_databases;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_mysql_databases_database_id
    ON databases_mysql (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ADD CONSTRAINT mysql_databases_database_id_fkey
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    RENAME CONSTRAINT databases_mysql_pkey TO mysql_databases_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    RENAME TO mysql_databases;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_postgresql_physical_databases_database_id
    ON databases_postgresql_physical (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    ADD CONSTRAINT uk_postgresql_physical_databases_database_id
    UNIQUE (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    ADD CONSTRAINT fk_postgresql_physical_databases_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    RENAME CONSTRAINT databases_postgresql_physical_pkey TO postgresql_physical_databases_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    RENAME TO postgresql_physical_databases;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_postgresql_logical_databases_database_id
    ON databases_postgresql_logical (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ADD CONSTRAINT uk_postgresql_logical_databases_database_id
    UNIQUE (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    RENAME CONSTRAINT databases_postgresql_logical_pkey TO postgresql_logical_databases_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    RENAME TO postgresql_logical_databases;
-- +goose StatementEnd
