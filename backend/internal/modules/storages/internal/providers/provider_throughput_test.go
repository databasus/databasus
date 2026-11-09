package providers_test

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

type archiveFilePair struct {
	segmentName  string
	metadataName string
}

func TestSaveFileThroughput(t *testing.T) {
	t.Parallel()

	const (
		writerCount = 4
		pairCount   = 24
		writeLimit  = 12 * time.Second
	)

	for _, target := range getProviderTargets(true) {
		t.Run(
			"stores 24 segment and metadata pairs within 12 seconds with four writers on "+target.name,
			func(t *testing.T) {
				t.Parallel()
				container := target.startContainer(t)
				storage := target.build(t, container)
				isNetworkStorage := target.name != "Local"
				if isNetworkStorage {
					containers.AddEgressDelay(t, container, roundTripEgressDelay)
					t.Cleanup(func() { containers.ClearEgressShaping(t, container) })
				}

				segmentContent := generateRandomContent(t, segmentSizedFileBytes)
				metadataContent := generateRandomContent(t, metadataSizedFileBytes)
				filePairs := make([]archiveFilePair, pairCount)
				for pairIndex := range filePairs {
					filePairs[pairIndex] = archiveFilePair{
						segmentName:  "segment-" + uuid.NewString(),
						metadataName: "metadata-" + uuid.NewString(),
					}
					storage.savedFileNames = append(storage.savedFileNames,
						filePairs[pairIndex].segmentName, filePairs[pairIndex].metadataName)
				}

				writeCtx, cancel := context.WithTimeout(t.Context(), writeLimit)
				defer cancel()
				startWrites := make(chan struct{})
				writeErrors := make(chan error, writerCount)
				var writers sync.WaitGroup
				for writerIndex := range writerCount {
					writers.Go(func() {
						<-startWrites
						for pairIndex := writerIndex; pairIndex < len(filePairs); pairIndex += writerCount {
							filePair := filePairs[pairIndex]
							if err := storage.provider.SaveFile(
								writeCtx,
								storage.access(filePair.segmentName),
								bytes.NewReader(segmentContent),
							); err != nil {
								writeErrors <- fmt.Errorf("write segment of pair %d: %w", pairIndex, err)
								return
							}

							if err := storage.provider.SaveFile(
								writeCtx,
								storage.access(filePair.metadataName),
								bytes.NewReader(metadataContent),
							); err != nil {
								writeErrors <- fmt.Errorf("write metadata of pair %d: %w", pairIndex, err)
								return
							}
						}
					})
				}

				startedAt := time.Now()
				close(startWrites)
				writers.Wait()
				writeDuration := time.Since(startedAt)
				close(writeErrors)

				for writeError := range writeErrors {
					require.NoError(t, writeError)
				}
				assert.LessOrEqual(t, writeDuration, writeLimit)
				t.Logf("stored %d pairs with %d writers in %s", pairCount, writerCount, writeDuration)

				if isNetworkStorage {
					containers.ClearEgressShaping(t, container)
				}

				for _, filePair := range filePairs {
					require.Equal(t, segmentContent, storage.read(t, filePair.segmentName))
					require.Equal(t, metadataContent, storage.read(t, filePair.metadataName))
				}
			},
		)
	}
}
