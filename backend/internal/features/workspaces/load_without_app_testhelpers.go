//go:build testhelpers

package workspaces_bridge

import (
	"databasus-backend/internal/config"
	"databasus-backend/internal/features/email"
	users_bridge "databasus-backend/internal/features/users"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/logger"
)

func loadWithoutApp() workspaces.Module {
	return NewModuleWithoutApp(users_bridge.GetModule())
}

// NewModuleWithoutApp builds the module from the old access points on top of the given users
// module, for a test that serves the module's routes itself.
func NewModuleWithoutApp(usersModule users.Module) workspaces.Module {
	systemClock := clock.NewSystem()

	return workspaces.New(workspaces.Deps{
		DB:         storage.GetDb(),
		Transactor: database.NewTransactor(storage.GetDb()),
		Clock:      systemClock,
		Logger:     logger.GetLogger(),
		Mailer:     email.GetInstanceSender(),
		AuditLog: audit.New(
			audit.Deps{DB: storage.GetDb(), Clock: systemClock, AuditLogger: logger.GetAuditLogger()},
		),
		Users:     usersModule,
		PublicURL: config.GetEnv().DatabasusURL,
	})
}
