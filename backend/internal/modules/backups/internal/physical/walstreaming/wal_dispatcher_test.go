package walstreaming

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSegmentUploadDispatcher(t *testing.T) {
	t.Run("keeps four uploads active and starts the next segment when a slot becomes free", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		started := make(chan string, 8)
		release := make(chan struct{}, 8)
		highestCount := 0
		dispatcher := newSegmentUploadDispatcher(func(ctx context.Context, _, filename string) error {
			started <- filename
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}, func(count int) { highestCount = max(highestCount, count) })
		defer func() { cancel(); dispatcher.waitForUploads(t.Context()) }()

		segments := []queuedWalSegment{}
		for segmentNo := uint64(100); segmentNo < 108; segmentNo++ {
			segments = append(segments, queuedWalSegment{walFilename: walName(1, segmentNo)})
		}
		pending := dispatcher.startUploads(ctx, slog.New(slog.DiscardHandler), segments)

		firstStarted := []string{<-started, <-started, <-started, <-started}
		slices.Sort(firstStarted)
		assert.Equal(t, []string{walName(1, 100), walName(1, 101), walName(1, 102), walName(1, 103)}, firstStarted)
		require.Len(t, pending, 4)
		assert.Equal(t, 4, dispatcher.getRunningBackupCount())

		release <- struct{}{}
		dispatcher.markFinished(<-dispatcher.finishedUploads)
		pending = dispatcher.startUploads(ctx, slog.New(slog.DiscardHandler), pending)
		assert.Equal(t, walName(1, 104), <-started)
		assert.Len(t, pending, 3)
		assert.Equal(t, 4, highestCount)
	})
	t.Run(
		"starts an older pending segment before a newer receivewal segment when a slot becomes free",
		func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			started := make(chan string, 6)
			release := make(chan struct{}, 1)
			dispatcher := newSegmentUploadDispatcher(func(ctx context.Context, _, filename string) error {
				started <- filename
				select {
				case <-release:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}, func(int) {})
			defer func() { cancel(); dispatcher.waitForUploads(t.Context()) }()

			segments := []queuedWalSegment{}
			for _, segmentNo := range []uint64{110, 111, 112, 113, 115, 114} {
				segments = append(
					segments,
					queuedWalSegment{
						walFilename: walName(1, segmentNo),
						timelineID:  1,
						startLSN:    segmentStartLSN(segmentNo),
					},
				)
			}

			sortQueuedWalSegments(segments)
			pending := dispatcher.startUploads(ctx, slog.New(slog.DiscardHandler), segments)

			for range 4 {
				<-started
			}

			release <- struct{}{}
			dispatcher.markFinished(<-dispatcher.finishedUploads)
			dispatcher.startUploads(ctx, slog.New(slog.DiscardHandler), pending)
			assert.Equal(t, walName(1, 114), <-started)
		},
	)
	t.Run("starts a blocked filename once when repeated listings move it into pending upload", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		started := make(chan string, 2)
		dispatcher := newSegmentUploadDispatcher(
			func(ctx context.Context, path, _ string) error { started <- path; <-ctx.Done(); return nil },
			func(int) {},
		)
		defer func() { cancel(); dispatcher.waitForUploads(t.Context()) }()
		captured := &capturedLog{}
		logger := slog.New(slog.NewTextHandler(captured, nil))
		segment := queuedWalSegment{walFilename: walName(1, 150), localPath: "receivewal/segment"}

		dispatcher.startUploads(ctx, logger, []queuedWalSegment{segment})
		assert.Equal(t, segment.localPath, <-started)
		segment.localPath = "pending-upload/segment"

		for range 3 {
			assert.Empty(t, dispatcher.startUploads(ctx, logger, []queuedWalSegment{segment}))
		}

		assert.Equal(t, 1, dispatcher.getRunningBackupCount())
		assert.Empty(t, started)
		assert.Empty(t, captured.String())
	})
	t.Run("retries a failed upload when the next pass lists its retained file", func(t *testing.T) {
		attempts := atomic.Int64{}
		dispatcher := newSegmentUploadDispatcher(
			func(context.Context, string, string) error { attempts.Add(1); return errors.New("storage unavailable") },
			func(int) {},
		)
		defer dispatcher.waitForUploads(t.Context())
		segments := []queuedWalSegment{{walFilename: walName(1, 71), localPath: "pending-upload/segment"}}

		for range 2 {
			dispatcher.startUploads(t.Context(), slog.New(slog.DiscardHandler), segments)
			dispatcher.markFinished(<-dispatcher.finishedUploads)
		}

		assert.Equal(t, int64(2), attempts.Load())
	})
}

func TestRecoverQueuedWalSegments(t *testing.T) {
	t.Run("uses concurrent slots for every segment when startup recovers a queue", func(t *testing.T) {
		arrived := make(chan struct{}, 4)
		release := make(chan struct{})
		recovered := make(chan string, 6)
		counts := []int{}
		segments := []queuedWalSegment{}
		for segmentNo := uint64(130); segmentNo < 136; segmentNo++ {
			segments = append(segments, queuedWalSegment{walFilename: walName(1, segmentNo)})
		}

		completed := make(chan struct{})
		go func() {
			defer close(completed)
			recoverQueuedWalSegments(t.Context(), slog.New(slog.DiscardHandler), queuedWalRecoverySpec{
				Segments: segments, RecordConcurrentUploads: func(count int) { counts = append(counts, count) },
				Upload: func(ctx context.Context, _, filename string) error {
					arrived <- struct{}{}
					select {
					case <-release:
					case <-ctx.Done():
						return ctx.Err()
					}
					recovered <- filename
					return nil
				},
			})
		}()

		for range 4 {
			<-arrived
		}
		close(release)
		<-completed

		assert.Len(t, recovered, 6)
		assert.Equal(t, 4, slices.Max(counts))
	})
	t.Run("attempts every segment once when storage fails throughout startup", func(t *testing.T) {
		attempts := atomic.Int64{}
		segments := make([]queuedWalSegment, 5)
		for index := range segments {
			segments[index].walFilename = walName(1, uint64(140+index))
		}

		recoverQueuedWalSegments(
			t.Context(),
			slog.New(slog.DiscardHandler),
			queuedWalRecoverySpec{
				Segments:                segments,
				RecordConcurrentUploads: func(int) {},
				Upload:                  func(context.Context, string, string) error { attempts.Add(1); return errors.New("storage unavailable") },
			},
		)

		assert.Equal(t, int64(5), attempts.Load())
	})
}

func TestQueuedWalSegments(t *testing.T) {
	t.Run("keeps the oldest queued position when newer segments finish first", func(t *testing.T) {
		watchDir := t.TempDir()
		writeWalFile(t, watchDir, walName(1, 50))
		newestPath := writeWalFile(t, watchDir, walName(1, 53))
		require.NoError(t, os.Remove(newestPath))

		segments := listQueuedWalSegments(watchDir, testWalSegmentSize)
		sortQueuedWalSegments(segments)
		supervisor := &Streamer{}
		supervisor.setLowestQueuedSegment(segments)

		assert.Equal(t, segmentStart(1, 50), supervisor.getLowestQueuedSegment())
	})
	t.Run("counts a short segment as the lowest queued position when it awaits restreaming", func(t *testing.T) {
		watchDir := t.TempDir()
		pendingDir := filepath.Join(watchDir, pendingUploadDirName)
		require.NoError(t, os.MkdirAll(pendingDir, 0o700))
		writeSegmentOfSize(t, watchDir, walName(1, 60), testWalSegmentSize/2)
		writeWalFile(t, pendingDir, walName(1, 61))
		writeSegmentOfSize(t, watchDir, "00000002.history", 64)

		segments := listQueuedWalSegments(watchDir, testWalSegmentSize)
		sortQueuedWalSegments(segments)

		require.Len(t, segments, 2)
		assert.False(t, segments[0].isComplete)
		assert.True(t, segments[1].isComplete)
		assert.Equal(t, filepath.Join(pendingDir, walName(1, 61)), segments[1].localPath)

		supervisor := &Streamer{}
		supervisor.setLowestQueuedSegment(segments)
		assert.Equal(t, segmentStart(1, 60), supervisor.getLowestQueuedSegment())
	})
}
