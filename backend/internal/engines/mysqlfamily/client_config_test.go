package mysqlfamily

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/mysqlfamily/mysqlfamilytest"
)

func TestWriteClientConfigFile(t *testing.T) {
	t.Parallel()

	t.Run("refuses a line break that would add a client option", func(t *testing.T) {
		t.Parallel()

		for _, testCase := range []struct {
			name       string
			connection func(resultPath string) ClientConnection
		}{
			{
				name: "user name",
				connection: func(resultPath string) ClientConnection {
					return newClosedPortClientConnection(t, "x\nresult-file="+resultPath, "stub-secret-password")
				},
			},
			{
				name: "password",
				connection: func(resultPath string) ClientConnection {
					return newClosedPortClientConnection(t, "backup_user", "p\nresult-file="+resultPath)
				},
			},
		} {
			t.Run("in the "+testCase.name, func(t *testing.T) {
				t.Parallel()
				mysqldump := getShippedDumpTools(t)[0]
				resultPath := filepath.Join(t.TempDir(), "result.sql")

				configFile, err := writeClientConfigFile(testCase.connection(resultPath))
				if err == nil {
					t.Cleanup(configFile.remove)
					mysqlfamilytest.RunClientTool(t, mysqldump.path, []string{
						configFile.getDefaultsFileArg(), "--no-data", "--", "orders",
					})
				}

				require.ErrorIs(t, err, ErrControlCharacterInClientConfig)
				assert.NoFileExists(t, resultPath)
				assert.NoFileExists(t, resultPath+`"`)
			})
		}
	})
}

func newClosedPortClientConnection(t *testing.T, username, password string) ClientConnection {
	t.Helper()

	return ClientConnection{
		Host:      "127.0.0.1",
		Port:      mysqlfamilytest.GetClosedPort(t),
		Username:  username,
		Password:  password,
		SSLOption: "ssl-mode=DISABLED",
	}
}
