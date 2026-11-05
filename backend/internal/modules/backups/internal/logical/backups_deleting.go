package logical

import (
	"context"

	"github.com/google/uuid"
)

type BackupsDeleting struct {
	BackupIDs []uuid.UUID
}

type BackupsDeletingHandler func(ctx context.Context, event BackupsDeleting) error
