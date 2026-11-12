package physicaltest

import (
	"log/slog"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

type patroniSource struct {
	Team      Team
	Fixture   *containers.Patroni
	Image     string
	Schema    SchemaRecovery
	Sequences SequenceRecovery
	Proxy     *containers.PostgresProxy
	SlotName  string
}

type patroniSourceSpec struct {
	Image                    string
	ShouldIntercept          bool
	ShouldIgnoreSlot         bool
	ShouldUseExternalConfig  bool
	ShouldKeepSlotOnAllNodes bool
	LogSink                  slog.Handler
}

func newPatroniSource(t *testing.T, spec patroniSourceSpec) patroniSource {
	t.Helper()

	fixture := containers.StartPatroni(
		t,
		containers.PatroniSpec{Image: spec.Image, ShouldUseExternalConfig: spec.ShouldUseExternalConfig},
	)
	endpoint := fixture.Endpoint
	var proxy *containers.PostgresProxy
	if spec.ShouldIntercept {
		proxy = containers.StartPostgresProxy(t, endpoint)
		endpoint = proxy.Endpoint
	}

	var options []apptest.Option
	if spec.LogSink != nil {
		options = append(options, apptest.WithLogSink(spec.LogSink))
	}

	team := newTeam(t, options...)
	team.createDatabase(t, physicalDatabaseSpec{
		Endpoint: endpoint, BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
		Tunnel: enginetest.NewDisabledSSHTunnelSettings(),
	})
	team.useReplicationOnlyUser(t)

	connection := openConnAt(t, fixture.Endpoint.Host, fixture.Endpoint.Port)
	_, err := connection.Exec(t.Context(), "CREATE TABLE backup_permission_probe(id integer)")
	require.NoError(t, err)
	team.requireRestrictedPromotionRole(t)

	var slotName string
	team.App.QuerySQL(
		t,
		&slotName,
		"SELECT replication_slot_name FROM databases_postgresql_physical WHERE database_id=?",
		team.Database.Id,
	)
	require.NotEmpty(t, slotName)

	if spec.ShouldIgnoreSlot {
		fixture.IgnoreSlot(t, slotName)
	}

	if spec.ShouldKeepSlotOnAllNodes {
		fixture.KeepSlotOnAllNodes(t, slotName)
	}

	schema := NewSchemaRecovery(t, connection)
	sequences := NewSequenceRecovery(t, connection, schema)

	return patroniSource{
		Team:      team,
		Fixture:   fixture,
		Image:     spec.Image,
		Schema:    schema,
		Sequences: sequences,
		Proxy:     proxy,
		SlotName:  slotName,
	}
}

func (source patroniSource) connect(t *testing.T) *pgx.Conn {
	t.Helper()
	return openConnAt(t, source.Fixture.Endpoint.Host, source.Fixture.Endpoint.Port)
}

func (source patroniSource) applySchemaChanges(t *testing.T, connection *pgx.Conn) {
	t.Helper()

	source.Schema.CreatePostFullObjects(t, connection)
	source.Sequences.PrepareCalls(t, connection)
	source.Schema.ApplyBetweenIncrementals(t, connection)
	source.Schema.BeginWalChanges(t, connection)
	source.Sequences.ApplyWalChanges(t, connection)
	source.Schema.execute(t, connection, "COMMIT")
	source.Schema.RequireStage(t, connection, SchemaAfterWal)
	source.Sequences.RequireDefinitions(t, connection, SchemaAfterWal)
}

func (source patroniSource) requireRestoredSchema(t *testing.T, connection *pgx.Conn) {
	t.Helper()

	source.Schema.RequireStage(t, connection, SchemaAfterWal)
	source.Sequences.RequireDefinitions(t, connection, SchemaAfterWal)
	source.Sequences.RequireAllocations(t, connection, SchemaAfterWal)
	source.Schema.RequireWrites(t, connection)
}
