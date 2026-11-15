package database_test

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/testdb"
)

func TestReadMigrationBundle(t *testing.T) {
	t.Parallel()

	t.Run("hashes length-prefixed names and contents in name order", func(t *testing.T) {
		t.Parallel()
		files := fstest.MapFS{
			"20260102000000_second.sql": {Data: []byte("SELECT 2;")},
			"20260101000000_first.sql":  {Data: []byte("SELECT 1;")},
			"README.md":                 {Data: []byte("not a migration")},
		}

		bundle, err := database.ReadMigrationBundle(files)

		require.NoError(t, err)
		expectedDigest := sha256.New()
		for _, file := range []struct{ name, content string }{
			{"20260101000000_first.sql", "SELECT 1;"},
			{"20260102000000_second.sql", "SELECT 2;"},
		} {
			_ = binary.Write(expectedDigest, binary.BigEndian, uint32(len(file.name)))
			expectedDigest.Write([]byte(file.name))
			_ = binary.Write(expectedDigest, binary.BigEndian, uint64(len(file.content)))
			expectedDigest.Write([]byte(file.content))
		}
		assert.Equal(t, hex.EncodeToString(expectedDigest.Sum(nil)), bundle.Digest)
		assert.Equal(t, []int64{20260101000000, 20260102000000}, bundle.Versions)
	})

	t.Run("changes the digest when a migration's content changes", func(t *testing.T) {
		t.Parallel()

		original, err := database.ReadMigrationBundle(fstest.MapFS{"20260101000000_a.sql": {Data: []byte("A")}})
		require.NoError(t, err)
		changed, err := database.ReadMigrationBundle(fstest.MapFS{"20260101000000_a.sql": {Data: []byte("B")}})
		require.NoError(t, err)

		assert.NotEqual(t, original.Digest, changed.Digest)
	})
}

func TestCheckCompatibility(t *testing.T) {
	t.Parallel()

	bundle := database.MigrationBundle{Digest: "bundle", Versions: []int64{20, 30}}

	for _, compatibilityCase := range []struct {
		name         string
		snapshot     database.SchemaIdentity
		isCompatible bool
	}{
		{"accepts the same bundle with every carried migration applied", identity("bundle", 20, 30), true},
		{"accepts an older retired migration the bundle no longer carries", identity("bundle", 10, 20, 30), true},
		{"refuses another bundle digest", identity("other", 20, 30), false},
		{"refuses a snapshot missing a carried migration", identity("bundle", 20), false},
		{"refuses a migration newer than the bundle", identity("bundle", 20, 30, 40), false},
	} {
		t.Run(compatibilityCase.name, func(t *testing.T) {
			t.Parallel()

			err := bundle.CheckCompatibility(compatibilityCase.snapshot)

			if compatibilityCase.isCompatible {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, database.ErrSchemaIncompatible)
			}
		})
	}
}

func TestReadAppliedMigrationVersions(t *testing.T) {
	t.Parallel()

	t.Run("keeps a version only when its newest history row says applied", func(t *testing.T) {
		t.Parallel()
		db, err := database.Open(t.Context(), testdb.OpenEmpty(t))
		require.NoError(t, err)
		t.Cleanup(func() { closePool(db) })
		require.NoError(t, db.Exec(`CREATE TABLE goose_db_version (
			id SERIAL PRIMARY KEY, version_id BIGINT NOT NULL, is_applied BOOLEAN NOT NULL, tstamp TIMESTAMP DEFAULT now()
		)`).Error)
		require.NoError(t, db.Exec(`INSERT INTO goose_db_version (version_id, is_applied) VALUES
			(0, true), (10, true), (20, true), (20, false), (30, true), (40, false), (40, true)`).Error)

		appliedVersions, err := database.ReadAppliedMigrationVersions(t.Context(), db)

		require.NoError(t, err)
		assert.Equal(t, []int64{10, 30, 40}, appliedVersions)
	})
}

func TestWithExportedSnapshot(t *testing.T) {
	t.Parallel()

	t.Run("exports a snapshot with the migrations applied in it", func(t *testing.T) {
		t.Parallel()
		db, err := database.Open(t.Context(), testdb.Open(t))
		require.NoError(t, err)
		t.Cleanup(func() { closePool(db) })

		var exported database.ExportedSnapshot
		err = database.WithExportedSnapshot(t.Context(), db, func(snapshot database.ExportedSnapshot) error {
			exported = snapshot

			return nil
		})

		require.NoError(t, err)
		assert.NotEmpty(t, exported.ID)
		assert.NotEmpty(t, exported.AppliedMigrationVersions)
	})
}

func identity(digest string, versions ...int64) database.SchemaIdentity {
	return database.SchemaIdentity{MigrationBundleDigest: digest, AppliedMigrationVersions: versions}
}

func closePool(db *gorm.DB) {
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}
