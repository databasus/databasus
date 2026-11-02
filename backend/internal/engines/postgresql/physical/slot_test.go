package physical

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

//nolint:tparallel // Each version's server stops before the next version starts, which bounds the memory the servers of one test hold.
func TestInspectionSlot(t *testing.T) {
	t.Parallel()

	runOnEverySupportedVersion(t, func(t *testing.T, server testServer) {
		t.Helper()

		t.Run("reports allowance and current position with and without a slot wal limit", func(t *testing.T) {
			inspection := server.openInspection(t, getAdminRole())
			slotName := newSlotName()
			require.NoError(t, inspection.CreateSlot(t.Context(), slotName))

			positionBeforeInspect, err := inspection.GetCurrentWalLSN(t.Context())
			require.NoError(t, err)
			slot, err := inspection.GetSlot(t.Context(), slotName)
			require.NoError(t, err)
			require.NotNil(t, slot)
			require.NotNil(t, slot.CurrentLSN)
			require.NotNil(t, slot.LagBytes)
			assert.Nil(t, slot.SafeWalSizeBytes)
			assert.GreaterOrEqual(t, *slot.CurrentLSN, positionBeforeInspect)

			adminConn := server.connectAs(t, getAdminRole())
			execStatements(t, adminConn, "ALTER SYSTEM SET max_slot_wal_keep_size = '100GB'", "SELECT pg_reload_conf()")
			t.Cleanup(func() {
				_, _ = adminConn.Exec(context.WithoutCancel(t.Context()), "ALTER SYSTEM RESET max_slot_wal_keep_size")
				_, _ = adminConn.Exec(context.WithoutCancel(t.Context()), "SELECT pg_reload_conf()")
			})

			testwait.Until(t, "slot allowance is visible", func(ctx context.Context) error {
				slot, err := inspection.GetSlot(ctx, slotName)
				if err != nil {
					return err
				}
				if slot.SafeWalSizeBytes == nil || *slot.SafeWalSizeBytes <= 0 {
					return fmt.Errorf("slot has no positive allowance")
				}
				return nil
			})
		})
		t.Run("reports a taken name when creating a slot twice", func(t *testing.T) {
			t.Parallel()
			inspection := server.openInspection(t, getAdminRole())
			slotName := newSlotName()
			require.NoError(t, inspection.CreateSlot(t.Context(), slotName))
			require.ErrorIs(t, inspection.CreateSlot(t.Context(), slotName), ErrSlotNameTaken)
		})
		t.Run("drops a logical slot when replacing a per backup slot", func(t *testing.T) {
			t.Parallel()
			inspection := server.openInspection(t, getAdminRole())
			slotName := newSlotName()
			server.createLogicalSlot(t, slotName)
			require.NoError(t, inspection.DropBackupSlotIfExists(t.Context(), slotName))
			require.NoError(t, inspection.DropBackupSlotIfExists(t.Context(), slotName))
			slot, err := inspection.GetSlot(t.Context(), slotName)
			require.NoError(t, err)
			require.Nil(t, slot)
		})
	})
}
