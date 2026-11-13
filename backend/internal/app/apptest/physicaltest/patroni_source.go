package physicaltest

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

// NewPatroniManager starts sources whose primary Patroni selects behind a plaintext HAProxy address.
func NewPatroniManager(image string) FailoverManager {
	return FailoverManager{startSource: func(t *testing.T, spec failoverSourceSpec) failoverSource {
		t.Helper()

		return newPatroniSource(t, patroniSourceSpec{
			Image:                    image,
			ShouldIntercept:          spec.ShouldIntercept,
			ShouldIgnoreSlot:         true,
			ShouldKeepSlotOnAllNodes: spec.ShouldKeepSlotOnEveryNode,
			LogSink:                  spec.LogSink,
		}).failoverSource
	}}
}

type patroniSource struct {
	failoverSource
	Fixture *containers.Patroni
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

	cluster := patroniCluster{fixture: fixture}
	connection := cluster.Connect(t)
	_, err := connection.Exec(t.Context(), "CREATE TABLE backup_permission_probe(id integer)")
	require.NoError(t, err)
	team.requireRestrictedPromotionRole(t)

	slotName := team.getSlotName(t)
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
		Cluster:   cluster,
		Image:     spec.Image,
		Schema:    schema,
		Sequences: sequences,
		Proxy:     proxy,
		SlotName:  slotName,
		Fixture:   fixture,
	}
}
