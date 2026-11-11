package physicaltest

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type copiedAuthentication struct {
	HBAHash   string
	IdentHash string
}

func (source patroniSource) prepareCopiedAuthentication(t *testing.T, shouldHaveDefaultAuthentication bool) {
	t.Helper()

	connection := source.connect(t)
	var hbaPath string
	var identPath string
	require.NoError(
		t,
		connection.QueryRow(t.Context(), "SELECT current_setting('hba_file'), current_setting('ident_file')").
			Scan(&hbaPath, &identPath),
	)

	node := source.Fixture.GetNode(source.Fixture.GetIdentity(t).Node)
	configuration := "listen_addresses = '*'\ndata_directory '/var/lib/postgresql/patroni'\n" +
		"hba_file '/var/lib/postgresql/patroni/auth/hba.rules'\n" +
		"ident_file '/var/lib/postgresql/patroni/auth/ident.rules'\n"
	code, _, err := node.Container.Exec(t.Context(), []string{
		"sh", "-c", `mkdir -p /var/lib/postgresql/patroni/auth
cp "$1" /var/lib/postgresql/patroni/auth/hba.rules
cp "$2" /var/lib/postgresql/patroni/auth/ident.rules
printf '%s' "$3" > /var/lib/postgresql/patroni/postgresql.conf
chown -R postgres:postgres /var/lib/postgresql/patroni/auth /var/lib/postgresql/patroni/postgresql.conf
test ! -e /var/lib/postgresql/patroni/pg_hba.conf
test ! -e /var/lib/postgresql/patroni/pg_ident.conf`,
		"configuration", hbaPath, identPath, configuration,
	})
	require.NoError(t, err)
	require.Zero(t, code)

	if shouldHaveDefaultAuthentication {
		code, _, err := node.Container.Exec(t.Context(), []string{
			"sh",
			"-c",
			`printf 'local all all reject\nhost all all 0.0.0.0/0 reject\nhost all all ::/0 reject\n' > /var/lib/postgresql/patroni/pg_hba.conf
printf 'unrelated_root postgres unrelated_user\n' > /var/lib/postgresql/patroni/pg_ident.conf
chown postgres:postgres /var/lib/postgresql/patroni/pg_hba.conf /var/lib/postgresql/patroni/pg_ident.conf
chmod 600 /var/lib/postgresql/patroni/pg_hba.conf /var/lib/postgresql/patroni/pg_ident.conf
! cmp -s /var/lib/postgresql/patroni/pg_hba.conf /var/lib/postgresql/patroni/auth/hba.rules &&
! cmp -s /var/lib/postgresql/patroni/pg_ident.conf /var/lib/postgresql/patroni/auth/ident.rules`,
		})
		require.NoError(t, err)
		require.Zero(t, code)
	}
}

func requireCopiedAuthentication(t *testing.T, connection *pgx.Conn, authentication copiedAuthentication) {
	t.Helper()

	var hbaHash string
	var identHash string
	require.NoError(
		t,
		connection.QueryRow(t.Context(), "SELECT md5(pg_read_file(current_setting('hba_file'))), md5(pg_read_file(current_setting('ident_file')))").
			Scan(&hbaHash, &identHash),
	)

	require.Equal(t, authentication.HBAHash, hbaHash)
	require.Equal(t, authentication.IdentHash, identHash)
}

func (source patroniSource) requireCopiedAuthenticationOnHost(
	t *testing.T,
	bundle string,
	authentication copiedAuthentication,
) {
	t.Helper()

	directory := reconstructOnHost(t, restoreSpec{App: source.Team.App, Image: source.Image, BundlePath: bundle})
	endpoint := bootBoundCluster(t, clusterDirectory{Image: source.Image, OutputDirectory: directory})
	connection := openConnAt(t, endpoint.Host, endpoint.Port)

	requireCopiedAuthentication(t, connection, authentication)

	var isInRecovery bool
	require.NoError(t, connection.QueryRow(t.Context(), "SELECT pg_is_in_recovery()").Scan(&isInRecovery))
	require.False(t, isInRecovery)
}
