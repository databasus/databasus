package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/migrations"
)

const healthcheckMigrationName = "20261029175054_healthcheck_module.sql"

func TestHealthcheckMigration(t *testing.T) {
	t.Parallel()
	t.Run("repairs invalid rows and fills missing configs while preserving valid values", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, healthcheckMigrationName)
		workspaceID := insertWorkspace(t, db)
		repairedID := insertMigratedDatabase(t, db, &workspaceID)
		missingID := insertMigratedDatabase(t, db, &workspaceID)
		validID := insertMigratedDatabase(t, db, &workspaceID)
		execute(
			t,
			db,
			`INSERT INTO healthcheck_configs (database_id,is_healthcheck_enabled,is_sent_notification_when_unavailable,interval_minutes,attempts_before_considered_as_down,store_attempts_days) VALUES (?,false,false,0,-2,0),(?,false,false,7,4,11)`,
			repairedID,
			validID,
		)
		execute(
			t,
			db,
			`INSERT INTO healthcheck_attempts (database_id,status,created_at) VALUES (?,'UNKNOWN','2026-01-01'),(?,'AVAILABLE','2026-01-02')`,
			repairedID,
			validID,
		)
		execute(t, db, "ALTER TABLE healthcheck_configs DROP CONSTRAINT fk_healthcheck_configs_database_id")
		execute(t, db, "ALTER TABLE healthcheck_attempts DROP CONSTRAINT fk_healthcheck_attempts_database_id")
		orphanID := uuid.New()
		execute(t, db, `INSERT INTO healthcheck_configs VALUES (?,true,true,1,3,7)`, orphanID)
		execute(
			t,
			db,
			`INSERT INTO healthcheck_attempts (database_id,status,created_at) VALUES (?,'UNAVAILABLE','2026-01-01')`,
			orphanID,
		)
		execute(
			t,
			db,
			`ALTER TABLE healthcheck_configs ADD CONSTRAINT fk_healthcheck_configs_database_id FOREIGN KEY (database_id) REFERENCES databases(id) NOT VALID`,
		)
		execute(
			t,
			db,
			`ALTER TABLE healthcheck_attempts ADD CONSTRAINT fk_healthcheck_attempts_database_id FOREIGN KEY (database_id) REFERENCES databases(id) NOT VALID`,
		)
		applyAllMigrations(t, db, healthcheckMigrationName)
		assert.ElementsMatch(
			t,
			[]uuid.UUID{repairedID, missingID, validID},
			listIDs(t, db, "SELECT database_id FROM healthcheck_configs"),
		)
		assert.Equal(t, []uuid.UUID{validID}, listIDs(t, db, "SELECT database_id FROM healthcheck_attempts"))
		for _, id := range []uuid.UUID{repairedID, missingID} {
			assert.Equal(
				t,
				[]string{"1/3/7"},
				listStrings(
					t,
					db,
					`SELECT interval_minutes || '/' || failed_attempts_before_unavailable || '/' || attempt_retention_days FROM healthcheck_configs WHERE database_id = ?`,
					id,
				),
			)
		}
		assert.Equal(
			t,
			[]bool{true},
			listBools(
				t,
				db,
				`SELECT is_healthcheck_enabled AND is_notification_enabled FROM healthcheck_configs WHERE database_id = ?`,
				missingID,
			),
		)
		assert.Equal(
			t,
			[]bool{false},
			listBools(
				t,
				db,
				`SELECT is_healthcheck_enabled OR is_notification_enabled FROM healthcheck_configs WHERE database_id = ?`,
				repairedID,
			),
		)
		assert.Equal(
			t,
			[]string{"7/4/11"},
			listStrings(
				t,
				db,
				`SELECT interval_minutes || '/' || failed_attempts_before_unavailable || '/' || attempt_retention_days FROM healthcheck_configs WHERE database_id = ?`,
				validID,
			),
		)
		assert.Equal(
			t,
			[]bool{true},
			listBools(t, db, "SELECT to_regclass('idx_healthcheck_configs_database_id') IS NULL"),
		)
		for _, column := range []string{"interval_minutes", "failed_attempts_before_unavailable", "attempt_retention_days"} {
			err := db.WithContext(t.Context()).
				Exec("UPDATE healthcheck_configs SET "+column+" = 0 WHERE database_id = ?", validID).
				Error
			require.Error(t, err)
			assert.Contains(t, err.Error(), "ck_healthcheck_configs_"+column)
		}
		err := db.WithContext(t.Context()).Exec("DELETE FROM databases WHERE id = ?", missingID).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fk_healthcheck_configs_database_id")
		execute(t, db, "DELETE FROM healthcheck_configs WHERE database_id = ?", validID)
		err = db.WithContext(t.Context()).Exec("DELETE FROM databases WHERE id = ?", validID).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fk_healthcheck_attempts_database_id")
	})
	t.Run("restores column names and cascading keys when rolled back", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, healthcheckMigrationName)
		workspaceID := insertWorkspace(t, db)
		databaseID := insertMigratedDatabase(t, db, &workspaceID)
		applyAllMigrations(t, db, healthcheckMigrationName)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		provider, err := goose.NewProvider(
			goose.DialectPostgres,
			sqlDB,
			migrations.Files,
			goose.WithDisableGlobalRegistry(true),
		)
		require.NoError(t, err)
		_, err = provider.ApplyVersion(t.Context(), 20261029175054, false)
		require.NoError(t, err)
		execute(
			t,
			db,
			`UPDATE healthcheck_configs SET interval_minutes=0,attempts_before_considered_as_down=0,store_attempts_days=0,is_sent_notification_when_unavailable=false`,
		)
		execute(
			t,
			db,
			`INSERT INTO healthcheck_attempts (database_id,status,created_at) VALUES (?,'UNKNOWN','2026-01-01')`,
			databaseID,
		)
		execute(t, db, "DELETE FROM databases WHERE id = ?", databaseID)
		assert.Empty(t, listIDs(t, db, "SELECT database_id FROM healthcheck_configs"))
		assert.Empty(t, listIDs(t, db, "SELECT id FROM healthcheck_attempts"))
		assert.Equal(
			t,
			[]bool{true},
			listBools(t, db, "SELECT to_regclass('idx_healthcheck_configs_database_id') IS NOT NULL"),
		)
	})
}
