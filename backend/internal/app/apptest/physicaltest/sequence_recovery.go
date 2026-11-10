package physicaltest

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type SequenceRecovery struct {
	Schema SchemaRecovery
}

type generatedIDTable struct {
	Name            string
	InsertStatement string
	ForwardFloor    int64
}

func NewSequenceRecovery(t *testing.T, conn *pgx.Conn, schema SchemaRecovery) SequenceRecovery {
	t.Helper()

	schema.execute(t, conn, `CREATE SEQUENCE {base}.position_seq AS BIGINT START WITH 401 INCREMENT BY 3
		MINVALUE 1 MAXVALUE 10000000 CACHE 1 NO CYCLE;
		CREATE TABLE {base}.position_ids (id BIGINT DEFAULT nextval('{base}.position_seq'::regclass) PRIMARY KEY,
			label TEXT NOT NULL);
		INSERT INTO {base}.position_ids (label) VALUES ('baseline')`)

	return SequenceRecovery{Schema: schema}
}

func (sequences SequenceRecovery) PrepareCalls(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	schema := sequences.Schema
	var calledValue int64
	err := conn.QueryRow(t.Context(), schema.sql(`SELECT nextval('{base}.position_seq'::regclass)`)).Scan(&calledValue)
	require.NoError(t, err)
	require.EqualValues(t, 404, calledValue)

	schema.execute(t, conn, `BEGIN;
		INSERT INTO {base}.position_ids (label) VALUES ('rolled-back-nextval');
		ROLLBACK`)
	err = conn.QueryRow(t.Context(), schema.sql(`SELECT nextval('{base}.position_seq'::regclass)`)).Scan(&calledValue)
	require.NoError(t, err)
	require.EqualValues(t, 410, calledValue)

	schema.execute(t, conn, `BEGIN;
		SELECT setval('{base}.position_seq'::regclass, 10001, TRUE);
		INSERT INTO {base}.position_ids (label) VALUES ('rolled-back-setval');
		ROLLBACK`)
	err = conn.QueryRow(t.Context(), schema.sql(`SELECT nextval('{base}.position_seq'::regclass)`)).Scan(&calledValue)
	require.NoError(t, err)
	require.EqualValues(t, 10007, calledValue)

	schema.execute(t, conn, `INSERT INTO {base}.position_ids (label) VALUES ('after-rollback')`)
	schema.execute(t, conn, `BEGIN;
		ALTER SEQUENCE {base}.position_seq RESTART WITH 900001 INCREMENT BY 19 CACHE 8;
		ROLLBACK`)
	err = conn.QueryRow(t.Context(), schema.sql(`SELECT nextval('{base}.position_seq'::regclass)`)).Scan(&calledValue)
	require.NoError(t, err)
	require.EqualValues(t, 10013, calledValue)
}

func (sequences SequenceRecovery) ApplyWalChanges(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	sequences.Schema.execute(t, conn, `ALTER SEQUENCE {base}.position_seq INCREMENT BY 7 CACHE 32 START WITH 701;
		SELECT setval('{base}.position_seq'::regclass, 1000001, TRUE);
		SELECT setval('{added}.orders_id_seq'::regclass, 2000001, FALSE)`)
}

func (sequences SequenceRecovery) RequireDefinitions(t *testing.T, conn *pgx.Conn, stage SchemaRecoveryStage) {
	t.Helper()

	labels := getRecoveryText(t, conn, recoveryQuery{
		Statement: sequences.Schema.sql(`SELECT label FROM {base}.position_ids ORDER BY label`),
	})
	require.Equal(t, []string{"after-rollback", "baseline"}, labels)

	states := sequences.GetStates(t, conn)
	var position *SequenceState
	for index := range states {
		state := &states[index]
		require.False(t, state.IsCycle, state.Name)

		if state.Name == sequences.Schema.Namespace+".position_seq" {
			position = state
		}
	}

	require.NotNil(t, position)
	require.Equal(t, "bigint", position.DataType)
	require.EqualValues(t, 1, position.Minimum)
	require.EqualValues(t, 10000000, position.Maximum)
	require.Empty(t, position.Dependency)

	if stage == SchemaBeforeWal {
		require.EqualValues(t, 401, position.Start)
		require.EqualValues(t, 3, position.Increment)
		require.EqualValues(t, 1, position.Cache)
	} else {
		require.EqualValues(t, 701, position.Start)
		require.EqualValues(t, 7, position.Increment)
		require.EqualValues(t, 32, position.Cache)
	}

	var serialDependency, identityDependency string
	for _, state := range states {
		if state.Name == sequences.Schema.Namespace+".accounts_id_seq" {
			serialDependency = state.Dependency
			require.EqualValues(t, 1, state.Cache)
		}

		if state.Name == sequences.Schema.AddedNamespace+".orders_id_seq" {
			identityDependency = state.Dependency
			require.EqualValues(t, 32, state.Cache)
		}
	}

	require.Equal(t, sequences.Schema.Namespace+".accounts.id|a", serialDependency)
	require.Equal(t, sequences.Schema.AddedNamespace+".orders.id|i", identityDependency)
}

func (sequences SequenceRecovery) RequireAllocations(t *testing.T, conn *pgx.Conn, stage SchemaRecoveryStage) {
	t.Helper()

	var positionFloor, identityFloor int64
	if stage != SchemaBeforeWal {
		positionFloor = 1000002
		identityFloor = 2000001
	}

	sequenceTables := map[string]generatedIDTable{
		sequences.Schema.Namespace + ".accounts_id_seq": {
			Name: "{base}.accounts",
			InsertStatement: `INSERT INTO {base}.accounts (name, amount)
				VALUES ('restored-generated', 'units:29') RETURNING id`,
		},
		sequences.Schema.Namespace + ".position_seq": {
			Name: "{base}.position_ids", ForwardFloor: positionFloor,
			InsertStatement: `INSERT INTO {base}.position_ids (label) VALUES ('restored-generated') RETURNING id`,
		},
		sequences.Schema.AddedNamespace + ".orders_id_seq": {
			Name: "{added}.orders", ForwardFloor: identityFloor,
			InsertStatement: `INSERT INTO {added}.orders (name, amount) VALUES ('restored-generated', 29) RETURNING id`,
		},
		sequences.Schema.AddedNamespace + ".wal_records_id_seq": {
			Name:            "{added}.wal_records",
			InsertStatement: `INSERT INTO {added}.wal_records (payload) VALUES ('restored-generated') RETURNING id`,
		},
	}

	states := sequences.GetStates(t, conn)
	for _, state := range states {
		table, hasTable := sequenceTables[state.Name]
		require.True(t, hasTable, state.Name)

		var largestRestoredID int64
		err := conn.QueryRow(t.Context(), sequences.Schema.sql(`SELECT max(id) FROM `+table.Name)).
			Scan(&largestRestoredID)
		require.NoError(t, err, state.Name)

		var insertedID int64
		err = conn.QueryRow(t.Context(), sequences.Schema.sql(table.InsertStatement)).Scan(&insertedID)
		require.NoError(t, err, state.Name)

		require.Greater(t, insertedID, largestRestoredID, state.Name)
		require.GreaterOrEqual(t, insertedID, table.ForwardFloor, state.Name)

		var previousValue int64
		for allocation := range 3 {
			var allocatedValue int64
			err := conn.QueryRow(t.Context(), `SELECT nextval($1::regclass)`, state.Name).Scan(&allocatedValue)
			require.NoError(t, err, state.Name)

			require.Greater(t, allocatedValue, insertedID, state.Name)
			require.GreaterOrEqual(t, allocatedValue, table.ForwardFloor, state.Name)

			if allocation > 0 {
				require.Equal(t, state.Increment, allocatedValue-previousValue, state.Name)
			}

			previousValue = allocatedValue
		}
	}
}

func (sequences SequenceRecovery) RequireDistinctCaches(t *testing.T, connections []*pgx.Conn) {
	t.Helper()

	require.NotEmpty(t, connections)

	var largestStoredID int64
	err := connections[0].QueryRow(t.Context(), sequences.Schema.sql(`SELECT max(id) FROM {added}.orders`)).
		Scan(&largestStoredID)
	require.NoError(t, err)

	allocatedValues := make(map[int64]bool)
	for range 3 {
		for _, conn := range connections {
			var allocatedValue int64
			err := conn.QueryRow(t.Context(), sequences.Schema.sql(`SELECT nextval('{added}.orders_id_seq'::regclass)`)).
				Scan(&allocatedValue)
			require.NoError(t, err)

			require.Greater(t, allocatedValue, largestStoredID)
			require.False(t, allocatedValues[allocatedValue], "cached sessions must allocate distinct values")

			allocatedValues[allocatedValue] = true
		}
	}
}

func (sequences SequenceRecovery) RequireDefinitionSensitivity(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	definitions := sequences.GetStates(t, conn)
	sequences.Schema.execute(t, conn, `ALTER SEQUENCE {base}.position_seq CACHE 64`)
	changedDefinitions := sequences.GetStates(t, conn)
	require.NotEqual(t, definitions, changedDefinitions)
	for _, definition := range changedDefinitions {
		if definition.Name == sequences.Schema.Namespace+".position_seq" {
			require.EqualValues(t, 64, definition.Cache)
		}
	}
}
