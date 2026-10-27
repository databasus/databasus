package backups_dto_logical

import (
	"io"
	"time"

	"github.com/google/uuid"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/lib/crypto"
)

type GetBackupsRequest struct {
	DatabaseID string     `form:"database_id" binding:"required"`
	Limit      int        `form:"limit"`
	Offset     int        `form:"offset"`
	Statuses   []string   `form:"status"`
	BeforeDate *time.Time `form:"beforeDate"`
}

type GetBackupsResponse struct {
	Backups []*backups_core_logical.LogicalBackup `json:"backups"`
	Total   int64                                 `json:"total"`
	Limit   int                                   `json:"limit"`
	Offset  int                                   `json:"offset"`
}

type DecryptionReaderCloser struct {
	*crypto.DecryptingReader
	BaseReader io.ReadCloser
}

func (r *DecryptionReaderCloser) Close() error {
	return r.BaseReader.Close()
}

type MakeBackupRequest struct {
	DatabaseID uuid.UUID `json:"database_id" binding:"required"`
}
