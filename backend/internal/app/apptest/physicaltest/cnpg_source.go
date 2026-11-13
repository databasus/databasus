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

// CNPGManagerSpec selects the CloudNativePG image of the source and the official image of the same
// major that boots recovered clusters outside Kubernetes.
type CNPGManagerSpec struct {
	Pool         *containers.CNPGPool
	ClusterImage string
	BootImage    string
}

// NewCNPGManager starts sources whose primary CloudNativePG selects behind a Kubernetes Service, reached
// with the declared backup role over verified TLS.
func NewCNPGManager(spec CNPGManagerSpec) FailoverManager {
	return FailoverManager{startSource: func(t *testing.T, sourceSpec failoverSourceSpec) failoverSource {
		t.Helper()

		source := newCNPGSource(t, cnpgSourceSpec{
			Manager:         spec,
			ShouldIntercept: sourceSpec.ShouldIntercept,
			SlotMode:        containers.CNPGSlotSynchronized,
			LogSink:         sourceSpec.LogSink,
		})

		if sourceSpec.ShouldKeepSlotOnEveryNode {
			source.Cluster = cnpgRetainedSlotCluster{FailoverCluster: source.Cluster, source: source}
		}

		return source.failoverSource
	}}
}

type cnpgSource struct {
	failoverSource
	Fixture *containers.CNPGCluster
}

type cnpgSourceSpec struct {
	Manager         CNPGManagerSpec
	ShouldIntercept bool
	SlotMode        containers.CNPGSlotMode
	LogSink         slog.Handler
}

func newCNPGSource(t *testing.T, spec cnpgSourceSpec) cnpgSource {
	t.Helper()

	fixture := spec.Manager.Pool.StartCluster(t, containers.CNPGClusterSpec{Image: spec.Manager.ClusterImage})
	endpoint := fixture.Endpoint
	var proxy *containers.PostgresProxy
	if spec.ShouldIntercept {
		proxy = containers.StartPostgresProxy(
			t,
			endpoint,
			containers.WithProxyTLS(fixture.Authority, fixture.ServerCertificate),
		)
		endpoint = proxy.Endpoint
	}

	var options []apptest.Option
	if spec.LogSink != nil {
		options = append(options, apptest.WithLogSink(spec.LogSink))
	}

	authority := fixture.Authority.CertificatePEM
	team := newTeam(t, options...)
	team.createDatabase(t, physicalDatabaseSpec{
		Endpoint:   endpoint,
		BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
		Tunnel:     enginetest.NewDisabledSSHTunnelSettings(),
		Access: sourceAccess{
			Username:        containers.CNPGBackupRole,
			Password:        containers.PostgresPassword,
			SslMode:         apiclient.PostgresqlSslModeVerifyFull,
			RootCertificate: &authority,
		},
	})

	cluster := cnpgCluster{fixture: fixture}
	connection := cluster.Connect(t)
	_, err := connection.Exec(t.Context(), "CREATE TABLE backup_permission_probe(id integer)")
	require.NoError(t, err)

	team.requireRestrictedRoleAttributes(t, connection)
	team.requireRestrictedPromotionRole(t)

	slotName := team.getSlotName(t)
	fixture.ApplySlotMode(t, spec.SlotMode, slotName)

	schema := NewSchemaRecovery(t, connection)
	sequences := NewSequenceRecovery(t, connection, schema)

	return cnpgSource{
		Team:                   team,
		Cluster:                cluster,
		Image:                  spec.Manager.BootImage,
		Schema:                 schema,
		Sequences:              sequences,
		Proxy:                  proxy,
		SlotName:               slotName,
		prepareRestoredCluster: prepareCNPGRestoredCluster(spec.Manager.BootImage),
		stopSource:             fixture.Hibernate,
		Fixture:                fixture,
	}
}
