package local

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages/internal/providers"
)

const recoveryProbeBytes = 64

// probeFiles reads and removes the recovery probe; tests replace it to fail a step the real file
// system cannot be made to fail on demand.
type probeFiles struct {
	readFile func(path string) ([]byte, error)
	remove   func(path string) error
}

func getOSProbeFiles() probeFiles {
	return probeFiles{readFile: os.ReadFile, remove: os.Remove}
}

// TestRecoveredConnection checks the backup folder itself, which the ordinary test leaves out: a
// recovered storage points at a host folder the Databasus backup did not restore.
func (p *Provider) TestRecoveredConnection(_ context.Context, _ providers.Location) error {
	return probeBackupFolder(p.folders.DataFolder, getOSProbeFiles())
}

// probeBackupFolder writes a unique probe in the folder, reads it back, compares it and removes it,
// touching no other file. A failed step still attempts the removal.
func probeBackupFolder(folder string, files probeFiles) (probeErr error) {
	probePath := filepath.Join(folder, ".databasus-recovery-probe-"+uuid.NewString())

	defer func() {
		if probeErr != nil {
			_ = files.remove(probePath)
		}
	}()

	content := make([]byte, recoveryProbeBytes)
	if _, err := rand.Read(content); err != nil {
		return fmt.Errorf("generate the backup folder probe: %w", err)
	}

	if err := writeNewFile(probePath, content); err != nil {
		return fmt.Errorf("%w: write %s: %w", ErrBackupFolderProbeFailed, folder, err)
	}

	readContent, err := files.readFile(probePath)
	if err != nil {
		return fmt.Errorf("%w: read %s: %w", ErrBackupFolderProbeFailed, folder, err)
	}

	if !bytes.Equal(readContent, content) {
		return fmt.Errorf("%w: %s returned other content", ErrBackupFolderProbeFailed, folder)
	}

	if err := files.remove(probePath); err != nil {
		return fmt.Errorf("%w: remove the probe from %s: %w", ErrBackupFolderProbeFailed, folder, err)
	}

	return nil
}

// writeNewFile refuses an existing path, so the probe can never overwrite a backup file.
func writeNewFile(path string, content []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}

	_, writeErr := file.Write(content)
	syncErr := file.Sync()
	closeErr := file.Close()

	return errors.Join(writeErr, syncErr, closeErr)
}
