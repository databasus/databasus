package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	old_config "databasus-backend/internal/config"
	local_storage "databasus-backend/internal/features/storages/models/local"
	"databasus-backend/internal/util/encryption"
)

const (
	storagePermissionsDocumentationURL = "https://databasus.com/advanced-config/#docker-storage-permissions"
	storageProbeContent                = "databasus storage test"
)

// ProbeStorage saves and deletes a file through local storage. Container startup runs it before the
// internal database exists, so it needs neither the database nor New. The error describes what the
// operator has to fix.
func ProbeStorage(ctx context.Context, environment *Environment) error {
	old_config.Install(environment.Config)

	return probeStorage(ctx, environment, &local_storage.LocalStorage{})
}

type probedStorage interface {
	SaveFile(
		ctx context.Context,
		encryptor encryption.FieldEncryptor,
		logger *slog.Logger,
		fileName string,
		file io.Reader,
	) error
	DeleteFile(ctx context.Context, encryptor encryption.FieldEncryptor, logger *slog.Logger, fileName string) error
}

func probeStorage(ctx context.Context, environment *Environment, localStorage probedStorage) error {
	probeLogger := environment.Logging.GetLogger()
	fieldEncryptor := encryption.GetFieldEncryptor()
	probeFileName := ".databasus-storage-test-" + uuid.NewString()

	err := localStorage.SaveFile(
		ctx,
		fieldEncryptor,
		probeLogger,
		probeFileName,
		strings.NewReader(storageProbeContent),
	)
	if err != nil {
		removeTemporaryProbe(ctx, probeLogger, filepath.Join(environment.Config.TempFolder, probeFileName))

		return newStorageProbeError(environment, "save a file through local storage", err)
	}

	err = localStorage.DeleteFile(ctx, fieldEncryptor, probeLogger, probeFileName)
	if err != nil {
		return newStorageProbeError(environment, "delete a file through local storage", err)
	}

	return nil
}

func removeTemporaryProbe(ctx context.Context, probeLogger *slog.Logger, temporaryProbePath string) {
	err := os.Remove(temporaryProbePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		probeLogger.WarnContext(ctx, "failed to remove storage test temporary file",
			slog.String("path", temporaryProbePath), slog.Any("error", err))
	}
}

func newStorageProbeError(environment *Environment, operation string, operationError error) error {
	return &StorageProbeError{
		TempFolder: environment.Config.TempFolder,
		DataFolder: environment.Config.DataFolder,
		Operation:  operation,
		Err:        operationError,
	}
}
