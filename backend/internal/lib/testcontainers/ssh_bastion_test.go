package containers

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/config"
)

func TestMongodbBastionFailureDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("prints source and bastion logs before cleanup when a scenario fails", func(t *testing.T) {
		t.Parallel()

		variables, _, err := config.ReadVariables(config.LoadSpec{})
		require.NoError(t, err)

		if variables["DATABASUS_MONGODB_DIAGNOSTIC_PROBE"] == "1" {
			StartMongodbBehindSSHBastion(t, "mongo:7.0")
			t.Fatal("deliberate MongoDB diagnostic failure")
		}

		probe := exec.CommandContext(t.Context(), os.Args[0], "-test.run", "^TestMongodbBastionFailureDiagnostics$")
		probe.Env = append(probe.Environ(), "DATABASUS_MONGODB_DIAGNOSTIC_PROBE=1")

		output, err := probe.CombinedOutput()

		require.Error(t, err)
		assert.Contains(t, string(output), "deliberate MongoDB diagnostic failure")
		assert.Contains(t, string(output), "mongo:7.0 container output tail:")
		assert.Contains(t, string(output), "SSH bastion container output tail:")
		assert.NotContains(t, string(output), MongodbPassword)
		assert.NotContains(t, string(output), SSHBastionPassword)
	})
}
