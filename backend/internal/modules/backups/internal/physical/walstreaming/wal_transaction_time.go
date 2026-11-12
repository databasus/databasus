package walstreaming

import (
	"context"
	"log/slog"
	"time"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

func (u *WalUploader) getWalTransactionTime(ctx context.Context, localPath string, endLSN walmath.LSN) *time.Time {
	transactionAt, err := pgphysical.GetWalTransactionTime(ctx, pgphysical.WalTransactionTimeSpec{
		ClientTools: u.deps.ClientTools, Version: u.deps.Version, LocalPath: localPath, EndLSN: endLSN,
	})
	if err != nil {
		// A record can continue in the next segment. Preserve the WAL bytes without
		// extending PITR coverage beyond the complete records already decoded.
		u.deps.Logger.WarnContext(
			ctx,
			"wal transaction time is unavailable",
			slog.String("end_lsn", endLSN.String()),
			slog.Any("error", err),
		)
	}

	return transactionAt
}
