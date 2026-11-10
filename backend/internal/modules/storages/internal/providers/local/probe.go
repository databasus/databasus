package local

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages/internal/providers"
)

const probeContent = "databasus storage test"

type ProbeSpec struct {
	Folders providers.LocalFolders
	Logger  *slog.Logger
}

// Probe saves and deletes a file through local storage, so a container whose folders the process
// cannot write stops at startup instead of at the first backup.
func Probe(ctx context.Context, spec ProbeSpec) error {
	provider := New(Deps{Repositories: Repositories{settings: nil}, Folders: spec.Folders})
	probeFile := providers.FileAccess{
		StorageID: uuid.Nil, Settings: Settings{},
		FileName: ".databasus-storage-test-" + uuid.NewString(),
		Logger:   spec.Logger,
	}

	if err := provider.SaveFile(ctx, probeFile, strings.NewReader(probeContent)); err != nil {
		removeTemporaryProbe(ctx, spec.Logger, filepath.Join(spec.Folders.TempFolder, probeFile.FileName))

		return fmt.Errorf("%w: %w", ErrProbeSaveFailed, err)
	}

	if err := provider.DeleteFile(ctx, probeFile); err != nil {
		return fmt.Errorf("%w: %w", ErrProbeDeleteFailed, err)
	}

	return nil
}

func removeTemporaryProbe(ctx context.Context, logger *slog.Logger, temporaryProbePath string) {
	err := os.Remove(temporaryProbePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		logger.WarnContext(ctx, "failed to remove storage test temporary file",
			slog.String("path", temporaryProbePath), slog.Any("error", err))
	}
}
