package physicaltest

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

func RunCNPGCopiedAuthentication(t *testing.T, spec CNPGManagerSpec) {
	t.Helper()

	recovery := prepareCNPGIndependentRecovery(t, CNPGIndependentSpec{Manager: spec, Mode: CNPGRecoveryBackup})
	sourceConfiguration := recovery.Source.getServingConfiguration(t)

	restored := recovery.boot(t)

	var hbaPath string
	var hbaHash string
	require.NoError(
		t,
		restored.QueryRow(t.Context(), "SELECT current_setting('hba_file'), md5(pg_read_file(current_setting('hba_file')))").
			Scan(&hbaPath, &hbaHash),
	)
	require.Equal(t, restoredClusterDir(recovery.Source.Image)+"/pg_hba.conf", hbaPath)
	require.Equal(t, sourceConfiguration.HBAHash, hbaHash)

	config, err := pgconn.ParseConfig(fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable connect_timeout=5",
		recovery.Target.Host(),
		recovery.Target.MappedPort(),
		containers.CNPGBackupRole,
		containers.PostgresPassword,
		restoredPgDatabase,
	))
	require.NoError(t, err)

	rejected, err := pgconn.ConnectConfig(t.Context(), config)
	if rejected != nil {
		_ = rejected.Close(context.WithoutCancel(t.Context()))
	}

	var serverError *pgconn.PgError
	require.ErrorAs(t, err, &serverError)
	require.Equal(t, "28000", serverError.Code)
	require.Contains(t, serverError.Message, "no encryption")
}
