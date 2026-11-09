package providers_test

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/storages/internal/providers/providertest"
)

func TestSaveFileConcurrency(t *testing.T) {
	t.Parallel()

	for _, target := range getProviderTargets(true) {
		t.Run("stores independent files when four writers overlap on "+target.name, func(t *testing.T) {
			t.Parallel()
			storage := target.build(t, target.startContainer(t))
			writeCtx, cancel := context.WithCancel(t.Context())
			var writes sync.WaitGroup
			defer func() { cancel(); writes.Wait() }()

			const writerCount = 4
			sources := make([]*providertest.PausedReader, writerCount)
			segmentNames := make([]string, writerCount)
			metadataNames := make([]string, writerCount)
			segmentContents := make([][]byte, writerCount)
			metadataContents := make([][]byte, writerCount)
			writeErrors := make(chan error, writerCount)
			for writerIndex := range writerCount {
				segmentNames[writerIndex] = "segment-" + uuid.NewString()
				metadataNames[writerIndex] = "metadata-" + uuid.NewString()
				segmentContents[writerIndex] = bytes.Repeat([]byte{byte(writerIndex + 1)}, segmentSizedFileBytes)
				metadataContents[writerIndex] = bytes.Repeat([]byte{byte(writerIndex + 11)}, metadataSizedFileBytes)
				sources[writerIndex] = providertest.NewPausedReader(writeCtx, segmentContents[writerIndex])
				storage.savedFileNames = append(
					storage.savedFileNames,
					segmentNames[writerIndex],
					metadataNames[writerIndex],
				)
			}

			for writerIndex := range writerCount {
				writes.Go(func() {
					if err := storage.provider.SaveFile(
						writeCtx,
						storage.access(segmentNames[writerIndex]),
						sources[writerIndex],
					); err != nil {
						writeErrors <- err
						return
					}

					writeErrors <- storage.provider.SaveFile(writeCtx, storage.access(metadataNames[writerIndex]),
						bytes.NewReader(metadataContents[writerIndex]))
				})
			}

			for _, source := range sources {
				source.WaitUntilRead(t)
			}

			for _, source := range sources {
				source.Resume()
			}

			completedWriters := 0
			testwait.Until(t, "all concurrent storage writes to finish", func(context.Context) error {
				for completedWriters < writerCount {
					select {
					case writeError := <-writeErrors:
						require.NoError(t, writeError)
						completedWriters++
					default:
						return fmt.Errorf("%d of %d writers completed", completedWriters, writerCount)
					}
				}

				return nil
			})

			for writerIndex := range writerCount {
				require.Equal(t, segmentContents[writerIndex], storage.read(t, segmentNames[writerIndex]))
				require.Equal(t, metadataContents[writerIndex], storage.read(t, metadataNames[writerIndex]))
			}
		})
	}
}
