package physical

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifySlot(t *testing.T) {
	t.Parallel()

	t.Run("returns slot name error without a slot name", func(t *testing.T) {
		t.Parallel()

		err := NewSource(Settings{}).VerifySlot(t.Context(), newTestLogger(t), "")

		require.ErrorIs(t, err, ErrSlotNameEmpty)
	})

	runOnEverySupportedVersion(t, func(t *testing.T, server testServer) {
		t.Helper()

		t.Run("creates a physical slot when the slot is missing", func(t *testing.T) {
			t.Parallel()
			slotName := newSlotName()

			err := server.newSource(getAdminRole()).VerifySlot(t.Context(), newTestLogger(t), slotName)
			require.NoError(t, err)

			slotType, isSlotPresent := server.readSlotType(t, slotName)
			require.True(t, isSlotPresent)
			assert.Equal(t, "physical", slotType)
		})

		t.Run("keeps the existing slot when called a second time", func(t *testing.T) {
			t.Parallel()
			slotName := newSlotName()
			source := server.newSource(getAdminRole())

			require.NoError(t, source.VerifySlot(t.Context(), newTestLogger(t), slotName))
			require.NoError(t, source.VerifySlot(t.Context(), newTestLogger(t), slotName))

			var slotCount int

			err := server.connectAs(t, getAdminRole()).
				QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots WHERE slot_name = $1", slotName).
				Scan(&slotCount)
			require.NoError(t, err)
			assert.Equal(t, 1, slotCount)
		})

		t.Run("returns error when a logical slot has the same name", func(t *testing.T) {
			t.Parallel()
			slotName := newSlotName()
			server.createLogicalSlot(t, slotName)

			err := server.newSource(getAdminRole()).VerifySlot(t.Context(), newTestLogger(t), slotName)

			assert.ErrorContains(t, err, "expected physical")
		})
	})
}

func TestDropSlot(t *testing.T) {
	t.Parallel()

	t.Run("returns slot name error without a slot name", func(t *testing.T) {
		t.Parallel()

		err := NewSource(Settings{}).DropSlot(t.Context(), newTestLogger(t), "")

		require.ErrorIs(t, err, ErrSlotNameEmpty)
	})

	runOnEverySupportedVersion(t, func(t *testing.T, server testServer) {
		t.Helper()

		t.Run("drops the slot when it exists", func(t *testing.T) {
			t.Parallel()
			slotName := newSlotName()
			server.createReservedPhysicalSlot(t, slotName)

			err := server.newSource(getAdminRole()).DropSlot(t.Context(), newTestLogger(t), slotName)
			require.NoError(t, err)

			_, isSlotPresent := server.readSlotType(t, slotName)
			assert.False(t, isSlotPresent)
		})

		t.Run("succeeds when the slot is missing", func(t *testing.T) {
			t.Parallel()

			err := server.newSource(getAdminRole()).DropSlot(t.Context(), newTestLogger(t), newSlotName())

			assert.NoError(t, err)
		})

		t.Run("keeps a logical slot with the same name", func(t *testing.T) {
			t.Parallel()
			slotName := newSlotName()
			server.createLogicalSlot(t, slotName)

			err := server.newSource(getAdminRole()).DropSlot(t.Context(), newTestLogger(t), slotName)
			require.ErrorContains(t, err, "expected physical")

			_, isSlotPresent := server.readSlotType(t, slotName)
			assert.True(t, isSlotPresent)
		})

		t.Run("keeps the slot when a consumer streams from it", func(t *testing.T) {
			t.Parallel()
			slotName := newSlotName()
			server.createReservedPhysicalSlot(t, slotName)
			server.attachSlotConsumer(t, slotName)

			err := server.newSource(getAdminRole()).DropSlot(t.Context(), newTestLogger(t), slotName)
			require.ErrorContains(t, err, "is active")

			_, isSlotPresent := server.readSlotType(t, slotName)
			assert.True(t, isSlotPresent)
		})
	})
}

func TestDropSlotForRemoval(t *testing.T) {
	t.Parallel()

	t.Run("returns slot name error without a slot name", func(t *testing.T) {
		t.Parallel()

		err := NewSource(Settings{}).DropSlotForRemoval(t.Context(), newTestLogger(t), "")

		require.ErrorIs(t, err, ErrSlotNameEmpty)
	})

	runOnEverySupportedVersion(t, func(t *testing.T, server testServer) {
		t.Helper()

		t.Run("drops the slot when it is inactive", func(t *testing.T) {
			t.Parallel()
			slotName := newSlotName()
			server.createReservedPhysicalSlot(t, slotName)

			err := server.newSource(getAdminRole()).DropSlotForRemoval(t.Context(), newTestLogger(t), slotName)
			require.NoError(t, err)

			_, isSlotPresent := server.readSlotType(t, slotName)
			assert.False(t, isSlotPresent)
		})

		// This is what separates it from DropSlot: a deleted database must never leave WAL pinned,
		// so the consumer is evicted instead of the drop being refused.
		t.Run("terminates the consumer and drops the slot when a consumer streams from it", func(t *testing.T) {
			t.Parallel()
			slotName := newSlotName()
			server.createReservedPhysicalSlot(t, slotName)
			server.attachSlotConsumer(t, slotName)

			err := server.newSource(getAdminRole()).DropSlotForRemoval(t.Context(), newTestLogger(t), slotName)
			require.NoError(t, err)

			_, isSlotPresent := server.readSlotType(t, slotName)
			assert.False(t, isSlotPresent)
		})

		t.Run("succeeds when the slot is missing", func(t *testing.T) {
			t.Parallel()

			err := server.newSource(getAdminRole()).DropSlotForRemoval(t.Context(), newTestLogger(t), newSlotName())

			assert.NoError(t, err)
		})

		t.Run("keeps a logical slot with the same name", func(t *testing.T) {
			t.Parallel()
			slotName := newSlotName()
			server.createLogicalSlot(t, slotName)

			err := server.newSource(getAdminRole()).DropSlotForRemoval(t.Context(), newTestLogger(t), slotName)
			require.ErrorContains(t, err, "expected physical")

			_, isSlotPresent := server.readSlotType(t, slotName)
			assert.True(t, isSlotPresent)
		})
	})
}
