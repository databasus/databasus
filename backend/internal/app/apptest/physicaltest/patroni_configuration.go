package physicaltest

import (
	"testing"

	containers "databasus-backend/internal/lib/testcontainers"
)

func prepareIndependentConfiguration(t *testing.T, target containers.RestoreTarget, image string) {
	t.Helper()

	for name, body := range map[string]string{
		"postgresql.conf": "listen_addresses = '*'\n",
		"pg_hba.conf":     "local all all trust\nhost all all 0.0.0.0/0 scram-sha-256\n",
		"pg_ident.conf":   "",
	} {
		destination := restoredClusterDir(image) + "/" + name
		target.Exec(t, "rm", "-f", destination)
		target.Exec(t, "sh", "-c", `printf '%s' "$1" > "$2"`, "configuration", body, destination)
		target.Exec(t, "chmod", "0600", destination)
		target.Exec(t, "chown", "postgres:postgres", destination)
	}
}
