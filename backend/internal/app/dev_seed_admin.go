package app

import (
	"context"
	"log/slog"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/users"
)

type DevSeedAdminRequest struct {
	EnvMode     config.EnvMode
	DatabaseDsn string
	Account     users.DevSeedAccount
	Clock       clock.Source
	AuditLogger *slog.Logger
}

func EnsureDevSeedBootstrapAdmin(ctx context.Context, request DevSeedAdminRequest) (bool, error) {
	if request.EnvMode != config.EnvModeDevelopment {
		return false, ErrDevSeedNotDevelopment
	}

	if request.DatabaseDsn == "" {
		return false, ErrDevSeedDatabaseRequired
	}

	db, err := database.Open(ctx, request.DatabaseDsn)
	if err != nil {
		return false, &devSeedDatabaseError{err: err}
	}

	defer closeDatabase(db)

	admin := users.NewDevSeedAdmin(users.DevSeedAdminDeps{
		DB:         db,
		Transactor: database.NewTransactor(db),
		AuditLog: audit.New(audit.Deps{
			DB:          db,
			Clock:       request.Clock,
			AuditLogger: request.AuditLogger,
		}),
	})

	return admin.EnsureBootstrapAdmin(ctx, request.Account)
}
