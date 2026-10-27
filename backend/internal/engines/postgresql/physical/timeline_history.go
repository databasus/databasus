package physical

import (
	"context"
	"fmt"
)

func (i *Inspection) GetTimelineHistory(ctx context.Context, historyFilename string) ([]byte, error) {
	var body []byte

	if err := i.conn.QueryRow(ctx, `SELECT pg_read_binary_file('pg_wal/' || $1)`, historyFilename).
		Scan(&body); err != nil {
		return nil, fmt.Errorf("read history file %q from cluster: %w", historyFilename, err)
	}

	return body, nil
}
