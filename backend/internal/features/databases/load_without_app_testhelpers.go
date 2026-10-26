//go:build testhelpers

package databases_bridge

import (
	"databasus-backend/internal/config"
	notifiers_bridge "databasus-backend/internal/features/notifiers"
	users_bridge "databasus-backend/internal/features/users"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

func loadWithoutApp() databases.Module {
	return databases.New(databases.Deps{
		DB:         storage.GetDb(),
		Transactor: storage.GetTransactor(),
		Logger:     logger.GetLogger(),
		AuditLog: audit.New(
			audit.Deps{DB: storage.GetDb(), Clock: clock.NewSystem(), AuditLogger: logger.GetAuditLogger()},
		),
		FieldEncryptor: crypto.NewFieldEncryptor(encryption.GetMasterKey()),
		ClientToolsDir: config.GetEnv().ClientToolsDir,
		Users:          users_bridge.GetModule(),
		Workspaces:     workspaces_bridge.GetModule(),
		Notifiers:      notifiers_bridge.GetModule(),
	})
}
