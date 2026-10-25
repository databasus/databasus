//go:build !testhelpers

package audit_logs

import "databasus-backend/internal/lib/audit"

func loadWithoutApp() *audit.Log {
	panic("audit_logs writes an audit event before app.New installed the audit log")
}
