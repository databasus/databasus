package retention

import (
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

func TestRetentionConfiguration(t *testing.T) {
	t.Run("does no catalog work when chain retention has a zero count", func(t *testing.T) {
		cleaner := &Cleaner{}
		err := cleaner.cleanByChains(
			t.Context(),
			slog.New(slog.DiscardHandler),
			config.PhysicalBackupConfig{Retention: config.PhysicalRetention{ChainsCount: 0}},
		)
		require.NoError(t, err)
	})
	t.Run("returns no retention selection when a last N policy has a zero count", func(t *testing.T) {
		cleaner := &Cleaner{}
		fullBackups := []*catalog.Backup{{ID: uuid.New()}, {ID: uuid.New()}}
		retained := cleaner.getRetainedTiersByFullBackupID(
			config.PhysicalBackupConfig{
				Retention: config.PhysicalRetention{
					FullBackups: config.FullBackupsRetention{
						Policy: config.FullBackupsRetentionPolicyLastN,
						Count:  0,
					},
				},
			},
			fullBackups,
		)
		assert.Nil(t, retained)
	})
}
