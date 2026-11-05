package walstreaming

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/modules/databases"
)

func TestBuildChainRiskAlert(t *testing.T) {
	t.Run("uses separate incident kinds when archive risks differ", func(t *testing.T) {
		database := databases.Database{ID: uuid.New(), Name: "orders"}
		lastArchivedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		retentionAlert := buildChainRiskAlert(database, ChainRiskReport{
			Reason:        ChainRiskReasonSlotRetention,
			SlotWalStatus: "extended",
			LagBytes:      64 * 1024 * 1024,
		})
		staleAlert := buildChainRiskAlert(database, ChainRiskReport{
			Reason:            ChainRiskReasonArchiveStale,
			LastArchivedWalAt: &lastArchivedAt,
			LagBytes:          16 * 1024 * 1024,
		})
		rotationAlert := buildChainRiskAlert(
			database,
			ChainRiskReport{Reason: ChainRiskReasonRotationDenied},
		)

		assert.NotEqual(t, retentionAlert.Kind, staleAlert.Kind)
		assert.NotEqual(t, retentionAlert.Kind, rotationAlert.Kind)
		assert.NotEqual(t, staleAlert.Kind, rotationAlert.Kind)
		assert.Contains(t, staleAlert.Message, lastArchivedAt.Format(time.RFC3339))
		assert.Contains(t, rotationAlert.Message, "pg_switch_wal()")
	})
}
