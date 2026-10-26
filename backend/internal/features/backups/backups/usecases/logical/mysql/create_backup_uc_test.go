package usecases_logical_mysql

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	mysql_engine "databasus-backend/internal/engines/mysql"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	mysqltypes "databasus-backend/internal/features/databases/databases/mysql"
	"databasus-backend/internal/util/testing/containers"
)

// One INSERT per row costs ~127x on restore and saves no memory: with --quick the
// dumper streams rows and caps a batched statement at net_buffer_length (~1 MB)
// however large the table is (issue #630).
func Test_BuildMysqldumpArgs_ForAnyDatabase_NeverSkipsExtendedInsert(t *testing.T) {
	uc := &CreateMysqlBackupUsecase{}
	database := &mysqltypes.MysqlDatabase{
		Version:       mysql_engine.Version80,
		Database:      new("oa_db"),
		ExcludeTables: []string{"personnel_real_time"},
	}

	dumpArgs := uc.buildMysqldumpArgs(database)

	if slices.Contains(dumpArgs, "--skip-extended-insert") {
		t.Fatalf("mysqldump args must never contain --skip-extended-insert: %v", dumpArgs)
	}
}

func Test_BuildMysqldumpArgs_WithExcludedTables_AddsQualifiedIgnoreTableArgs(t *testing.T) {
	uc := &CreateMysqlBackupUsecase{}
	databaseName := "oa_db"
	database := &mysqltypes.MysqlDatabase{
		Version:       mysql_engine.Version80,
		Database:      &databaseName,
		ExcludeTables: []string{"personnel_access_control_event", "personnel_real_time"},
	}

	args := uc.buildMysqldumpArgs(database)

	if !slices.Contains(args, "--ignore-table=oa_db.personnel_access_control_event") ||
		!slices.Contains(args, "--ignore-table=oa_db.personnel_real_time") {
		t.Fatalf("expected an --ignore-table arg per excluded table, got %v", args)
	}
}

func Test_BuildMysqldumpArgs_WhenExcludedTablesArePastedMultiline_TrimsAndSplitsThem(t *testing.T) {
	uc := &CreateMysqlBackupUsecase{}
	databaseName := "oa_db"
	database := &mysqltypes.MysqlDatabase{
		Version:  mysql_engine.Version80,
		Database: &databaseName,
		ExcludeTables: []string{
			"personnel_access_control_event",
			"\npersonnel_real_time",
			" ",
			"ext_alarm_message,\nmonitor_toxic_gas",
		},
	}

	args := uc.buildMysqldumpArgs(database)

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
func Test_BuildMysqldumpArgs_ForAnyDatabase_AlwaysSkipsTablespaces(t *testing.T) {
	uc := &CreateMysqlBackupUsecase{}
	databaseName := "oa_db"
	database := &mysqltypes.MysqlDatabase{
		Version:  mysql_engine.Version80,
		Database: &databaseName,
	}

	args := uc.buildMysqldumpArgs(database)

	if !slices.Contains(args, "--no-tablespaces") {
		t.Fatalf("expected --no-tablespaces, got %v", args)
	}
}

func Test_BuildMysqldumpArgs_WithTriggerPrivilege_KeepsTriggers(t *testing.T) {
	uc := &CreateMysqlBackupUsecase{}
	databaseName := "oa_db"
	database := &mysqltypes.MysqlDatabase{
		Version:    mysql_engine.Version80,
		Database:   &databaseName,
		Privileges: "SELECT,SHOW VIEW,TRIGGER",
	}

	args := uc.buildMysqldumpArgs(database)

	if !slices.Contains(args, "--triggers") || slices.Contains(args, "--skip-triggers") {
		t.Fatalf("expected --triggers and no --skip-triggers, got %v", args)
	}
}

// Triggers are on by default, so without the explicit opt-out mysqldump fails on SHOW TRIGGERS
// for a role that lacks the privilege instead of dumping without them.
func Test_BuildMysqldumpArgs_WithoutTriggerPrivilege_SkipsTriggers(t *testing.T) {
	uc := &CreateMysqlBackupUsecase{}
	databaseName := "oa_db"
	database := &mysqltypes.MysqlDatabase{
		Version:    mysql_engine.Version80,
		Database:   &databaseName,
		Privileges: "SELECT,SHOW VIEW",
	}

	args := uc.buildMysqldumpArgs(database)

	if !slices.Contains(args, "--skip-triggers") || slices.Contains(args, "--triggers") {
		t.Fatalf("expected --skip-triggers and no --triggers, got %v", args)
	}
}

func Test_BuildMysqldumpArgs_WithoutExcludedTables_OmitsIgnoreTableArgs(t *testing.T) {
	uc := &CreateMysqlBackupUsecase{}
	databaseName := "oa_db"
	database := &mysqltypes.MysqlDatabase{
		Version:  mysql_engine.Version80,
		Database: &databaseName,
	}

	args := uc.buildMysqldumpArgs(database)

	for _, arg := range args {
		if strings.HasPrefix(arg, "--ignore-table=") {
			t.Fatalf("expected no --ignore-table args, got %v", args)
		}
	}
}

// Test_Execute_WhenStorageTimesOut_ReportsStorageFailure_RegressionForIssue816 pins issue #816 for
// MySQL: a storage timeout that ended the backup context was reported as "backup cancelled".
func Test_Execute_WhenStorageTimesOut_ReportsStorageFailure_RegressionForIssue816(t *testing.T) {
	endpoint := containers.StartMysql(t, "mysql:8.0")
	database := &databases.Database{
		ID:    uuid.New(),
		Type:  databases.DatabaseTypeMysql,
		Mysql: databases.GetTestMysqlConfig(endpoint.Host, endpoint.Port),
	}

	backups_core_logical.AssertStorageTimeoutReportedAsStorageFailure(t, func(
		ctx context.Context,
		backup *backups_core_logical.LogicalBackup,
		backupConfig *backups_config_logical.LogicalBackupConfig,
		fileStore backups_core_logical.BackupFileStore,
	) error {
		_, err := GetCreateMysqlBackupUsecase().Execute(ctx, backup, backupConfig, database, fileStore, func(float64) {})

		return err
	})
}

func Test_Execute_WhenMysqldumpFailsBeforeProducingData_ReportsMysqldumpError(t *testing.T) {
	backups_core_logical.AssertDumpToolErrorReported(t, func(
		ctx context.Context,
		backup *backups_core_logical.LogicalBackup,
		backupConfig *backups_config_logical.LogicalBackupConfig,
		fileStore backups_core_logical.BackupFileStore,
		dumpToolBin string,
	) error {
		_, err := GetCreateMysqlBackupUsecase().streamToStorage(
			ctx, backup, backupConfig, dumpToolBin, nil, "", fileStore, func(float64) {},
			&mysqltypes.MysqlDatabase{Version: mysql_engine.Version80, Database: new("orders_db")},
		)

		return err
	})
}
