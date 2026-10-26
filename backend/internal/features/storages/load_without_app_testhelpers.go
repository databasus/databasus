//go:build testhelpers

package storages_bridge

import (
	"time"

	"databasus-backend/internal/config"
	users_bridge "databasus-backend/internal/features/users"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

// fileTimingsWithoutApp keep every bound long enough to be observable in a single test and short
// enough that no old test waits on a production interval.
func fileTimingsWithoutApp() storages.FileTimings {
	return storages.FileTimings{
		CommitWindow:   2 * time.Second,
		AttemptLease:   time.Second,
		RetryBaseDelay: 10 * time.Millisecond,
		RetryMaxDelay:  50 * time.Millisecond,
		WorkerTick:     10 * time.Millisecond,
		ClaimBatchSize: 10,
	}
}

func loadWithoutApp() storages.Module {
	return storages.New(storages.Deps{
		DB:         storage.GetDb(),
		Transactor: storage.GetTransactor(),
		Logger:     logger.GetLogger(),
		AuditLog: audit.New(
			audit.Deps{DB: storage.GetDb(), Clock: clock.NewSystem(), AuditLogger: logger.GetAuditLogger()},
		),
		FieldEncryptor: crypto.NewFieldEncryptor(encryption.GetMasterKey()),
		Users:          users_bridge.GetModule(),
		Workspaces:     workspaces_bridge.GetModule(),
		Clock:          clock.NewSystem(),
		LocalFolders: storages.LocalFolders{
			TempFolder: config.GetEnv().TempFolder,
			DataFolder: config.GetEnv().DataFolder,
		},
		FileTimings: fileTimingsWithoutApp(),
	})
}
