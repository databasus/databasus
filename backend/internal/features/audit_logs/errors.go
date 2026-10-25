package audit_logs

import "errors"

var (
	ErrOnlyAdminsCanViewGlobalLogs = errors.New(
		"only administrators can view global audit logs",
	)
	ErrInsufficientPermissionsToViewLogs = errors.New(
		"insufficient permissions to view user audit logs",
	)
	ErrInsufficientPermissionsToViewWorkspaceLogs = errors.New(
		"insufficient permissions to view workspace audit logs",
	)
)
