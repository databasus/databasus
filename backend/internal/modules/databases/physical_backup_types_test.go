package databases_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/util/testing/containers"
)

func Test_GetPhysicalBackupTypes(t *testing.T) {
	t.Run("returns the backup type of each physical database and leaves out the other IDs", func(t *testing.T) {
		physicalServer := containers.StartPhysicalPostgres(t, "postgres:17")
		tm := newTeam(t)
		physicalDatabase := tm.createDatabaseNamed(t, "Cluster",
			fromPhysicalSettings(t, newPhysicalSettings(physicalServer, apiclient.PhysicalBackupTypeFULL)))
		logicalDatabaseID := tm.app.InsertDatabase(t, tm.workspace.Id)
		missingDatabaseID := uuid.New()

		backupTypes := tm.app.GetPhysicalBackupTypes(t,
			[]uuid.UUID{physicalDatabase.Id, logicalDatabaseID, missingDatabaseID})

		assert.Equal(t, map[uuid.UUID]databases.PhysicalBackupType{
			physicalDatabase.Id: databases.PhysicalBackupTypeFull,
		}, backupTypes)
	})
}
