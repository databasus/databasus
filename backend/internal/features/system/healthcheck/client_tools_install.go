package system_healthcheck

import (
	"sync/atomic"

	"databasus-backend/internal/engines/clienttools"
)

var installedClientToolsChecker atomic.Pointer[clienttools.Checker]

// InstallClientToolsChecker hands the checker that internal/app ran at startup to the health check,
// so a probe reuses the startup verdict on whether the clients start.
func InstallClientToolsChecker(checker *clienttools.Checker) {
	installedClientToolsChecker.Store(checker)
}

func getClientToolsChecker() *clienttools.Checker {
	checker := installedClientToolsChecker.Load()
	if checker == nil {
		panic("the system health check runs before app.New installed the client tools checker")
	}

	return checker
}
