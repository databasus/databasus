package usecases_logical_postgresql

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	pgtypes "databasus-backend/internal/features/databases/databases/postgresql/logical"
)

func Test_BuildPgDumpArgs_TreatsDuplicateParameterAsLiteralDatabaseName(t *testing.T) {
	databaseName := "postgres dbname=databasus"
	postgresDatabase := &pgtypes.PostgresqlLogicalDatabase{
		Version:  pgshared.Version17,
		Host:     "localhost",
		Port:     5437,
		Username: "postgres",
		Database: &databaseName,
	}

	usecase := &CreatePostgresqlBackupUsecase{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	args := usecase.buildPgDumpArgs(postgresDatabase)

	databaseArgumentIndex := -1
	for argumentIndex, argument := range args {
		if argument == "-d" {
			databaseArgumentIndex = argumentIndex + 1
			break
		}
	}

	require.Less(t, databaseArgumentIndex, len(args))
	require.GreaterOrEqual(t, databaseArgumentIndex, 0)
	require.Equal(t, "dbname='postgres dbname=databasus'", args[databaseArgumentIndex])
}

func Test_BuildPgDumpArgs_WhenBothExclusionListsAreSet_AddsWholeTableThenDataFlagsNormalized(t *testing.T) {
	databaseName := "appdb"
	postgresDatabase := &pgtypes.PostgresqlLogicalDatabase{
		Version:          pgshared.Version17,
		Host:             "localhost",
		Port:             5437,
		Username:         "postgres",
		Database:         &databaseName,
		ExcludeTables:    []string{" audit_tmp", "audit_tmp\n"},
		ExcludeTableData: []string{"logs, history_*", "\nlogs", " "},
	}

	usecase := &CreatePostgresqlBackupUsecase{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	args := usecase.buildPgDumpArgs(postgresDatabase)

	var exclusionArgs []string
	for _, argument := range args {
		if strings.HasPrefix(argument, "--exclude-table") {
			exclusionArgs = append(exclusionArgs, argument)
		}
	}

	require.Equal(t, []string{
		"--exclude-table=audit_tmp",
		"--exclude-table-data=logs",
		"--exclude-table-data=history_*",
	}, exclusionArgs)
}

// Test_Execute_WhenStorageTimesOut_ReportsStorageFailure_RegressionForIssue816 pins issue #816:
// a storage timeout that ended the backup context was reported as "backup cancelled".
func Test_Execute_WhenStorageTimesOut_ReportsStorageFailure_RegressionForIssue816(t *testing.T) {
	database := &databases.Database{
		ID:                uuid.New(),
		Type:              databases.DatabaseTypePostgresLogical,
		PostgresqlLogical: databases.GetTestPostgresConfig(),
	}

	backups_core_logical.AssertStorageTimeoutReportedAsStorageFailure(t, func(
		ctx context.Context,
		backup *backups_core_logical.LogicalBackup,
		backupConfig *backups_config_logical.LogicalBackupConfig,
		fileStore backups_core_logical.BackupFileStore,
	) error {
		_, err := GetCreatePostgresqlBackupUsecase().Execute(ctx, backup, backupConfig, database, fileStore, func(float64) {})

		return err
	})
}

func Test_Execute_WhenPgDumpFailsBeforeProducingData_ReportsPgDumpError(t *testing.T) {
	backups_core_logical.AssertDumpToolErrorReported(t, func(
		ctx context.Context,
		backup *backups_core_logical.LogicalBackup,
		backupConfig *backups_config_logical.LogicalBackupConfig,
		fileStore backups_core_logical.BackupFileStore,
		dumpToolBin string,
	) error {
		_, err := GetCreatePostgresqlBackupUsecase().streamToStorage(
			ctx, backup, backupConfig, dumpToolBin, nil, "", fileStore,
			&databases.Database{PostgresqlLogical: &pgtypes.PostgresqlLogicalDatabase{}},
			func(float64) {},
		)

		return err
	})
}
