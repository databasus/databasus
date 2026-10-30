package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	"databasus-backend/internal/features/caching"
	databases_bridge "databasus-backend/internal/features/databases"
	notifiers_bridge "databasus-backend/internal/features/notifiers"
	storages_bridge "databasus-backend/internal/features/storages"
	users_bridge "databasus-backend/internal/features/users"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

func newSeedBackups(t *testing.T) backups.Module {
	backupsModule := backups.New(backups.Deps{
		PhysicalExecutionClock: clock.NewSystem(),
		DB:                     storage.GetDb(),
		Transactor:             storage.GetTransactor(),
		Logger:                 logger.GetLogger(),
		Clock:                  clock.NewSystem(),
		AuditLog: audit.New(
			audit.Deps{DB: storage.GetDb(), Clock: clock.NewSystem(), AuditLogger: logger.GetAuditLogger()},
		),
		MasterKey:        encryption.GetMasterKey(),
		ClientToolsDir:   config.GetEnv().ClientToolsDir,
		TempDir:          config.GetEnv().TempFolder,
		WalQueueDir:      filepath.Join(config.GetEnv().DataFolder, "wal-queue"),
		WalStreamTimings: backups.ProductionWalStreamTimings(),
		Cache:            caching.GetStore(),
		Users:            users_bridge.GetModule(),
		Workspaces:       workspaces_bridge.GetModule(),
		Notifiers:        notifiers_bridge.GetModule(),
		Storages:         storages_bridge.GetModule(),
		Databases:        databases_bridge.GetModule(),
	})

	storages_bridge.GetModule().SubscribeToStorageTransferring(backupsModule.OnStorageTransferring)

	databasesModule := databases_bridge.GetModule()
	databasesModule.SubscribeToDatabaseCopied(backupsModule.OnDatabaseCopied)
	databasesModule.SubscribeToDatabaseBackupTypeChanged(backupsModule.OnDatabaseBackupTypeChanged)
	databasesModule.SubscribeToDatabaseDeleting(backupsModule.OnDatabaseDeleting)

	ctx, stop := context.WithCancel(t.Context())
	require.NoError(t, backupsModule.Start(ctx))
	t.Cleanup(func() {
		stop()
		waitCtx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Minute)
		defer cancel()
		require.NoError(t, backupsModule.WaitForRuns(waitCtx))
	})
	return backupsModule
}
