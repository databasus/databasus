//go:build testhelpers

package audit_logs

import (
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/logger"
)

func loadWithoutApp() *audit.Log {
	return audit.New(audit.Deps{
		DB:          storage.GetDb(),
		Clock:       clock.NewSystem(),
		AuditLogger: logger.GetAuditLogger(),
	})
}
