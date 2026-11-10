package physicaltest

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest/enginetest"
)

type SchemaRecovery struct {
	Namespace      string
	AddedNamespace string
}

func NewSchemaRecovery(t *testing.T, conn *pgx.Conn) SchemaRecovery {
	t.Helper()

	suffix := enginetest.NewUniqueSuffix()
	schema := SchemaRecovery{Namespace: "recovery_" + suffix, AddedNamespace: "added_" + suffix}

	schema.execute(t, conn, `CREATE SCHEMA {base};
		CREATE TABLE {base}.accounts (id SERIAL PRIMARY KEY, name TEXT NOT NULL UNIQUE, amount INTEGER NOT NULL);
		INSERT INTO {base}.accounts (name, amount) VALUES ('baseline', 7);
		CREATE TABLE {base}.rename_before (payload TEXT);
		CREATE TABLE {base}.drop_before (payload TEXT);
		CREATE TABLE {base}.commit_rename_before (payload TEXT);
		CREATE TABLE {base}.commit_drop (payload TEXT);
		CREATE TABLE {base}.late_drop (payload TEXT)`)

	return schema
}

func (schema SchemaRecovery) CreatePostFullObjects(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	schema.execute(t, conn, `CREATE SCHEMA {added};
		CREATE TABLE {added}.orders (
			id BIGINT GENERATED ALWAYS AS IDENTITY (START WITH 101 CACHE 32) PRIMARY KEY,
			name TEXT NOT NULL UNIQUE, amount INTEGER NOT NULL CONSTRAINT positive_amount CHECK (amount > 0));
		INSERT INTO {added}.orders (name, amount) VALUES ('after-full', 11);
		ALTER TABLE {base}.accounts ADD COLUMN note TEXT NOT NULL DEFAULT 'ready';
		CREATE INDEX accounts_amount_idx ON {base}.accounts (amount);
		INSERT INTO {base}.accounts (name, amount) VALUES ('after-full', 13)`)
}

func (schema SchemaRecovery) ApplyBetweenIncrementals(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	schema.execute(t, conn, `ALTER TABLE {base}.accounts RENAME COLUMN note TO label;
		ALTER TABLE {base}.accounts ALTER COLUMN amount TYPE TEXT USING ('units:' || amount::TEXT);
		ALTER TABLE {base}.rename_before RENAME TO renamed_before;
		DROP TABLE {base}.drop_before;
		INSERT INTO {base}.accounts (name, amount) VALUES ('between-incrementals', 'units:17')`)
	schema.execute(t, conn, `BEGIN;
		CREATE TABLE {added}.rolled_back (payload TEXT);
		ALTER TABLE {base}.accounts ADD COLUMN rolled_back_column TEXT;
		DROP TABLE {base}.renamed_before;
		ROLLBACK`)
}

func (schema SchemaRecovery) BeginWalChanges(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	schema.execute(t, conn, `BEGIN;
		CREATE TABLE {added}.wal_records (id BIGSERIAL PRIMARY KEY, payload TEXT NOT NULL);
		INSERT INTO {added}.wal_records (payload) VALUES ('only-in-wal');
		ALTER TABLE {base}.accounts RENAME COLUMN label TO description;
		ALTER TABLE {base}.accounts ADD COLUMN committed_flag BOOLEAN NOT NULL DEFAULT TRUE;
		ALTER TABLE {base}.commit_rename_before RENAME TO commit_rename_after;
		DROP TABLE {base}.commit_drop`)
}

func (schema SchemaRecovery) ApplyLaterChanges(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	schema.execute(t, conn, `BEGIN;
		ALTER TABLE {added}.wal_records ADD COLUMN late_column TEXT DEFAULT 'after-target';
		CREATE TABLE {added}.late_table (payload TEXT);
		DROP TABLE {base}.late_drop;
		COMMIT`)
}

func (schema SchemaRecovery) RequireStage(t *testing.T, conn *pgx.Conn, stage SchemaRecoveryStage) {
	t.Helper()

	require.True(t, schema.hasRelation(t, conn, "renamed_before"))
	require.False(t, schema.hasRelation(t, conn, "rename_before"))
	require.False(t, schema.hasRelation(t, conn, "drop_before"))
	require.Equal(t, stage == SchemaBeforeWal, schema.hasRelation(t, conn, "commit_rename_before"))
	require.Equal(t, stage != SchemaBeforeWal, schema.hasRelation(t, conn, "commit_rename_after"))
	require.Equal(t, stage == SchemaBeforeWal, schema.hasRelation(t, conn, "commit_drop"))
	require.Equal(t, stage != SchemaLatest, schema.hasRelation(t, conn, "late_drop"))

	definitions := schema.GetState(t, conn)
	require.Contains(t, definitions.Definitions, "column|"+schema.Namespace+".accounts|amount|text|true||")
	require.NotContains(t, definitions.Definitions, "relation|"+schema.AddedNamespace+".rolled_back|r|p")
	require.Contains(t, definitions.Definitions, "column|"+schema.AddedNamespace+".orders|id|bigint|true||a")
	for _, definition := range definitions.Definitions {
		require.NotContains(t, definition, "rolled_back_column")
	}

	column := "label"
	if stage != SchemaBeforeWal {
		column = "description"
	}
	require.Contains(t, definitions.Definitions,
		"column|"+schema.Namespace+".accounts|"+column+"|text|true|'ready'::text|")

	rows, err := conn.Query(t.Context(), schema.sql(`SELECT amount FROM {base}.accounts ORDER BY name`))
	require.NoError(t, err)

	amounts, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)
	require.Equal(t, []string{"units:13", "units:7", "units:17"}, amounts)
}

func (schema SchemaRecovery) RequireWrites(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	schema.execute(t, conn, `INSERT INTO {added}.orders (name, amount) VALUES ('restored-valid', 23)`)

	_, err := conn.Exec(
		t.Context(),
		schema.sql(`INSERT INTO {added}.orders (name, amount) VALUES ('restored-invalid', -1)`),
	)
	var constraintError *pgconn.PgError
	require.ErrorAs(t, err, &constraintError)

	require.Equal(t, "23514", constraintError.Code)
	require.Equal(t, "positive_amount", constraintError.ConstraintName)
}

func (schema SchemaRecovery) RequireObservationSensitivity(t *testing.T, conn *pgx.Conn, stage SchemaRecoveryStage) {
	t.Helper()

	definitions := schema.GetState(t, conn).Definitions
	column := "label"
	if stage != SchemaBeforeWal {
		column = "description"
	}
	schema.execute(t, conn, `ALTER TABLE {base}.accounts DROP COLUMN `+pgx.Identifier{column}.Sanitize())
	changedDefinitions := schema.GetState(t, conn).Definitions
	columnDefinition := "column|" + schema.Namespace + ".accounts|" + column + "|text|true|'ready'::text|"
	require.Contains(t, definitions, columnDefinition)
	require.NotContains(t, changedDefinitions, columnDefinition)
	require.NotEqual(t, definitions, changedDefinitions)
}

func (schema SchemaRecovery) hasRelation(t *testing.T, conn *pgx.Conn, name string) bool {
	t.Helper()

	var hasRelation bool
	err := conn.QueryRow(t.Context(), `SELECT to_regclass($1) IS NOT NULL`,
		pgx.Identifier{schema.Namespace, name}.Sanitize()).Scan(&hasRelation)
	require.NoError(t, err)

	return hasRelation
}
