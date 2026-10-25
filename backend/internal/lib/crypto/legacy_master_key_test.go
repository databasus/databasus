package crypto_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/testdb"
)

const legacyMasterKey = "fedcba9876543210fedcba9876543210-legacy"

func TestMoveLegacyMasterKeyToFile(t *testing.T) {
	t.Parallel()

	t.Run("writes the legacy key into a new key file and removes it from the table", func(t *testing.T) {
		t.Parallel()
		db := openMigratedDatabase(t)
		insertLegacyKeys(t, db, legacyMasterKey)
		masterKeyPath := filepath.Join(t.TempDir(), "data", "secret.key")

		err := crypto.MoveLegacyMasterKeyToFile(t.Context(), newDeps(db, masterKeyPath))

		require.NoError(t, err)
		assertFileContent(t, masterKeyPath, legacyMasterKey)
		assert.Empty(t, listLegacyKeys(t, db))
		openedKey, err := crypto.OpenOrCreateMasterKey(masterKeyPath)
		require.NoError(t, err)
		assert.Equal(t, legacyMasterKey, openedKey.GetSecret())
	})

	t.Run("replaces an existing key file with the legacy key", func(t *testing.T) {
		t.Parallel()
		db := openMigratedDatabase(t)
		insertLegacyKeys(t, db, legacyMasterKey)
		masterKeyPath := writeMasterKeyFile(t, storedMasterKey)

		err := crypto.MoveLegacyMasterKeyToFile(t.Context(), newDeps(db, masterKeyPath))

		require.NoError(t, err)
		assertFileContent(t, masterKeyPath, legacyMasterKey)
		assert.Empty(t, listLegacyKeys(t, db))
		keyDirectoryEntries, err := os.ReadDir(filepath.Dir(masterKeyPath))
		require.NoError(t, err)
		assert.Len(t, keyDirectoryEntries, 1)
	})

	t.Run("leaves the key file alone when the table holds no key", func(t *testing.T) {
		t.Parallel()
		db := openMigratedDatabase(t)
		masterKeyPath := writeMasterKeyFile(t, storedMasterKey)

		err := crypto.MoveLegacyMasterKeyToFile(t.Context(), newDeps(db, masterKeyPath))

		require.NoError(t, err)
		assertFileContent(t, masterKeyPath, storedMasterKey)
	})

	t.Run("leaves a row with an empty secret and the key file alone", func(t *testing.T) {
		t.Parallel()
		db := openMigratedDatabase(t)
		insertLegacyKeys(t, db, "")
		masterKeyPath := writeMasterKeyFile(t, storedMasterKey)

		err := crypto.MoveLegacyMasterKeyToFile(t.Context(), newDeps(db, masterKeyPath))

		require.NoError(t, err)
		assertFileContent(t, masterKeyPath, storedMasterKey)
		assert.Equal(t, []string{""}, listLegacyKeys(t, db))
	})

	t.Run("fails with the file's path and keeps the key in the table when the key file cannot be written",
		func(t *testing.T) {
			t.Parallel()
			skipUnderRoot(t)
			db := openMigratedDatabase(t)
			insertLegacyKeys(t, db, legacyMasterKey)
			masterKeyPath := filepath.Join(newReadOnlyDirectory(t), "secret.key")

			err := crypto.MoveLegacyMasterKeyToFile(t.Context(), newDeps(db, masterKeyPath))

			require.Error(t, err)
			assert.Contains(t, err.Error(), masterKeyPath)
			assert.Equal(t, []string{legacyMasterKey}, listLegacyKeys(t, db))
			assert.NoFileExists(t, masterKeyPath)
		})
}

func newDeps(db *gorm.DB, masterKeyPath string) crypto.Deps {
	return crypto.Deps{
		Transactor:          database.NewTransactor(db),
		LegacyKeyRepository: crypto.NewLegacyMasterKeyRepository(db),
		MasterKeyPath:       masterKeyPath,
		Logger:              slog.New(slog.DiscardHandler),
	}
}

func openMigratedDatabase(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := database.Open(t.Context(), testdb.Open(t))
	require.NoError(t, err)

	t.Cleanup(func() {
		sqlDB, sqlErr := db.DB()
		if sqlErr == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func insertLegacyKeys(t *testing.T, db *gorm.DB, secrets ...string) {
	t.Helper()

	for _, secret := range secrets {
		require.NoError(t, db.WithContext(t.Context()).
			Exec("INSERT INTO lib_crypto_secret_keys (secret) VALUES (?)", secret).Error)
	}
}

func listLegacyKeys(t *testing.T, db *gorm.DB) []string {
	t.Helper()

	var secrets []string
	require.NoError(t, db.WithContext(t.Context()).
		Raw("SELECT secret FROM lib_crypto_secret_keys ORDER BY secret").Scan(&secrets).Error)

	return secrets
}
