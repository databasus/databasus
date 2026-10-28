package backuping_physical

import (
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/modules/backups"
)

func TestRetentionConfiguration(t *testing.T) {
	t.Run("does no catalog work when chain retention has a zero count", func(t *testing.T) {
		cleaner := &PhysicalBackupCleaner{}
		err := cleaner.cleanByChains(
			t.Context(),
			slog.New(slog.DiscardHandler),
			backups.PhysicalBackupConfig{Retention: backups.PhysicalRetention{ChainsCount: 0}},
		)
		require.NoError(t, err)
	})
	t.Run("returns no retention selection when a last N policy has a zero count", func(t *testing.T) {
		cleaner := &PhysicalBackupCleaner{}
		fullBackups := []*physical_models.PhysicalFullBackup{{ID: uuid.New()}, {ID: uuid.New()}}
		retained := cleaner.getRetainedTiersByFullBackupID(
			backups.PhysicalBackupConfig{
				Retention: backups.PhysicalRetention{
					FullBackups: backups.FullBackupsRetention{
						Policy: backups.FullBackupsRetentionPolicyLastN,
						Count:  0,
					},
				},
			},
			fullBackups,
		)
		assert.Nil(t, retained)
	})
}
