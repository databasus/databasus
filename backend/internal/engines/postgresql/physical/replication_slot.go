package physical

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

// pg_receivewal clears the active flag a moment after SIGTERM, so the removal polls instead of
// assuming the detach is instantaneous.
const slotDetachPollInterval = 250 * time.Millisecond

// VerifySlot ensures the physical replication slot of the settings exists on the cluster. A second
// call after the first create is a no-op. The backup runner calls it, because the slot is a backup
// precondition; the connection test does not, because it only probes settings, not state. It
// refuses to overwrite a same-named logical slot, which would clobber another tool's state.
func (s *Source) VerifySlot(ctx context.Context, logger *slog.Logger) error {
	slotName := s.settings.ReplicationSlotName
	if slotName == "" {
		return ErrSlotNameEmpty
	}

	conn, err := s.connect(ctx)
	if err != nil {
		return fmt.Errorf("open conn: %w", err)
	}
	defer closeConn(ctx, conn, logger)

	var slotType string

	err = conn.QueryRow(ctx, "SELECT slot_type FROM pg_replication_slots WHERE slot_name = $1", slotName).
		Scan(&slotType)

	switch {
	case err == nil:
		if slotType != "physical" {
			return fmt.Errorf(
				"replication slot %q exists with type %q (expected physical); resolve manually",
				slotName, slotType,
			)
		}

		logger.DebugContext(ctx, "replication slot already exists", slog.String("slot_name", slotName))

		return nil
	case errors.Is(err, pgx.ErrNoRows):
	default:
		return fmt.Errorf("query pg_replication_slots: %w", err)
	}

	if _, err := conn.Exec(ctx, "SELECT pg_create_physical_replication_slot($1, true)", slotName); err != nil {
		if classifyServerError(err) == serverErrorNameTaken {
			logger.DebugContext(
				ctx,
				"replication slot created by concurrent verify",
				slog.String("slot_name", slotName),
			)

			return nil
		}

		return fmt.Errorf("create physical replication slot: %w", err)
	}

	logger.InfoContext(ctx, "replication slot created", slog.String("slot_name", slotName))

	return nil
}

// DropSlot removes the physical replication slot from the cluster, and is a no-op when it is
// already gone. The caller stops any local pg_receivewal on this slot first: DropSlot refuses an
// active slot rather than terminate sessions of external tools that may share the name. It refuses
// a same-named logical slot too.
func (s *Source) DropSlot(ctx context.Context, logger *slog.Logger) error {
	slotName := s.settings.ReplicationSlotName
	if slotName == "" {
		return ErrSlotNameEmpty
	}

	conn, err := s.connect(ctx)
	if err != nil {
		return fmt.Errorf("open conn: %w", err)
	}
	defer closeConn(ctx, conn, logger)

	slot, isSlotPresent, err := readSlot(ctx, conn, slotName)
	if err != nil {
		return err
	}

	if !isSlotPresent {
		logger.DebugContext(ctx, "replication slot already absent", slog.String("slot_name", slotName))

		return nil
	}

	if slot.Type != "physical" {
		return fmt.Errorf(
			"replication slot %q exists with type %q (expected physical); refusing to drop",
			slotName, slot.Type,
		)
	}

	if slot.IsActive {
		return fmt.Errorf(
			"replication slot %q is active; stop the consumer (pg_receivewal) before deletion",
			slotName,
		)
	}

	return dropSlot(ctx, conn, logger, slotName)
}

// DropSlotForRemoval drops the slot when its database is being removed. Unlike DropSlot it does
// not refuse an active slot: the local pg_receivewal is torn down concurrently by the removal, so
// the consumer may still be attached or mid-detach. It terminates any remaining consumer of this
// slot, which is safe because the slot name (databasus_slot_<uuid>) belongs to this database alone,
// waits for the slot to go inactive and drops it, so a deleted database never leaves a slot pinning
// WAL on the source. The caller bounds it with a deadline on ctx.
func (s *Source) DropSlotForRemoval(ctx context.Context, logger *slog.Logger) error {
	slotName := s.settings.ReplicationSlotName
	if slotName == "" {
		return ErrSlotNameEmpty
	}

	conn, err := s.connect(ctx)
	if err != nil {
		return fmt.Errorf("open conn: %w", err)
	}
	defer closeConn(ctx, conn, logger)

	isSlotPresent, err := waitForSlotDetach(ctx, conn, logger, slotName)
	if err != nil {
		return err
	}

	if !isSlotPresent {
		logger.DebugContext(ctx, "replication slot already absent", slog.String("slot_name", slotName))

		return nil
	}

	return dropSlot(ctx, conn, logger, slotName)
}

type replicationSlot struct {
	Type      string
	IsActive  bool
	ActivePID *int
}

func readSlot(ctx context.Context, conn *pgx.Conn, slotName string) (replicationSlot, bool, error) {
	var slot replicationSlot

	err := conn.QueryRow(
		ctx,
		"SELECT slot_type, active, active_pid FROM pg_replication_slots WHERE slot_name = $1",
		slotName,
	).Scan(&slot.Type, &slot.IsActive, &slot.ActivePID)
	if errors.Is(err, pgx.ErrNoRows) {
		return replicationSlot{}, false, nil
	}

	if err != nil {
		return replicationSlot{}, false, fmt.Errorf("query pg_replication_slots: %w", err)
	}

	return slot, true, nil
}

// waitForSlotDetach terminates the consumer of an active slot until the slot goes inactive, and
// reports whether the slot still exists.
func waitForSlotDetach(ctx context.Context, conn *pgx.Conn, logger *slog.Logger, slotName string) (bool, error) {
	for {
		slot, isSlotPresent, err := readSlot(ctx, conn, slotName)
		if err != nil || !isSlotPresent {
			return isSlotPresent, err
		}

		if slot.Type != "physical" {
			return true, fmt.Errorf(
				"replication slot %q exists with type %q (expected physical); refusing to drop",
				slotName, slot.Type,
			)
		}

		if !slot.IsActive {
			return true, nil
		}

		if slot.ActivePID != nil {
			if _, err := conn.Exec(ctx, "SELECT pg_terminate_backend($1)", *slot.ActivePID); err != nil {
				logger.WarnContext(ctx, "failed to terminate replication slot consumer",
					slog.String("slot_name", slotName),
					slog.Int("active_pid", *slot.ActivePID),
					slog.Any("error", err),
				)
			}
		}

		select {
		case <-ctx.Done():
			return true, fmt.Errorf("waiting for replication slot %q to detach: %w", slotName, ctx.Err())
		case <-time.After(slotDetachPollInterval):
		}
	}
}

func dropSlot(ctx context.Context, conn *pgx.Conn, logger *slog.Logger, slotName string) error {
	if _, err := conn.Exec(ctx, "SELECT pg_drop_replication_slot($1)", slotName); err != nil {
		if classifyServerError(err) == serverErrorObjectMissing {
			logger.DebugContext(
				ctx,
				"replication slot dropped by concurrent caller",
				slog.String("slot_name", slotName),
			)

			return nil
		}

		return fmt.Errorf("drop replication slot: %w", err)
	}

	logger.InfoContext(ctx, "replication slot dropped", slog.String("slot_name", slotName))

	return nil
}
