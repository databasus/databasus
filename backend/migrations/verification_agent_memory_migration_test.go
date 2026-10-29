package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/migrations"
)

func TestVerificationAgentMemoryMigration(t *testing.T) {
	t.Parallel()
	t.Run("preserves stored memory in megabytes and restores gigabytes when rolled back", func(t *testing.T) {
		t.Parallel()
		const migrationName = "20261029160000_verification_agent_memory_megabytes.sql"
		db := openDatabaseMigratedBefore(t, migrationName)
		agentID := uuid.New()
		execute(
			t,
			db,
			"INSERT INTO verification_agents (id, name, token_hash, max_ram_gb) VALUES (?, 'agent', ?, 2)",
			agentID,
			agentID.String(),
		)
		largeAgentID := uuid.New()
		execute(
			t,
			db,
			"INSERT INTO verification_agents (id, name, token_hash, max_ram_gb) VALUES (?, 'large agent', ?, 2147483647)",
			largeAgentID,
			largeAgentID.String(),
		)

		applyAllMigrations(t, db, migrationName)
		var memory int
		require.NoError(
			t,
			db.WithContext(t.Context()).
				Raw("SELECT max_ram_mb FROM verification_agents WHERE id = ?", agentID).
				Scan(&memory).
				Error,
		)
		assert.Equal(t, 2048, memory)
		var largeMemory int64
		require.NoError(
			t,
			db.WithContext(t.Context()).
				Raw("SELECT max_ram_mb FROM verification_agents WHERE id = ?", largeAgentID).
				Scan(&largeMemory).
				Error,
		)
		assert.Equal(t, int64(2147483647)*1024, largeMemory)

		err := db.WithContext(t.Context()).
			Exec("UPDATE verification_agents SET max_ram_mb = -1 WHERE id = ?", agentID).
			Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ck_verification_agents_max_ram_mb")
		sqlDB, err := db.DB()
		require.NoError(t, err)
		provider, err := goose.NewProvider(
			goose.DialectPostgres,
			sqlDB,
			migrations.Files,
			goose.WithDisableGlobalRegistry(true),
		)
		require.NoError(t, err)
		_, err = provider.ApplyVersion(t.Context(), 20261029160000, false)
		require.NoError(t, err)
		require.NoError(
			t,
			db.WithContext(t.Context()).
				Raw("SELECT max_ram_gb FROM verification_agents WHERE id = ?", agentID).
				Scan(&memory).
				Error,
		)
		assert.Equal(t, 2, memory)
	})
}
