package providers_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/storages/internal/providers/providertest"
)

func TestSaveFileStreaming(t *testing.T) {
	t.Parallel()

	t.Run("reads ahead when an SFTP write acknowledgement is withheld", func(t *testing.T) {
		t.Parallel()
		container := containers.StartSftpContainer(t)
		forwarder := providertest.StartCountingForwarder(t, providertest.GetAddress(container.Endpoint))
		container.Endpoint = containers.Endpoint{Host: forwarder.Host, Port: forwarder.Port}
		storage := buildSftpStorage(t, container)
		content := generateRandomContent(t, segmentSizedFileBytes)
		sourceCtx, cancel := context.WithCancel(t.Context())
		source := providertest.NewPausedReader(sourceCtx, content)
		fileName := "streamed-" + uuid.NewString()
		storage.savedFileNames = append(storage.savedFileNames, fileName)
		writeErrors := make(chan error, 1)
		var writes sync.WaitGroup
		defer func() { cancel(); source.Resume(); writes.Wait() }()
		writes.Go(func() {
			writeErrors <- storage.provider.SaveFile(sourceCtx, storage.access(fileName), source)
		})

		source.WaitUntilRead(t)
		gate := forwarder.PauseResponses(t)
		defer gate.Release()
		source.Resume()
		gate.WaitUntilBlocked(t)

		testwait.Until(
			t,
			"SFTP to read another block before receiving the first acknowledgement",
			func(context.Context) error {
				if source.GetReadCount() < 2 {
					return fmt.Errorf("only %d source blocks have been read", source.GetReadCount())
				}

				return nil
			},
		)

		gate.Release()
		testwait.Until(t, "the streamed SFTP write to finish", func(context.Context) error {
			select {
			case writeError := <-writeErrors:
				require.NoError(t, writeError)
				return nil
			default:
				return fmt.Errorf("the SFTP write is still running")
			}
		})

		require.Equal(t, content, storage.read(t, fileName))
	})
}
