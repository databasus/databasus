package physicaltest

import (
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

// CloudNativePG keeps its certificates, socket and log directories outside the data directory. The
// recovery script must name them, and the harness then overrides each in postgresql.auto.conf as its
// instructions tell a user to, keeping the copied configuration and access rules.
func prepareCNPGRestoredCluster(image string) func(*testing.T, containers.RestoreTarget, []byte) {
	return func(t *testing.T, target containers.RestoreTarget, output []byte) {
		t.Helper()

		requireExternalConfigurationReported(t, output)
		target.Exec(
			t,
			"sh",
			"-c",
			`printf '%s' "$1" >> "$2"`,
			"configuration",
			"ssl = off\nunix_socket_directories = '/var/run/postgresql'\nlog_directory = 'log'\n",
			restoredClusterDir(image)+"/postgresql.auto.conf",
		)
	}
}

func requireExternalConfigurationReported(t *testing.T, output []byte) {
	t.Helper()

	report := string(output)
	require.Contains(t, report, "ACTION REQUIRED: the restored configuration points outside the data directory")
	for _, setting := range []string{
		"ssl_cert_file = '/controller/certificates/server.crt'",
		"ssl_key_file = '/controller/certificates/server.key'",
		"ssl_ca_file = '/controller/certificates/client-ca.crt'",
		"unix_socket_directories = '/controller/run'",
		"log_directory = '/controller/log'",
	} {
		require.Contains(t, report, setting)
	}
}
