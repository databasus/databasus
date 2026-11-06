package pg17_test

import (
	"testing"

	"databasus-backend/internal/app/apptest/physicaltest"
)

func TestPhysicalBackupWithRLS(t *testing.T) {
	t.Parallel()

	t.Run("restores every row and rls policy with a full backup without row access", func(t *testing.T) {
		t.Parallel()
		physicaltest.RunRLSRestore(t, "postgres:17", false)
	})

	t.Run("restores both stages and rls policy after rls is introduced before an incremental", func(t *testing.T) {
		t.Parallel()
		physicaltest.RunRLSRestore(t, "postgres:17", true)
	})
}
