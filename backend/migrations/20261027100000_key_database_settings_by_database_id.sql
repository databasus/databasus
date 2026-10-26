-- +goose Up
-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    DROP CONSTRAINT databases_postgresql_logical_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    DROP CONSTRAINT uq_databases_postgresql_logical_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    DROP COLUMN id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ADD CONSTRAINT databases_postgresql_logical_pkey
    PRIMARY KEY (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    DROP CONSTRAINT databases_postgresql_physical_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    DROP CONSTRAINT uq_databases_postgresql_physical_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    DROP COLUMN id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    ADD CONSTRAINT databases_postgresql_physical_pkey
    PRIMARY KEY (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    DROP CONSTRAINT databases_mysql_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    DROP CONSTRAINT uq_databases_mysql_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    DROP COLUMN id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ADD CONSTRAINT databases_mysql_pkey
    PRIMARY KEY (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    DROP CONSTRAINT databases_mariadb_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    DROP CONSTRAINT uq_databases_mariadb_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    DROP COLUMN id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ADD CONSTRAINT databases_mariadb_pkey
    PRIMARY KEY (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    DROP CONSTRAINT databases_mongodb_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    DROP CONSTRAINT uq_databases_mongodb_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    DROP COLUMN id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    ADD CONSTRAINT databases_mongodb_pkey
    PRIMARY KEY (database_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE databases_mongodb
    DROP CONSTRAINT databases_mongodb_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    ADD COLUMN id UUID NOT NULL DEFAULT gen_random_uuid();
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    ADD CONSTRAINT databases_mongodb_pkey
    PRIMARY KEY (id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mongodb
    ADD CONSTRAINT uq_databases_mongodb_database_id
    UNIQUE (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    DROP CONSTRAINT databases_mariadb_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ADD COLUMN id UUID NOT NULL DEFAULT gen_random_uuid();
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ADD CONSTRAINT databases_mariadb_pkey
    PRIMARY KEY (id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mariadb
    ADD CONSTRAINT uq_databases_mariadb_database_id
    UNIQUE (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    DROP CONSTRAINT databases_mysql_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ADD COLUMN id UUID NOT NULL DEFAULT gen_random_uuid();
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ADD CONSTRAINT databases_mysql_pkey
    PRIMARY KEY (id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_mysql
    ADD CONSTRAINT uq_databases_mysql_database_id
    UNIQUE (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    DROP CONSTRAINT databases_postgresql_physical_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    ADD COLUMN id UUID NOT NULL DEFAULT gen_random_uuid();
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    ADD CONSTRAINT databases_postgresql_physical_pkey
    PRIMARY KEY (id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_physical
    ADD CONSTRAINT uq_databases_postgresql_physical_database_id
    UNIQUE (database_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    DROP CONSTRAINT databases_postgresql_logical_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ADD COLUMN id UUID NOT NULL DEFAULT gen_random_uuid();
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ADD CONSTRAINT databases_postgresql_logical_pkey
    PRIMARY KEY (id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE databases_postgresql_logical
    ADD CONSTRAINT uq_databases_postgresql_logical_database_id
    UNIQUE (database_id);
-- +goose StatementEnd
