package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"gorm.io/gorm"
)

// migrationHistoryTable is goose's history. It is the one table a fresh schema always populates.
const migrationHistoryTable = "public.goose_db_version"

// HasTable reports whether a table exists, so a reader of an older or a fresh schema can tell a
// missing table from an empty one. The name may be qualified by its schema.
func HasTable(ctx context.Context, db *gorm.DB, name string) (bool, error) {
	var isPresent bool

	err := Conn(ctx, db).Raw("SELECT to_regclass(?) IS NOT NULL", name).Scan(&isPresent).Error
	if err != nil {
		return false, fmt.Errorf("check for table %s: %w", name, err)
	}

	return isPresent, nil
}

// ListPopulatedTables returns every table outside the system schemas that holds at least one row,
// as schema.table, leaving out the migration history. It reads contents only, so it classifies a
// database of any schema version without migrating it.
func ListPopulatedTables(ctx context.Context, db *gorm.DB) ([]string, error) {
	var tables []struct {
		SchemaName string
		TableName  string
	}

	err := Conn(ctx, db).Raw(`SELECT n.nspname AS schema_name, c.relname AS table_name
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r', 'p')
		  AND n.nspname NOT IN ('pg_catalog', 'information_schema')
		  AND n.nspname NOT LIKE 'pg\_toast%'
		  AND n.nspname NOT LIKE 'pg\_temp\_%'
		ORDER BY n.nspname, c.relname`).Scan(&tables).Error
	if err != nil {
		return nil, fmt.Errorf("list the tables: %w", err)
	}

	populated := []string{}

	for _, table := range tables {
		qualifiedName := table.SchemaName + "." + table.TableName
		if qualifiedName == migrationHistoryTable {
			continue
		}

		var hasRows bool

		err := Conn(ctx, db).
			Raw("SELECT EXISTS (SELECT 1 FROM " + pgx.Identifier{table.SchemaName, table.TableName}.Sanitize() + ")").
			Scan(&hasRows).Error
		if err != nil {
			return nil, fmt.Errorf("check table %s for rows: %w", qualifiedName, err)
		}

		if hasRows {
			populated = append(populated, qualifiedName)
		}
	}

	return populated, nil
}

// ReadRecordedMigrationVersions reads the applied versions of a database that may never have been
// migrated, without creating the history table.
func ReadRecordedMigrationVersions(ctx context.Context, db *gorm.DB) ([]int64, error) {
	isPresent, err := HasTable(ctx, db, migrationHistoryTable)
	if err != nil {
		return nil, err
	}

	if !isPresent {
		return []int64{}, nil
	}

	return ReadAppliedMigrationVersions(ctx, db)
}
