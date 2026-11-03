package backuping

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/lib/toolstub"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/storages"
)

// archiveFileSizeFittingAnyPipe keeps the archive inside the pipe buffer, so the tool can exit while
// the storage is stuck. The kernel gives a new pipe only 8 KiB once the user holds too many pipes,
// which happens while several test runs share a machine.
const archiveFileSizeFittingAnyPipe = 1024

func TestRunStream(t *testing.T) {
	t.Parallel()

	t.Run("reports a network stall when the storage never returns after accepting data", func(t *testing.T) {
		t.Parallel()
		spec := newStuckStorageStreamSpec(t, writeTarOfSize(t, 80*1024))

		outcome := waitForStreamOutcome(t, startStream(t.Context(), spec))

		require.NotNil(t, outcome.ErrorReason)
		assert.Equal(t, catalog.BackupErrorNetworkStallTimeout, *outcome.ErrorReason)
	})

	t.Run("reports a network stall when the storage stops before the last encrypted chunk", func(t *testing.T) {
		t.Parallel()
		spec := newStuckStorageStreamSpec(t, writeTarOfSize(t, 1024))
		masterKey, err := crypto.OpenOrCreateMasterKey(filepath.Join(t.TempDir(), "secret.key"))
		require.NoError(t, err)
		spec.Params.Encryption = catalog.BackupEncryptionEncrypted
		spec.Params.MasterKey = masterKey

		outcome := waitForStreamOutcome(t, startStream(t.Context(), spec))

		require.NotNil(t, outcome.ErrorReason)
		assert.Equal(t, catalog.BackupErrorNetworkStallTimeout, *outcome.ErrorReason)
	})

	t.Run("reports a cancellation within the limit when the tool exited and the storage is stuck", func(t *testing.T) {
		t.Parallel()
		tarPath := writeTarOfSize(t, archiveFileSizeFittingAnyPipe)
		spec := newStuckStorageStreamSpec(t, tarPath)
		spec.Backup.StallTimeout = time.Hour
		firstReadMarkerPath := filepath.Join(t.TempDir(), "first-read")
		spec.Params.FileStore = &stuckStorage{firstReadMarkerPath: firstReadMarkerPath}
		exitMarkerPath := filepath.Join(t.TempDir(), "exited")
		// The tool sends the rest only after the storage read the first part, so a write stays blocked
		// on the stuck storage when the tool exits.
		toolstub.Write(
			t,
			spec.Backup.ClientTools.GetExecutable(pgshared.Version17, pgshared.ExecutablePgBasebackup),
			toolstub.Spec{
				Script: fmt.Sprintf(
					"head -c 512 %q\nuntil [ -e %q ]; do sleep 0.01; done\ntail -c +513 %q\ntouch %q",
					tarPath, firstReadMarkerPath, tarPath, exitMarkerPath,
				),
			},
		)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		completed := startStream(ctx, spec)

		testwait.Until(t, "tool exits", func(context.Context) error {
			_, err := os.Stat(exitMarkerPath)

			return err
		})

		cancelledAt := time.Now()
		cancel()
		outcome := waitForStreamOutcome(t, completed)

		assert.Equal(t, catalog.BackupStatusCanceled, outcome.Status)
		assert.Less(t, time.Since(cancelledAt), 10*time.Second)
	})
}

func newStuckStorageStreamSpec(t *testing.T, tarPath string) streamSpec {
	t.Helper()

	clientTools := pgshared.ClientTools{Root: t.TempDir()}
	toolstub.Write(t, clientTools.GetExecutable(pgshared.Version17, pgshared.ExecutablePgBasebackup), toolstub.Spec{
		Script: fmt.Sprintf("cat %q", tarPath),
	})
	logger := slog.New(slog.DiscardHandler)

	return streamSpec{
		Params: runStreamParams{
			FileStore:  &stuckStorage{},
			StorageID:  uuid.New(),
			Logger:     logger,
			Source:     pgphysical.NewSource(pgphysical.Settings{}),
			FileName:   "stuck-backup",
			Encryption: catalog.BackupEncryptionNone,
			BackupID:   uuid.New(),
			Codec:      catalog.BackupCompressionNone,
		},
		Backup: pgphysical.BaseBackupSpec{
			ClientTools:       clientTools,
			Version:           pgshared.Version17,
			Credentials:       &pgshared.CredentialTempFiles{PgpassPath: "/tmp/pgpass"},
			Label:             "stuck-backup",
			Codec:             pgphysical.CodecNone,
			SystemID:          7361234567890123456,
			Clock:             clock.NewSystem(),
			FirstByteTimeout:  time.Hour,
			StallTimeout:      200 * time.Millisecond,
			StallPollInterval: time.Millisecond,
			ProgressInterval:  time.Hour,
			Logger:            logger,
		},
		Classify: func(streamErr error, _ []byte) streamOutcome {
			return streamOutcome{ErrorMessage: streamErr.Error()}
		},
	}
}

func startStream(ctx context.Context, spec streamSpec) <-chan streamOutcome {
	completed := make(chan streamOutcome, 1)

	go func() {
		outcome, _ := runStream(ctx, spec)
		completed <- outcome
	}()

	return completed
}

func waitForStreamOutcome(t *testing.T, completed <-chan streamOutcome) streamOutcome {
	t.Helper()

	var outcome streamOutcome

	testwait.Until(t, "the stream ends", func(context.Context) error {
		select {
		case outcome = <-completed:
			return nil
		default:
			return errors.New("stream is still running")
		}
	})

	return outcome
}

func writeTarOfSize(t *testing.T, fileSize int) string {
	t.Helper()

	var archive bytes.Buffer
	writer := tar.NewWriter(&archive)
	fileContent := make([]byte, fileSize)
	require.NoError(t, writer.WriteHeader(&tar.Header{
		Name:     "PG_VERSION",
		Mode:     0o600,
		Size:     int64(len(fileContent)),
		Typeflag: tar.TypeReg,
	}))
	_, err := writer.Write(fileContent)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	path := filepath.Join(t.TempDir(), "base.tar")
	require.NoError(t, os.WriteFile(path, archive.Bytes(), 0o600))

	return path
}

// stuckStorage accepts the first chunk of an upload and then never reads again, until its context
// ends, the way a storage that stopped responding behaves.
type stuckStorage struct {
	storages.Module

	firstReadMarkerPath string
}

func (s *stuckStorage) StartBackgroundWrite(
	ctx context.Context,
	spec storages.BackgroundWriteSpec,
) storages.BackgroundWrite {
	writeErrors := make(chan error, 1)

	go func() {
		_, _ = spec.Source.Read(make([]byte, 64*1024))
		if s.firstReadMarkerPath != "" {
			_ = os.WriteFile(s.firstReadMarkerPath, nil, 0o600)
		}

		<-ctx.Done()
		writeErrors <- ctx.Err()
	}()

	return storages.BackgroundWrite{Errors: writeErrors, Receipts: make(chan storages.WriteReceipt, 1)}
}

func TestManifestSaveTimeout(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name             string
		payloadSizeBytes int
		timeout          time.Duration
	}{
		{"uses the minimum timeout with an empty manifest", 0, manifestSaveMinTimeout},
		{"uses the minimum timeout with a 1 mib manifest", 1024 * 1024, manifestSaveMinTimeout},
		{"scales the timeout with a 100 mib manifest", 100 * 1024 * 1024, 400 * time.Second},
		{"caps the timeout with a 1 gib manifest", 1024 * 1024 * 1024, manifestSaveMaxTimeout},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.timeout, manifestSaveTimeout(testCase.payloadSizeBytes))
		})
	}
}
