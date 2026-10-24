package usecases_physical_postgresql

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/storages"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/walmath"
)

// setupMtlsFixture boots a throwaway replication-capable mTLS PG 17 source and wires a WAL-stream
// fixture against it. The server image is built from the committed testdata/mtls dir (certs +
// replication-aware pg_hba.conf), since postgres rejects a group/world-readable key.
func setupMtlsFixture(t *testing.T) *PhysicalDBFixture {
	t.Helper()

	source := containers.StartPhysicalPostgresMtls(t, databases.GetPhysicalMtlsTestdataDir())

	return SetupPhysicalDBForStreamMtls(t, source.Host, source.Port)
}

// Test_PhysicalMtls_ReplicationConnectionAndSlot_Succeeds proves the source
// cluster is reachable for replication over client-cert TLS: PopulateDbData runs
// IDENTIFY-style queries and VerifyWalSlot creates a physical slot — both require
// a working `replication=database` mTLS handshake against the
// test-physical-postgres-17-mtls container's `hostssl replication ... cert` rule.
func Test_PhysicalMtls_ReplicationConnectionAndSlot_Succeeds(t *testing.T) {
	fixture := setupMtlsFixture(t)

	conn := OpenAdminConn(t, fixture)

	var one int
	require.NoError(t, conn.QueryRow(context.Background(), "SELECT 1").Scan(&one),
		"a basic query over the mTLS inspection connection must succeed")

	logg := logger.GetLogger()
	require.NoError(t,
		fixture.DB.PostgresqlPhysical.VerifyWalSlot(context.Background(), logg, encryption.GetFieldEncryptor()),
		"creating the persistent replication slot must work over mTLS")

	require.True(t, SlotExists(t, conn, fixture.DB.PostgresqlPhysical.ReplicationSlotName),
		"the persistent slot must exist after VerifyWalSlot over mTLS")

	t.Cleanup(func() {
		_ = fixture.DB.PostgresqlPhysical.DropWalSlot(context.Background(), logg, encryption.GetFieldEncryptor())
	})
}

// Test_PhysicalMtls_MissingClientCert_Rejected confirms the cluster enforces
// client-cert auth: the same DB with the client cert stripped cannot open a
// replication-grade connection.
func Test_PhysicalMtls_MissingClientCert_Rejected(t *testing.T) {
	fixture := setupMtlsFixture(t)

	noCert := *fixture.DB.PostgresqlPhysical
	noCert.SslClientCert = ""
	noCert.SslClientKey = ""

	_, err := noCert.OpenInspectionConn(context.Background(), encryption.GetFieldEncryptor())
	require.Error(t, err, "a connection without the client certificate must be rejected by the mTLS cluster")
}

// Test_FullOverMtls_ProducesArtifactAndManifest runs pg_basebackup end-to-end
// over mTLS — the same cred/PGSSL env path the streamer uses — and asserts the
// artifact + reconstructed manifest land in storage.
func Test_FullOverMtls_ProducesArtifactAndManifest(t *testing.T) {
	fixture := setupMtlsFixture(t)

	uc := NewCreateFullBackupUsecase()

	fullRow, err := physical_repositories.GetFullBackupRepository().FindByID(fixture.BackupID)
	require.NoError(t, err)
	require.NotNil(t, fullRow)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	result, err := uc.Execute(ctx, FullBackupSpec{
		CommonBackupSpec: CommonBackupSpec{
			SourceDB:       fixture.DB.PostgresqlPhysical,
			DatabaseName:   fixture.DB.Name,
			StorageID:      fixture.Storage.ID,
			Storage:        fixture.Storage,
			FileStore:      storages.GetStorageFileStore(),
			Encryption:     fullRow.Encryption,
			FieldEncryptor: encryption.GetFieldEncryptor(),
			FullRepo:       physical_repositories.GetFullBackupRepository(),
			HistoryRepo:    physical_repositories.GetWalHistoryRepository(),
			Logger:         logger.GetLogger(),
		},
		Backup: fullRow,
	})
	require.NoError(t, err)
	require.Equal(
		t,
		"COMPLETED",
		string(result.Status),
		"FULL over mTLS must complete; message=%s",
		result.ErrorMessage,
	)

	require.NotEmpty(t, result.FileName)
	artifact, err := fixture.Storage.GetFile(
		t.Context(),
		encryption.GetFieldEncryptor(),
		logger.GetLogger(),
		result.FileName,
	)
	require.NoError(t, err, "artifact must be in storage")
	require.NoError(t, artifact.Close())

	require.NotEmpty(t, result.ManifestFileName)
	manifest, err := fixture.Storage.GetFile(
		t.Context(),
		encryption.GetFieldEncryptor(),
		logger.GetLogger(),
		result.ManifestFileName,
	)
	require.NoError(t, err, "reconstructed manifest must be in storage")
	require.NoError(t, manifest.Close())
}

// Test_WalStreamOverMtls_StreamerArchivesSegments runs the full streamer
// (pg_receivewal over client-cert TLS) and asserts rotated segments are archived
// through the insert-first uploader.
func Test_WalStreamOverMtls_StreamerArchivesSegments(t *testing.T) {
	fixture := setupMtlsFixture(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixture.DB.ID)
	})

	store := newMockWalStorage()

	stop := StartWalStreamerForTest(
		t,
		WalStreamerTestSpec{Fixture: fixture, Storage: store, WatchDirRoot: t.TempDir()},
	).Stop
	t.Cleanup(stop)

	adminConn := OpenAdminConn(t, fixture)

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	for range 3 {
		_, err := ForceWalRotation(ctx, adminConn)
		require.NoError(t, err)
	}

	WaitForCommittedWalSegmentCount(t, fixture.DB.ID, 1, 90*time.Second)

	segments, err := physical_repositories.GetWalSegmentRepository().FindByChainSpan(
		fixture.DB.ID, 1, walmath.LSN(0), lsnSpanUpperBoundForTests,
	)
	require.NoError(t, err)

	var committed int
	for _, seg := range segments {
		if seg.FileName == nil {
			continue
		}

		committed++

		require.True(t, store.hasObject(*seg.FileName), "archived segment must exist in storage over mTLS")
	}

	require.GreaterOrEqual(t, committed, 1, "at least one segment must be archived over mTLS")
}
