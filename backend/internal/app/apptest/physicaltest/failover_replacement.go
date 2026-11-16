package physicaltest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func (source restorableSource) waitForNewFull(t *testing.T, previous uuid.UUID) backupRow {
	t.Helper()

	var full backupRow
	var precedingCreatedAt time.Time
	source.Team.App.QuerySQL(t, &precedingCreatedAt, "SELECT created_at FROM backups_physical WHERE id=?", previous)

	testwait.Until(t, "a new full backup completes", func(context.Context) error {
		for _, backup := range source.Team.listBackups(t, "FULL") {
			if backup.CreatedAt.After(precedingCreatedAt) && backup.Status == "COMPLETED" {
				full = backup
				return nil
			}
		}

		return fmt.Errorf("no completed full newer than %s", previous)
	})

	return full
}

type fullReplacement struct {
	Preceding   uuid.UUID
	Replacement uuid.UUID
}

func (source restorableSource) requireSingleReplacement(t *testing.T, spec fullReplacement) {
	t.Helper()

	source.Team.App.RunJobPass(t, "backups_physical_scheduler")

	require.Equal(t, []uuid.UUID{spec.Replacement}, source.getFullsCreatedAfter(t, spec.Preceding))
}

func (source restorableSource) getFullsCreatedAfter(t *testing.T, precedingID uuid.UUID) []uuid.UUID {
	t.Helper()

	fulls := []uuid.UUID{}
	source.Team.App.QuerySQL(
		t,
		&fulls,
		"SELECT id FROM backups_physical WHERE database_id=? AND kind='FULL' AND created_at > (SELECT created_at FROM backups_physical WHERE id=?)",
		source.Team.Database.Id,
		precedingID,
	)

	return fulls
}

func (source restorableSource) waitForNewIncremental(t *testing.T, previous uuid.UUID) backupRow {
	t.Helper()

	var incremental backupRow
	var precedingCreatedAt time.Time
	source.Team.App.QuerySQL(t, &precedingCreatedAt, "SELECT created_at FROM backups_physical WHERE id=?", previous)

	testwait.Until(t, "a new incremental backup completes", func(context.Context) error {
		for _, backup := range source.Team.listBackups(t, "INCREMENTAL") {
			if !backup.CreatedAt.After(precedingCreatedAt) {
				continue
			}

			require.NotContains(t, []string{"ERROR", "CHAIN_BROKEN"}, backup.Status, "%+v", backup)
			if backup.Status == "COMPLETED" {
				incremental = backup
				return nil
			}
		}

		return fmt.Errorf("no completed incremental newer than %s", previous)
	})

	return incremental
}

func (source restorableSource) waitForCompletedFull(t *testing.T, backupID uuid.UUID) backupRow {
	t.Helper()

	var completed backupRow
	testwait.Until(t, "the identified scheduled full completes", func(context.Context) error {
		for _, backup := range source.Team.listBackups(t, "FULL") {
			if backup.ID != backupID {
				continue
			}

			require.NotEqual(t, "ERROR", backup.Status)
			if backup.Status == "COMPLETED" {
				completed = backup
				return nil
			}
		}

		return fmt.Errorf("full %s has not completed", backupID)
	})

	return completed
}
