package sftp

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/modules/storages/internal/providers"
	"databasus-backend/internal/modules/storages/internal/providers/providertest"
)

func TestDeleteFile(t *testing.T) {
	// A non-empty directory is the one refusal the provider's own API can set up.
	t.Run("returns the error when the server refuses the removal", func(t *testing.T) {
		testedServer := startSftpServer(t)
		directory := "refused-" + uuid.New().String()

		client, sshConn, err := testedServer.connectWithContext(t.Context(), 30*time.Second)
		require.NoError(t, err)

		require.NoError(t, client.Mkdir(testedServer.getFilePath(directory)))

		t.Cleanup(func() {
			_ = client.Close()
			_ = sshConn.Close()
		})

		require.NoError(t, testedServer.provider.SaveFile(
			t.Context(), testedServer.access(directory+"/artifact"), strings.NewReader("payload"),
		))

		err = testedServer.provider.DeleteFile(t.Context(), testedServer.access(directory))

		assert.Error(t, err, "a removal the server refuses must not count as a completed deletion")
	})
}

func startSftpServer(t *testing.T) *sftpServer {
	t.Helper()

	endpoint := containers.StartSftp(t)

	return &sftpServer{
		provider: New(
			Deps{Repositories: Repositories{settings: nil}, FieldEncryptor: providertest.NewFieldEncryptor(t)},
		),
		settings: Settings{
			AuthMethods:         []AuthMethod{AuthMethodPassword},
			Host:                endpoint.Host,
			Port:                endpoint.Port,
			Username:            containers.SftpUsername,
			Password:            containers.SftpPassword,
			IsSkipHostKeyVerify: true,
			Path:                containers.SftpUploadDir,
		},
		storageID: uuid.New(),
	}
}

func (s *sftpServer) access(fileName string) providers.FileAccess {
	return providers.FileAccess{
		Location: providers.Location{StorageID: s.storageID, Settings: s.settings},
		FileName: fileName,
		Logger:   providertest.NewDiscardLogger(),
	}
}
