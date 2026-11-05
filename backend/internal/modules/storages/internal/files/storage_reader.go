package files

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"
)

// StorageReader opens files of one storage. The returned reader may keep fetching from the provider
// while it is read, so ctx must outlive the stream.
type StorageReader interface {
	OpenFile(ctx context.Context, fileName string) (io.ReadCloser, error)
}

// GetStorageReader locates the storage once, so a caller reading many files of one storage, such as a
// restore stream over thousands of WAL segments, does not load it again for each file.
func (s *Store) GetStorageReader(ctx context.Context, storageID uuid.UUID) (StorageReader, error) {
	located, err := s.locator.LocateStorage(ctx, storageID)
	if err != nil {
		return nil, fmt.Errorf("locate storage for read: %w", err)
	}

	return &storageReader{located: located, logger: s.logger}, nil
}

type storageReader struct {
	located LocatedStorage
	logger  *slog.Logger
}

func (r *storageReader) OpenFile(ctx context.Context, fileName string) (io.ReadCloser, error) {
	return r.located.Provider.OpenFile(ctx, r.located.getFileAccess(fileName, r.logger))
}
