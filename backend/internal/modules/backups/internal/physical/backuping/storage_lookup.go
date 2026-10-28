package backuping

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages"
)

func getStorage(ctx context.Context, storagesModule storages.Module, storageID uuid.UUID) (storages.Storage, error) {
	foundStorages, err := storagesModule.GetStorages(ctx, []uuid.UUID{storageID})
	if err != nil {
		return storages.Storage{}, err
	}

	if len(foundStorages) == 0 {
		return storages.Storage{}, fmt.Errorf("%w: %s", errStorageNotFound, storageID)
	}

	return foundStorages[0], nil
}
