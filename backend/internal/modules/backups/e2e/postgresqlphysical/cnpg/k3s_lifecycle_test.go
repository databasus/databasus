package cnpg_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

func TestCNPGFixtureLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("runs the pinned k3s and CloudNativePG operator after startup", func(t *testing.T) {
		t.Parallel()

		fixture := startK3s(t, containers.K3sSpec{})

		k3sVersion, err := fixture.GetK3sVersion(t.Context())
		require.NoError(t, err)
		require.Equal(t, "v1.36.5+k3s1", k3sVersion)

		operatorVersion, err := fixture.GetOperatorVersion(t.Context())
		require.NoError(t, err)
		require.Contains(t, operatorVersion, "Version:"+containers.CNPGOperatorVersion)
	})

	for _, image := range []string{containers.CNPGPostgres17Image, containers.CNPGPostgres18Image} {
		t.Run("serves the configured primary through the route with "+image, func(t *testing.T) {
			t.Parallel()

			fixture := startK3s(t, containers.K3sSpec{})
			authority, err := containers.NewTestCertificateAuthority()
			require.NoError(t, err)
			cluster := containers.StartCNPGCluster(
				t,
				containers.CNPGClusterSpec{K3s: fixture, Authority: authority, Image: image},
			)

			identity := cluster.GetIdentity(t)
			for _, node := range cluster.Nodes {
				settings := cluster.QueryNode(t, containers.NodeQuery{
					Node: node,
					SQL: "SELECT string_agg(name || '=' || setting, ',' ORDER BY name) FROM pg_settings " +
						"WHERE name IN ('data_checksums','fsync','full_page_writes','summarize_wal','synchronous_commit','wal_keep_size','ssl')",
				})
				require.Equal(
					t,
					"data_checksums=on,fsync=on,full_page_writes=on,ssl=on,summarize_wal=on,synchronous_commit=on,wal_keep_size=64",
					settings,
					node,
				)
			}

			backupRole := cluster.QueryNode(t, containers.NodeQuery{
				Node: identity.Node,
				SQL: "SELECT rolsuper, rolreplication, rolcreaterole, rolcreatedb, rolbypassrls, " +
					"pg_has_role(rolname, '" + containers.CNPGBackupRoleGroup + "', 'member'), " +
					"has_function_privilege(rolname, 'pg_switch_wal()', 'execute') " +
					"FROM pg_roles WHERE rolname='" + containers.CNPGBackupRole + "'",
			})
			require.Equal(t, "f|t|f|f|f|t|t", backupRole)

			connection := cluster.ConnectAsBackupRole(t)
			var isEncrypted bool
			require.NoError(
				t,
				connection.QueryRow(t.Context(), "SELECT ssl FROM pg_stat_ssl WHERE pid = pg_backend_pid()").
					Scan(&isEncrypted),
			)
			require.True(t, isEncrypted)
			require.Positive(t, identity.Timeline)
			require.NotEmpty(t, identity.SystemIdentifier)
		})
	}
}

func startK3s(t *testing.T, spec containers.K3sSpec) *containers.K3s {
	t.Helper()

	spec.Logf = t.Logf
	fixture, err := containers.StartK3s(t.Context(), spec)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, fixture.Terminate(context.WithoutCancel(t.Context()))) })

	return fixture
}
