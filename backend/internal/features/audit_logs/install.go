package audit_logs

import (
	"sync"
	"sync/atomic"

	"databasus-backend/internal/lib/audit"
)

var installedAuditLog atomic.Pointer[audit.Log]

var getAuditLogWithoutApp = sync.OnceValue(loadWithoutApp)

// InstallAuditLog hands the audit log internal/app built to the old code. A test binary built with
// the testhelpers tag builds one on first use instead.
func InstallAuditLog(auditLog *audit.Log) {
	installedAuditLog.Store(auditLog)
}

func getAuditLog() *audit.Log {
	if auditLog := installedAuditLog.Load(); auditLog != nil {
		return auditLog
	}

	return getAuditLogWithoutApp()
}
