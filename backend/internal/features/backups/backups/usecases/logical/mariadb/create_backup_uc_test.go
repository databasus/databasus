package usecases_logical_mariadb

import (
	"context"
	"slices"
	"strings"
	"testing"

	mariadb_engine "databasus-backend/internal/engines/mariadb"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	databases_bridge "databasus-backend/internal/features/databases"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/testing/containers"
)

// One INSERT per row costs ~127x on restore and saves no memory: with --quick the
// dumper streams rows and caps a batched statement at net_buffer_length (~1 MB)
// however large the table is (issue #630).
func Test_BuildMariadbDumpArgs_ForAnyDatabase_NeverSkipsExtendedInsert(t *testing.T) {
	uc := &CreateMariadbBackupUsecase{}
	target := dumpTarget{
		connectionSettings: databases.MariadbConnectionSettings{
			Version:       mariadb_engine.Version1011,
			Engine:        mariadb_engine.Settings{Database: "oa_db"},
			ExcludeTables: []string{"personnel_real_time"},
		},
		settings: mariadb_engine.Settings{Database: "oa_db"},
	}

	dumpArgs := uc.buildMariadbDumpArgs(target)

	if slices.Contains(dumpArgs, "--skip-extended-insert") {
		t.Fatalf("mariadb-dump args must never contain --skip-extended-insert: %v", dumpArgs)
	}
}

func Test_BuildMariadbDumpArgs_WithExcludedTables_AddsQualifiedIgnoreTableArgs(t *testing.T) {
	uc := &CreateMariadbBackupUsecase{}
	target := dumpTarget{
		connectionSettings: databases.MariadbConnectionSettings{
			Version:       mariadb_engine.Version114,
			Engine:        mariadb_engine.Settings{Database: "oa_db"},
			ExcludeTables: []string{"personnel_access_control_event", "personnel_real_time"},
		},
		settings: mariadb_engine.Settings{Database: "oa_db"},
	}

	args := uc.buildMariadbDumpArgs(target)

	if !slices.Contains(args, "--ignore-table=oa_db.personnel_access_control_event") ||
		!slices.Contains(args, "--ignore-table=oa_db.personnel_real_time") {
		t.Fatalf("expected an --ignore-table arg per excluded table, got %v", args)
	}
}

func Test_BuildMariadbDumpArgs_WhenExcludedTablesArePastedMultiline_TrimsAndSplitsThem(t *testing.T) {
	uc := &CreateMariadbBackupUsecase{}
	target := dumpTarget{
		connectionSettings: databases.MariadbConnectionSettings{
			Version: mariadb_engine.Version114,
			Engine:  mariadb_engine.Settings{Database: "oa_db"},
			ExcludeTables: []string{
				"personnel_access_control_event",
				"\npersonnel_real_time",
				" ",
				"ext_alarm_message,\nmonitor_toxic_gas",
			},
		},
		settings: mariadb_engine.Settings{Database: "oa_db"},
	}

	args := uc.buildMariadbDumpArgs(target)

	ignoredTableArgs := []string{
		"--ignore-table=oa_db.personnel_access_control_event",
		"--ignore-table=oa_db.personnel_real_time",
		"--ignore-table=oa_db.ext_alarm_message",
		"--ignore-table=oa_db.monitor_toxic_gas",
	}
	for _, ignoredTableArg := range ignoredTableArgs {
		if !slices.Contains(args, ignoredTableArg) {
			t.Fatalf("expected %s, got %v", ignoredTableArg, args)
		}
	}

	if slices.Contains(args, "--ignore-table=oa_db.") {
		t.Fatalf("expected blank excluded tables to be dropped, got %v", args)
	}
}

// Dumping tablespace definitions reads INFORMATION_SCHEMA.FILES, which costs a global PROCESS
// privilege the connection test no longer requires, so the flag has to stay in every dump.
func Test_BuildMariadbDumpArgs_ForAnyDatabase_AlwaysSkipsTablespaces(t *testing.T) {
	uc := &CreateMariadbBackupUsecase{}
	target := dumpTarget{
		connectionSettings: databases.MariadbConnectionSettings{
			Version: mariadb_engine.Version1011,
			Engine:  mariadb_engine.Settings{Database: "oa_db"},
		},
		settings: mariadb_engine.Settings{Database: "oa_db"},
	}

	args := uc.buildMariadbDumpArgs(target)

	if !slices.Contains(args, "--no-tablespaces") {
		t.Fatalf("expected --no-tablespaces, got %v", args)
	}
}

func Test_BuildMariadbDumpArgs_WithTriggerPrivilege_KeepsTriggers(t *testing.T) {
	uc := &CreateMariadbBackupUsecase{}
	target := dumpTarget{
		connectionSettings: databases.MariadbConnectionSettings{
			Version:    mariadb_engine.Version1011,
			Engine:     mariadb_engine.Settings{Database: "oa_db"},
			Privileges: []string{"SELECT", "SHOW VIEW", "TRIGGER"},
		},
		settings: mariadb_engine.Settings{Database: "oa_db"},
	}

	args := uc.buildMariadbDumpArgs(target)

	if !slices.Contains(args, "--triggers") || slices.Contains(args, "--skip-triggers") {
		t.Fatalf("expected --triggers and no --skip-triggers, got %v", args)
	}
}

// Triggers are on by default, so without the explicit opt-out mariadb-dump fails on SHOW TRIGGERS
// for a role that lacks the privilege instead of dumping without them.
func Test_BuildMariadbDumpArgs_WithoutTriggerPrivilege_SkipsTriggers(t *testing.T) {
	uc := &CreateMariadbBackupUsecase{}
	target := dumpTarget{
		connectionSettings: databases.MariadbConnectionSettings{
			Version:    mariadb_engine.Version1011,
			Engine:     mariadb_engine.Settings{Database: "oa_db"},
			Privileges: []string{"SELECT", "SHOW VIEW"},
		},
		settings: mariadb_engine.Settings{Database: "oa_db"},
	}

	args := uc.buildMariadbDumpArgs(target)

	if !slices.Contains(args, "--skip-triggers") || slices.Contains(args, "--triggers") {
		t.Fatalf("expected --skip-triggers and no --triggers, got %v", args)
	}
}

func Test_BuildMariadbDumpArgs_WithoutExcludedTables_OmitsIgnoreTableArgs(t *testing.T) {
	uc := &CreateMariadbBackupUsecase{}
	target := dumpTarget{
		connectionSettings: databases.MariadbConnectionSettings{
			Version: mariadb_engine.Version114,
			Engine:  mariadb_engine.Settings{Database: "oa_db"},
		},
		settings: mariadb_engine.Settings{Database: "oa_db"},
	}

	args := uc.buildMariadbDumpArgs(target)

	for _, arg := range args {
		if strings.HasPrefix(arg, "--ignore-table=") {
			t.Fatalf("expected no --ignore-table args, got %v", args)
		}
	}
}

// Test_Execute_WhenStorageTimesOut_ReportsStorageFailure_RegressionForIssue816 pins issue #816 for
// MariaDB: a storage timeout that ended the backup context was reported as "backup cancelled".
func Test_Execute_WhenStorageTimesOut_ReportsStorageFailure_RegressionForIssue816(t *testing.T) {
	endpoint := containers.StartMariadb(t, "mariadb:10.11")
	database := createTestMariadbDatabase(t, endpoint)

	backups_core_logical.AssertStorageTimeoutReportedAsStorageFailure(t, func(
		ctx context.Context,
		backup *backups_core_logical.LogicalBackup,
		backupConfig *backups_config_logical.LogicalBackupConfig,
		fileStore backups_core_logical.BackupFileStore,
	) error {
		_, err := GetCreateMariadbBackupUsecase().Execute(ctx, backup, backupConfig, database, fileStore, func(float64) {})

		return err
	})
}

func Test_Execute_WhenMariadbDumpFailsBeforeProducingData_ReportsMariadbDumpError(t *testing.T) {
	backups_core_logical.AssertDumpToolErrorReported(t, func(
		ctx context.Context,
		backup *backups_core_logical.LogicalBackup,
		backupConfig *backups_config_logical.LogicalBackupConfig,
		fileStore backups_core_logical.BackupFileStore,
		dumpToolBin string,
	) error {
		_, err := GetCreateMariadbBackupUsecase().streamToStorage(
			ctx, backup, backupConfig, dumpToolBin, nil, fileStore, func(float64) {},
			dumpTarget{
				connectionSettings: databases.MariadbConnectionSettings{Version: mariadb_engine.Version1011},
				settings:           mariadb_engine.Settings{Database: "orders_db"},
			},
		)

		return err
	})
}

func createTestMariadbDatabase(t *testing.T, endpoint containers.Endpoint) databases.Database {
	t.Helper()

	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)
	database := databases_bridge.CreateTestMariadbDatabase(endpoint.Host, endpoint.Port, workspace.ID, nil)

	t.Cleanup(func() {
		databases_bridge.RemoveTestDatabase(t.Context(), database)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
	})

	return database
}
