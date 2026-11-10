package nas

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/background"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/modules/storages/internal/providers"
	"databasus-backend/internal/modules/storages/internal/providers/providertest"
)

func TestDeleteFile(t *testing.T) {
	// A non-empty directory is the one refusal the provider's own API can set up.
	t.Run("returns the error when the server refuses the removal", func(t *testing.T) {
		testedShare := startNasShare(t)
		directory := "refused-" + uuid.New().String()

		session, err := testedShare.dialSession(t.Context())
		require.NoError(t, err)
		t.Cleanup(func() { closeSession(session) })

		require.NoError(t, session.getShare(t.Context()).MkdirAll(testedShare.getFilePath(directory), 0o755))

		require.NoError(t, testedShare.provider.SaveFile(
			t.Context(), testedShare.access(directory+"/artifact"), strings.NewReader("payload"),
		))

		err = testedShare.provider.DeleteFile(t.Context(), testedShare.access(directory))

		assert.Error(t, err, "a removal the server refuses must not count as a completed deletion")
	})

	// The configured path is a directory that was never created, so this also covers a
	// removal whose parent is missing.
	t.Run("succeeds when the file is absent", func(t *testing.T) {
		testedShare := startNasShare(t)

		err := testedShare.provider.DeleteFile(t.Context(), testedShare.access("never-written-"+uuid.New().String()))

		assert.NoError(t, err)
	})
}

func startNasShare(t *testing.T) *nasShare {
	t.Helper()

	endpoint := containers.StartSamba(t)
	provider := New(Deps{
		Repositories:   Repositories{settings: nil},
		FieldEncryptor: providertest.NewFieldEncryptor(t),
		ChunkWrites:    background.New(background.Deps{Logger: providertest.NewDiscardLogger()}),
	})

	return &nasShare{
		provider: provider,
		settings: Settings{
			Host:     endpoint.Host,
			Port:     endpoint.Port,
			Share:    containers.SambaShare,
			Username: containers.SambaUsername,
			Password: containers.SambaPassword,
			Path:     "cleanup-" + uuid.New().String(),
		},
		storageID: uuid.New(),
	}
}

func (s *nasShare) access(fileName string) providers.FileAccess {
	return providers.FileAccess{
		StorageID: s.storageID, Settings: s.settings,
		FileName: fileName,
		Logger:   providertest.NewDiscardLogger(),
	}
}
