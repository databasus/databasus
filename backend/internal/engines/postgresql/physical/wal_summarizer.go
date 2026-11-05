package physical

import (
	"context"
	"fmt"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

type WalSummarizerState struct {
	IsEnabled      bool
	OldestStartLSN *walmath.LSN
	IsRunning      bool
}

func (i *Inspection) GetWalSummarizerState(ctx context.Context) (WalSummarizerState, error) {
	var state WalSummarizerState
	err := i.conn.QueryRow(ctx, `SELECT
  COALESCE((SELECT setting = 'on' FROM pg_settings WHERE name = 'summarize_wal'), false),
  (SELECT MIN(start_lsn) FROM pg_available_wal_summaries()),
  (SELECT summarizer_pid IS NOT NULL FROM pg_get_wal_summarizer_state())
 `).Scan(&state.IsEnabled, &state.OldestStartLSN, &state.IsRunning)
	if err != nil {
		return WalSummarizerState{}, fmt.Errorf("read WAL summarizer state: %w", err)
	}

	return state, nil
}
