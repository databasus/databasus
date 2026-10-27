package physical

import (
	"context"
	"fmt"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

func (i *Inspection) GetCurrentWalLSN(ctx context.Context) (walmath.LSN, error) {
	var currentLSNText string

	if err := i.conn.QueryRow(ctx, "SELECT pg_current_wal_lsn()::text").Scan(&currentLSNText); err != nil {
		return 0, fmt.Errorf("read current wal position: %w", err)
	}

	return walmath.ParseLSN(currentLSNText)
}

func (i *Inspection) SwitchWal(ctx context.Context) (walmath.LSN, error) {
	if _, err := i.conn.Exec(ctx, "SELECT pg_switch_wal()"); err != nil {
		if classifyServerError(err) == serverErrorPermissionDenied {
			return 0, fmt.Errorf("%w: %w", ErrWalSwitchDenied, err)
		}
		return 0, fmt.Errorf("pg_switch_wal: %w", err)
	}

	return i.GetCurrentWalLSN(ctx)
}
