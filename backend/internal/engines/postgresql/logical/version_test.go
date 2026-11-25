package logical

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
)

func TestParseVersion(t *testing.T) {
	t.Parallel()

	for _, versionCase := range []struct {
		name            string
		rawVersion      string
		expectedVersion pgshared.Version
	}{
		{"accepts PostgreSQL 12", "PostgreSQL 12.22 on x86_64-pc-linux-gnu", pgshared.Version12},
		{"accepts PostgreSQL 16 without a minor version", "PostgreSQL 16 maintained by a vendor", pgshared.Version16},
		{"accepts PostgreSQL 18", "PostgreSQL 18.7 (Debian 18.7-1.pgdg12+2) on aarch64", pgshared.Version18},
		{"accepts a PostgreSQL 19 minor newer than the bundled client", "PostgreSQL 19.3 on x86_64-pc-linux-gnu", pgshared.Version19},
	} {
		t.Run(versionCase.name, func(t *testing.T) {
			t.Parallel()

			parsedVersion, err := parseVersion(versionCase.rawVersion)

			require.NoError(t, err)
			assert.Equal(t, versionCase.expectedVersion, parsedVersion)
			assert.Equal(
				t,
				filepath.Join(
					"tools",
					"postgresql",
					"postgresql-"+string(versionCase.expectedVersion),
					"bin",
					"pg_dump",
				),
				pgshared.ClientTools{Root: "tools"}.GetExecutable(parsedVersion, pgshared.ExecutablePgDump),
			)
		})
	}

	for _, unsupportedMajor := range []string{"11", "20"} {
		t.Run("names the supported range when the server runs PostgreSQL "+unsupportedMajor, func(t *testing.T) {
			t.Parallel()

			_, err := parseVersion("PostgreSQL " + unsupportedMajor + ".1 on x86_64-pc-linux-gnu")

			require.ErrorIs(t, err, ErrUnsupportedVersion)
			require.ErrorContains(t, err, "found PostgreSQL "+unsupportedMajor)
			assert.ErrorContains(t, err, "supported versions are "+pgshared.GetSupportedVersionRange())
		})
	}

	t.Run("reports the text when the version cannot be read", func(t *testing.T) {
		t.Parallel()

		_, err := parseVersion("CockroachDB CCL v23.1")

		require.ErrorIs(t, err, ErrUnreadableVersion)
		assert.ErrorContains(t, err, "CockroachDB CCL v23.1")
	})
}
