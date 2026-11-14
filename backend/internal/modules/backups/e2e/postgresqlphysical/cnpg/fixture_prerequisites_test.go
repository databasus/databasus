package cnpg_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

func TestCNPGFixturePrerequisites(t *testing.T) {
	t.Parallel()

	for _, prerequisite := range []struct {
		mode    string
		failure error
	}{
		{mode: childDockerUnavailable, failure: containers.ErrDockerUnavailable},
		{mode: childUnprivileged, failure: containers.ErrPrivilegedContainerRequired},
		{mode: childUnreachableAddress, failure: containers.ErrContainerAddressUnreachable},
		{mode: childMissingImage, failure: containers.ErrImageUnavailable},
	} {
		t.Run("fails naming the prerequisite with "+prerequisite.mode, func(t *testing.T) {
			t.Parallel()

			spec := cnpgChildSpec{Mode: prerequisite.mode}
			if prerequisite.mode == childDockerUnavailable {
				spec.DockerHost = "unix://" + filepath.Join(t.TempDir(), "absent.sock")
			}

			child := startCNPGChild(t, spec)

			output, err := child.wait(t)

			require.Error(t, err, "%s", output)
			require.Contains(t, output, prerequisite.failure.Error())
			require.Contains(t, output, "--- FAIL: TestCNPGFixtureChild")
			require.NotContains(t, output, "--- SKIP")
			require.NotContains(t, output, "--- PASS")
			if prerequisite.mode != childDockerUnavailable {
				child.requireRemoved(t)
			}
		})
	}
}
