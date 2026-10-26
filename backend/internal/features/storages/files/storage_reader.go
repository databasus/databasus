package storage_files

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/util/encryption"
)

// GetStorageReader loads the storage once, so a caller reading many files of one storage, such as
// a restore stream over thousands of WAL segments, does not load it again for each file.
func (s *Store) GetStorageReader(ctx context.Context, storageID uuid.UUID) (StorageReader, error) {
	fileReader, err := s.locator.GetFileReader(ctx, storageID)
	if err != nil {
		return nil, fmt.Errorf("locate storage for read: %w", err)
	}

	return &storageReader{
		fileReader:     fileReader,
		fieldEncryptor: s.fieldEncryptor,
		logger:         s.logger.With("storage_id", storageID),
	}, nil
}

type storageReader struct {
	fileReader     FileReader
	fieldEncryptor encryption.FieldEncryptor
	logger         *slog.Logger
}

func (r *storageReader) OpenFile(ctx context.Context, fileName string) (io.ReadCloser, error) {
	return r.fileReader.GetFile(ctx, r.fieldEncryptor, r.logger.With("file_name", fileName), fileName)
}
