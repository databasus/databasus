package config

import (
	"context"

	"github.com/google/uuid"
)

type StoragePublication struct {
	DatabaseID uuid.UUID
	StorageID  uuid.UUID
	Generation int64
}

type PublicationFence struct {
	Configs     *Service
	Publication StoragePublication
}

func (f PublicationFence) Validate(ctx context.Context) error {
	return f.Configs.CheckStoragePublication(ctx, f.Publication)
}
