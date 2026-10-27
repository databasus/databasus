package physical

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

type Slot struct {
	SlotName        string
	Type            string
	IsActive        bool
	ActivePID       *int
	ApplicationName string
	WalStatus       string
	RestartLSN      walmath.LSN
	FlushLSN        walmath.LSN
	// Position and allowance are absent during recovery, where pg_current_wal_lsn is unavailable.
	CurrentLSN *walmath.LSN
	LagBytes   *int64
	// Nil also means unlimited retention or a lost slot; WalStatus distinguishes them.
	SafeWalSizeBytes *int64
}

func (i *Inspection) GetSlot(ctx context.Context, slotName string) (*Slot, error) {
	slot := &Slot{SlotName: slotName}
	err := i.conn.QueryRow(ctx, `
  SELECT s.slot_type, s.active, s.active_pid,
   COALESCE(r.application_name, ''), COALESCE(s.wal_status, ''),
   COALESCE(s.restart_lsn, '0/0'::pg_lsn), COALESCE(s.confirmed_flush_lsn, '0/0'::pg_lsn),
   CASE WHEN pg_is_in_recovery() THEN NULL
    ELSE COALESCE(pg_wal_lsn_diff(pg_current_wal_lsn(), s.restart_lsn), 0)::bigint END,
   CASE WHEN pg_is_in_recovery() THEN NULL ELSE pg_current_wal_lsn() END,
   CASE WHEN pg_is_in_recovery() THEN NULL ELSE s.safe_wal_size END
  FROM pg_replication_slots s
  LEFT JOIN pg_stat_replication r ON r.pid = s.active_pid
  WHERE s.slot_name = $1
 `, slotName).Scan(&slot.Type, &slot.IsActive, &slot.ActivePID,
		&slot.ApplicationName, &slot.WalStatus, &slot.RestartLSN, &slot.FlushLSN,
		&slot.LagBytes, &slot.CurrentLSN, &slot.SafeWalSizeBytes)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read replication slot: %w", err)
	}

	return slot, nil
}

func (i *Inspection) CreateSlot(ctx context.Context, slotName string) error {
	if _, err := i.conn.Exec(ctx, "SELECT pg_create_physical_replication_slot($1, true)", slotName); err != nil {
		if classifyServerError(err) == serverErrorNameTaken {
			return fmt.Errorf("%w: %w", ErrSlotNameTaken, err)
		}
		return fmt.Errorf("create physical replication slot: %w", err)
	}

	return nil
}

// The per-backup slot is replaced even if a logical slot occupies its name.
func (i *Inspection) DropBackupSlotIfExists(ctx context.Context, slotName string) error {
	_, err := i.conn.Exec(ctx, `SELECT pg_drop_replication_slot(slot_name)
  FROM pg_replication_slots WHERE slot_name = $1`, slotName)
	if err != nil {
		return fmt.Errorf("drop backup slot: %w", err)
	}

	return nil
}

func (i *Inspection) TerminateBackend(ctx context.Context, pid int) (bool, error) {
	var isTerminated bool

	if err := i.conn.QueryRow(ctx, "SELECT pg_terminate_backend($1)", pid).Scan(&isTerminated); err != nil {
		return false, fmt.Errorf("terminate backend: %w", err)
	}

	return isTerminated, nil
}
