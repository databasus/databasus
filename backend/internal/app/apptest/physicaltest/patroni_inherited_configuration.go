package physicaltest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func (source patroniSource) prepareInheritedConfiguration(t *testing.T, spec PatroniIndependentSpec) {
	t.Helper()

	node := source.Fixture.GetNode(source.Fixture.GetIdentity(t).Node)
	configurationFile := "/var/lib/postgresql/patroni/postgresql.auto.conf"
	configuration := "\nrecovery_target_xid '1'\n"
	if !spec.ShouldUseExternalConfig && spec.Mode != "backup" {
		configuration += "hba_file '/source/pg_hba.conf'\nident_file '/source/pg_ident.conf'\n"
	}

	if spec.Mode == "latest" {
		files := map[string]string{
			"restore options/source.settings": "include '/var/lib/postgresql/patroni/restore options/entry.settings'\ndata_directory '/var/lib/postgresql/patroni'\n",
			"restore options/entry.settings":  "include_if_exists 'missing.settings'\ninclude_if_exists '/var/lib/postgresql/patroni/missing.settings'\ninclude_dir '/var/lib/postgresql/patroni/recovery.d'\n",
			"recovery.d/10-target.conf":       "include '../restore options/nested.settings'\n",
			"restore options/nested.settings": "include '/var/lib/postgresql/patroni/restore options/target.settings'\n",
			"restore options/target.settings": configuration,
		}
		for path, body := range files {
			code, _, err := node.Container.Exec(
				t.Context(),
				[]string{
					"sh",
					"-c",
					`mkdir -p "$(dirname "$1")"; printf '%s' "$2" > "$1"; chown postgres:postgres "$1"`,
					"configuration",
					"/var/lib/postgresql/patroni/" + path,
					body,
				},
			)
			require.NoError(t, err)
			require.Zero(t, code)
		}

		if !spec.ShouldUseExternalConfig {
			configurationFile = "/var/lib/postgresql/patroni/postgresql.conf"
		}

		configuration = "\ninclude 'restore options/source.settings'\n"
		code, _, err := node.Container.Exec(
			t.Context(),
			[]string{
				"sed", "-i", "/^[[:space:]]*data_directory[[:space:]=]/d",
				configurationFile, "/var/lib/postgresql/patroni/postgresql.auto.conf",
			},
		)
		require.NoError(t, err)
		require.Zero(t, code)
	}

	code, _, err := node.Container.Exec(
		t.Context(),
		[]string{
			"sh",
			"-c",
			`printf '%s' "$1" >> "$2"`,
			"configuration",
			configuration,
			configurationFile,
		},
	)
	require.NoError(t, err)
	require.Zero(t, code)
}
