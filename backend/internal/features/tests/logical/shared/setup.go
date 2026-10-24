//go:build testhelpers

// Package logicaltesting holds the engine-agnostic helpers shared by the
// per-engine logical backup/restore test packages (mysql, mariadb, mongodb,
// postgresql). It is normal (non-_test) code so the sibling test packages can
// import it; nothing in production imports anything under tests/, so it never
// enters the shipped binary.
package logicaltesting

import (
	"testing"

	"databasus-backend/internal/features/restores/restoring"
)

// Setup installs the in-process restorer the per-engine tests drive through the API. It returns a
// teardown to run from each engine package's TestMain so every parallel test binary runs against
// its own isolated control plane.
func Setup() func() {
	restorer := restoring.CreateTestRestorer()
	cancelRestore := restoring.StartRestorerForTest(&testing.T{}, restorer)

	return func() {
		restoring.StopRestorerForTest(&testing.T{}, cancelRestore, restorer)
	}
}
