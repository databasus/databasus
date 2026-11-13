package physicaltest

import (
	"context"
	"fmt"
	"testing"

	"databasus-backend/internal/lib/testwait"
)

func (team Team) hasArchivedSegment(t *testing.T, segment string) bool {
	t.Helper()

	var hasArchived bool
	team.App.QuerySQL(
		t,
		&hasArchived,
		"SELECT EXISTS(SELECT 1 FROM backups_physical_wal_segments WHERE database_id=? AND wal_filename=? AND file_name IS NOT NULL)",
		team.Database.Id,
		segment,
	)

	return hasArchived
}

func (team Team) waitForArchivedSegment(t *testing.T, segment string) {
	t.Helper()

	testwait.Until(t, "Databasus archives WAL segment "+segment, func(context.Context) error {
		if !team.hasArchivedSegment(t, segment) {
			return fmt.Errorf("segment %s is not archived", segment)
		}

		return nil
	})
}
