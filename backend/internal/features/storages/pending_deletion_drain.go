package storages

import (
	"context"
	"fmt"
	"strings"
	"time"

	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	"databasus-backend/internal/util/logger"
)

// The drain runs in the request that deletes the storage, so this is how long it
// keeps starting files. The deadline is read between files and a provider holds
// its own deadline on the call it already began, so an unreachable one costs this
// plus one provider timeout.
const storageDeletionDrainBudget = 10 * time.Second

// A deleted storage takes its pending deletions with it, so this is the last
// moment the credentials to carry them out exist. What the drain cannot remove is
// named in the audit log, because those files stay in the user's storage.
func (s *StorageService) drainPendingDeletions(ctx context.Context, storage *Storage) {
	remaining, err := getStorageFileDeletionWorker().DrainStorage(ctx, storage.ID, storageDeletionDrainBudget)
	if err != nil {
		logger.GetLogger().Warn("failed to drain pending deletions before removing a storage",
			"storage_id", storage.ID, "error", err)
	}

	if len(remaining) == 0 {
		return
	}

	message := fmt.Sprintf("Storage deleted with %d file(s) left behind: %s",
		len(remaining), strings.Join(remaining, ", "))

	logger.GetLogger().Warn(message, "storage_id", storage.ID)

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     message,
		WorkspaceID: &storage.WorkspaceID,
	})
}
