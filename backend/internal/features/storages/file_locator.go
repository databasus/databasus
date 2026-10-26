package storages

import (
	"context"

	"github.com/google/uuid"

	storage_files "databasus-backend/internal/features/storages/files"
)

// The file store needs a provider, not a configuration, so the lookup stays here
// where provider selection already lives and the child package keeps no import of
// this one.
func (s *StorageService) GetFileWriter(
	ctx context.Context,
	storageID uuid.UUID,
) (storage_files.FileWriter, error) {
	return s.storageRepository.FindByID(ctx, storageID)
}

func (s *StorageService) GetFileReader(
	ctx context.Context,
	storageID uuid.UUID,
) (storage_files.FileReader, error) {
	return s.storageRepository.FindByID(ctx, storageID)
}

func (s *StorageService) GetFileRemover(
	ctx context.Context,
	storageID uuid.UUID,
) (storage_files.FileRemover, error) {
	return s.storageRepository.FindByID(ctx, storageID)
}
