package physical

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

func getReplicationResultRow(results []*pgconn.Result, columns int) ([][]byte, error) {
	if len(results) != 1 || results[0] == nil || len(results[0].Rows) != 1 || len(results[0].Rows[0]) != columns {
		return nil, fmt.Errorf("%w: unexpected replication response shape", ErrInvalidTimelineHistory)
	}

	for _, value := range results[0].Rows[0][:columns-1] {
		if len(value) == 0 {
			return nil, fmt.Errorf("%w: empty replication response field", ErrInvalidTimelineHistory)
		}
	}

	return results[0].Rows[0], nil
}
