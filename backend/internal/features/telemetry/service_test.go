package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/features/databases/databases/mongodb"
	"databasus-backend/internal/features/databases/databases/mysql"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/features/storages"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
	"databasus-backend/internal/util/tools"
)

type fakeSender struct {
	calls []*CollectRequest
	err   error
}

func (f *fakeSender) Send(_ context.Context, req *CollectRequest) error {
	f.calls = append(f.calls, req)
	return f.err
}

type fakeDatabaseLister struct {
	databases []*databases.Database
	err       error
}

func (f *fakeDatabaseLister) GetAllDatabases() ([]*databases.Database, error) {
	return f.databases, f.err
}

type fakeStorageLister struct {
	storages []*storages.Storage
	err      error
}

func (f *fakeStorageLister) GetAllStorages() ([]*storages.Storage, error) {
	return f.storages, f.err
}

type fakeNotifierLister struct {
	notifiers []*notifiers.Notifier
	err       error
}

func (f *fakeNotifierLister) GetAllNotifiers() ([]*notifiers.Notifier, error) {
	return f.notifiers, f.err
}

type fakeBackupChecker struct {
	hasBackupSince map[uuid.UUID]bool
	latestBackups  map[uuid.UUID]*backups_core_logical.LogicalBackup
	err            error
	latestErr      error
}

func (f *fakeBackupChecker) HasSuccessfulBackupSince(
	databaseID uuid.UUID,
	_ time.Time,
) (bool, error) {
	if f.err != nil {
		return false, f.err
	}

	return f.hasBackupSince[databaseID], nil
}

func (f *fakeBackupChecker) GetLatestCompletedBackup(
	databaseID uuid.UUID,
) (*backups_core_logical.LogicalBackup, error) {
	if f.latestErr != nil {
		return nil, f.latestErr
	}

	return f.latestBackups[databaseID], nil
}

type fakeLatestPhysicalBackupReader struct {
	fullBackups         map[uuid.UUID]*physical_models.PhysicalFullBackup
	lastBackupTimes     map[uuid.UUID]time.Time
	latestFullBackupErr error
	lastBackupTimesErr  error
}

func (f *fakeLatestPhysicalBackupReader) GetLatestCompletedFullBackup(
	databaseID uuid.UUID,
) (*physical_models.PhysicalFullBackup, error) {
	if f.latestFullBackupErr != nil {
		return nil, f.latestFullBackupErr
	}

	return f.fullBackups[databaseID], nil
}

func (f *fakeLatestPhysicalBackupReader) GetLastBackupTimesByDatabaseIDs(
	databaseIDs []uuid.UUID,
) (map[uuid.UUID]time.Time, error) {
	if f.lastBackupTimesErr != nil {
		return nil, f.lastBackupTimesErr
	}

	foundLastBackupTimes := make(map[uuid.UUID]time.Time, len(databaseIDs))

	for _, databaseID := range databaseIDs {
		if lastBackupTime, hasBackup := f.lastBackupTimes[databaseID]; hasBackup {
			foundLastBackupTimes[databaseID] = lastBackupTime
		}
	}

	return foundLastBackupTimes, nil
}

type fakeUserCounter struct {
	count int64
	err   error
}

func (f *fakeUserCounter) GetUserCount(context.Context) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}

	return f.count, nil
}

type fakeVerificationAgentLister struct {
	agents []*verification_agents.Agent
	err    error
}

func (f *fakeVerificationAgentLister) ListAgents(context.Context) ([]*verification_agents.Agent, error) {
	return f.agents, f.err
}

type fakeVerificationConfigLister struct {
	enabled []*verification_config.BackupVerificationConfig
	err     error
}

func (f *fakeVerificationConfigLister) ListEnabled() (
	[]*verification_config.BackupVerificationConfig,
	error,
) {
	return f.enabled, f.err
}

type serviceDependencies struct {
	databaseLister           databaseLister
	storageLister            storageLister
	notifierLister           notifierLister
	backupChecker            backupChecker
	physicalBackupReader     latestPhysicalBackupReader
	verificationAgentLister  verificationAgentLister
	verificationConfigLister verificationConfigLister
	userCounter              userCounter
	sender                   TelemetrySender
}

// Every unset dependency falls back to an empty fake so a test names only what it varies.
func newServiceUnderTest(t *testing.T, dependencies serviceDependencies) *TelemetryService {
	t.Helper()

	if dependencies.databaseLister == nil {
		dependencies.databaseLister = &fakeDatabaseLister{}
	}

	if dependencies.storageLister == nil {
		dependencies.storageLister = &fakeStorageLister{}
	}

	if dependencies.notifierLister == nil {
		dependencies.notifierLister = &fakeNotifierLister{}
	}

	if dependencies.backupChecker == nil {
		dependencies.backupChecker = &fakeBackupChecker{}
	}

	if dependencies.physicalBackupReader == nil {
		dependencies.physicalBackupReader = &fakeLatestPhysicalBackupReader{}
	}

	if dependencies.verificationAgentLister == nil {
		dependencies.verificationAgentLister = &fakeVerificationAgentLister{}
	}

	if dependencies.verificationConfigLister == nil {
		dependencies.verificationConfigLister = &fakeVerificationConfigLister{}
	}

	if dependencies.userCounter == nil {
		dependencies.userCounter = &fakeUserCounter{}
	}

	loader := NewInstanceFileLoader(
		filepath.Join(t.TempDir(), "instance.json"),
		slog.New(slog.DiscardHandler),
	)

	return NewTelemetryService(
		loader,
		dependencies.sender,
		dependencies.databaseLister,
		dependencies.storageLister,
		dependencies.notifierLister,
		dependencies.backupChecker,
		dependencies.physicalBackupReader,
		dependencies.verificationAgentLister,
		dependencies.verificationConfigLister,
		dependencies.userCounter,
		"9.9.9",
		slog.New(slog.DiscardHandler),
	)
}

func availableStatus() *databases.HealthStatus {
	s := databases.HealthStatusAvailable
	return &s
}

func unavailableStatus() *databases.HealthStatus {
	s := databases.HealthStatusUnavailable
	return &s
}

func postgresDatabase(name string, status *databases.HealthStatus) *databases.Database {
	return &databases.Database{
		ID:           uuid.New(),
		Name:         name,
		Type:         databases.DatabaseTypePostgresLogical,
		HealthStatus: status,
		PostgresqlLogical: &postgresql_logical.PostgresqlLogicalDatabase{
			Version: tools.PostgresqlVersion("16"),
		},
	}
}

func physicalDatabase(
	name string,
	backupType postgresql_physical.BackupType,
	status *databases.HealthStatus,
) *databases.Database {
	return &databases.Database{
		ID:           uuid.New(),
		Name:         name,
		Type:         databases.DatabaseTypePostgresPhysical,
		HealthStatus: status,
		PostgresqlPhysical: &postgresql_physical.PostgresqlPhysicalDatabase{
			Version:    tools.PostgresqlVersion("17"),
			BackupType: backupType,
		},
	}
}

func databasesForEveryEngine(isSshTunnelEnabled bool) []*databases.Database {
	tunnel := sshtunnel.Config{
		IsEnabled: isSshTunnelEnabled,
		Host:      "bastion.internal",
		Port:      22,
		Username:  "backup",
		AuthType:  sshtunnel.AuthTypePassword,
		Password:  "tunnelpassword",
	}

	logicalPostgresDatabase := postgresDatabase("pg-logical", availableStatus())
	logicalPostgresDatabase.PostgresqlLogical.SshTunnel = tunnel

	physicalPostgresDatabase := physicalDatabase(
		"pg-physical", postgresql_physical.BackupTypeFullOnly, availableStatus())
	physicalPostgresDatabase.PostgresqlPhysical.SshTunnel = tunnel

	return []*databases.Database{
		logicalPostgresDatabase,
		physicalPostgresDatabase,
		{
			ID:           uuid.New(),
			Name:         "my",
			Type:         databases.DatabaseTypeMysql,
			HealthStatus: availableStatus(),
			Mysql:        &mysql.MysqlDatabase{Version: tools.MysqlVersion("8.0"), SshTunnel: tunnel},
		},
		{
			ID:           uuid.New(),
			Name:         "maria",
			Type:         databases.DatabaseTypeMariadb,
			HealthStatus: availableStatus(),
			Mariadb:      &mariadb.MariadbDatabase{Version: tools.MariadbVersion("10.6"), SshTunnel: tunnel},
		},
		{
			ID:           uuid.New(),
			Name:         "mongo",
			Type:         databases.DatabaseTypeMongodb,
			HealthStatus: availableStatus(),
			Mongodb:      &mongodb.MongodbDatabase{Version: tools.MongodbVersion("6.0"), SshTunnel: tunnel},
		},
	}
}

func assertEveryEntrySerializesSshTunnelEnabled(
	t *testing.T,
	sender *fakeSender,
	isSshTunnelEnabled bool,
) {
	t.Helper()

	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 5)

	for _, databaseEntry := range sender.calls[0].Databases {
		assert.Equal(t, isSshTunnelEnabled, databaseEntry.IsSshTunnelEnabled, databaseEntry.Type)

		encodedEntry, err := json.Marshal(databaseEntry)
		require.NoError(t, err)

		var decodedEntry map[string]any
		require.NoError(t, json.Unmarshal(encodedEntry, &decodedEntry))

		assert.Equal(t, isSshTunnelEnabled, decodedEntry["isSshTunnelEnabled"], databaseEntry.Type)
	}
}

func Test_BuildAndSend_WhenSshTunnelEnabled_EveryEngineSerializesIsSshTunnelEnabledAsTrue(
	t *testing.T,
) {
	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: databasesForEveryEngine(true)},
		sender:         sender,
	})

	require.NoError(t, service.BuildAndSend(t.Context()))

	assertEveryEntrySerializesSshTunnelEnabled(t, sender, true)
}

func Test_BuildAndSend_WhenSshTunnelDisabled_EveryEngineSerializesIsSshTunnelEnabledAsFalse(
	t *testing.T,
) {
	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: databasesForEveryEngine(false)},
		sender:         sender,
	})

	require.NoError(t, service.BuildAndSend(t.Context()))

	assertEveryEntrySerializesSshTunnelEnabled(t, sender, false)
}

func Test_BuildAndSend_ProducesExpectedRequest(t *testing.T) {
	pgDB := postgresDatabase("pg", availableStatus())
	mysqlDB := &databases.Database{
		ID:           uuid.New(),
		Name:         "my",
		Type:         databases.DatabaseTypeMysql,
		HealthStatus: availableStatus(),
		Mysql:        &mysql.MysqlDatabase{Version: tools.MysqlVersion("8.0")},
	}
	mariaDB := &databases.Database{
		ID:           uuid.New(),
		Name:         "maria",
		Type:         databases.DatabaseTypeMariadb,
		HealthStatus: availableStatus(),
		Mariadb:      &mariadb.MariadbDatabase{Version: tools.MariadbVersion("10.6")},
	}
	mongoDB := &databases.Database{
		ID:           uuid.New(),
		Name:         "mongo",
		Type:         databases.DatabaseTypeMongodb,
		HealthStatus: availableStatus(),
		Mongodb:      &mongodb.MongodbDatabase{Version: tools.MongodbVersion("6.0")},
	}

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{pgDB, mysqlDB, mariaDB, mongoDB}},
		storageLister: &fakeStorageLister{storages: []*storages.Storage{
			{Type: storages.StorageTypeS3},
			{Type: storages.StorageTypeLocal},
		}},
		notifierLister: &fakeNotifierLister{notifiers: []*notifiers.Notifier{
			{NotifierType: notifiers.NotifierTypeEmail},
			{NotifierType: notifiers.NotifierTypeTelegram},
		}},
		sender: sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)

	req := sender.calls[0]
	assert.Equal(t, "9.9.9", req.AppVersion)
	assert.Equal(t, runtime.GOOS, req.OS)
	assert.Equal(t, runtime.GOARCH, req.Arch)
	require.Len(t, req.Databases, 4)

	databaseTypes := make([]string, 0, len(req.Databases))

	for _, databaseEntry := range req.Databases {
		databaseTypes = append(databaseTypes, databaseEntry.Type)
	}

	assert.ElementsMatch(t,
		[]string{"POSTGRES_LOGICAL", "MYSQL", "MARIADB", "MONGODB"},
		databaseTypes,
	)

	assert.Equal(t, []string{"LOCAL", "S3"}, req.Storages)
	assert.Equal(t, []string{"EMAIL", "TELEGRAM"}, req.Notifiers)
	assert.Equal(t, time.Now().UTC().Format("2006-01-02"), req.InstalledAt)
	_, err := uuid.Parse(req.InstanceID)
	require.NoError(t, err)
}

func Test_BuildAndSend_PreservesStorageAndNotifierDuplicates(t *testing.T) {
	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		storageLister: &fakeStorageLister{storages: []*storages.Storage{
			{Type: storages.StorageTypeS3},
			{Type: storages.StorageTypeS3},
			{Type: storages.StorageTypeS3},
			{Type: storages.StorageTypeLocal},
		}},
		notifierLister: &fakeNotifierLister{notifiers: []*notifiers.Notifier{
			{NotifierType: notifiers.NotifierTypeEmail},
			{NotifierType: notifiers.NotifierTypeEmail},
			{NotifierType: notifiers.NotifierTypeTelegram},
		}},
		sender: sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)

	assert.Equal(t, []string{"LOCAL", "S3", "S3", "S3"}, sender.calls[0].Storages)
	assert.Equal(t, []string{"EMAIL", "EMAIL", "TELEGRAM"}, sender.calls[0].Notifiers)
}

func Test_BuildAndSend_WhenInstanceFileFails_DoesNotCallSender(t *testing.T) {
	// Construct a loader pointing at an unwritable path so LoadOrCreate returns false.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, writeFileForTest(blocker))

	sender := &fakeSender{}
	loader := NewInstanceFileLoader(
		filepath.Join(blocker, "nested", "instance.json"),
		slog.New(slog.DiscardHandler),
	)

	service := NewTelemetryService(
		loader,
		sender,
		&fakeDatabaseLister{},
		&fakeStorageLister{},
		&fakeNotifierLister{},
		&fakeBackupChecker{},
		&fakeLatestPhysicalBackupReader{},
		&fakeVerificationAgentLister{},
		&fakeVerificationConfigLister{},
		&fakeUserCounter{},
		"9.9.9",
		slog.New(slog.DiscardHandler),
	)

	require.NoError(t, service.BuildAndSend(context.Background()))
	assert.Empty(t, sender.calls)
}

func Test_BuildAndSend_WhenSenderFails_PropagatesError(t *testing.T) {
	sendErr := errors.New("network down")
	sender := &fakeSender{err: sendErr}

	service := newServiceUnderTest(t, serviceDependencies{
		sender: sender,
	})

	err := service.BuildAndSend(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, sendErr)
}

func Test_BuildAndSend_WhenDbHealthStatusAvailable_DbIncluded(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		sender:         sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	assert.Len(t, sender.calls[0].Databases, 1)
}

func Test_BuildAndSend_WhenDbHealthStatusUnavailable_DbExcluded(t *testing.T) {
	db := postgresDatabase("pg", unavailableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		sender:         sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	assert.Empty(t, sender.calls[0].Databases)
}

func Test_BuildAndSend_WhenHealthcheckOffAndRecentBackup_DbIncluded(t *testing.T) {
	db := postgresDatabase("pg", nil)

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		backupChecker:  &fakeBackupChecker{hasBackupSince: map[uuid.UUID]bool{db.ID: true}},
		sender:         sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)
	assert.Equal(t, "POSTGRES_LOGICAL", sender.calls[0].Databases[0].Type)
}

func Test_BuildAndSend_WhenHealthcheckOffAndNoRecentBackup_DbExcluded(t *testing.T) {
	db := postgresDatabase("pg", nil)

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		backupChecker:  &fakeBackupChecker{hasBackupSince: map[uuid.UUID]bool{}},
		sender:         sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	assert.Empty(t, sender.calls[0].Databases)
}

func Test_BuildAndSend_WhenBackupCheckerFails_ReturnsError(t *testing.T) {
	db := postgresDatabase("pg", nil)
	checkerErr := errors.New("db down")

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		backupChecker:  &fakeBackupChecker{err: checkerErr},
		sender:         sender,
	})

	err := service.BuildAndSend(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, checkerErr)
	assert.Empty(t, sender.calls)
}

func Test_BuildAndSend_WhenLatestBackupHasBothSizes_IncludesBoth(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		backupChecker: &fakeBackupChecker{
			latestBackups: map[uuid.UUID]*backups_core_logical.LogicalBackup{
				db.ID: {BackupSizeMb: 870.4, BackupRawDbSizeMb: 4321.7},
			},
		},
		sender: sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	databaseEntry := sender.calls[0].Databases[0]
	assert.Equal(t, int64(871), databaseEntry.BackupSizeMb)
	assert.Equal(t, int64(4322), databaseEntry.RawSizeMb)
}

func Test_BuildAndSend_WhenSizesAreSubMb_RoundsUpToOne(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		backupChecker: &fakeBackupChecker{
			latestBackups: map[uuid.UUID]*backups_core_logical.LogicalBackup{
				db.ID: {BackupSizeMb: 0.3, BackupRawDbSizeMb: 0.1},
			},
		},
		sender: sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	databaseEntry := sender.calls[0].Databases[0]
	assert.Equal(t, int64(1), databaseEntry.BackupSizeMb)
	assert.Equal(t, int64(1), databaseEntry.RawSizeMb)

	encodedEntry, err := json.Marshal(databaseEntry)
	require.NoError(t, err)
	assert.Contains(t, string(encodedEntry), "backupSizeMb")
	assert.Contains(t, string(encodedEntry), "rawSizeMb")
}

func Test_BuildAndSend_WhenRawSizeZero_IncludesOnlyBackupSize(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		backupChecker: &fakeBackupChecker{
			latestBackups: map[uuid.UUID]*backups_core_logical.LogicalBackup{
				db.ID: {BackupSizeMb: 100, BackupRawDbSizeMb: 0},
			},
		},
		sender: sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	databaseEntry := sender.calls[0].Databases[0]
	assert.Equal(t, int64(100), databaseEntry.BackupSizeMb)
	assert.Equal(t, int64(0), databaseEntry.RawSizeMb)

	encodedEntry, err := json.Marshal(databaseEntry)
	require.NoError(t, err)
	assert.NotContains(t, string(encodedEntry), "rawSizeMb")
	assert.Contains(t, string(encodedEntry), "backupSizeMb")
}

func Test_BuildAndSend_WhenBackupSizeZero_IncludesOnlyRawSize(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		backupChecker: &fakeBackupChecker{
			latestBackups: map[uuid.UUID]*backups_core_logical.LogicalBackup{
				db.ID: {BackupSizeMb: 0, BackupRawDbSizeMb: 999},
			},
		},
		sender: sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	databaseEntry := sender.calls[0].Databases[0]
	assert.Equal(t, int64(0), databaseEntry.BackupSizeMb)
	assert.Equal(t, int64(999), databaseEntry.RawSizeMb)
}

func Test_BuildAndSend_WhenNoCompletedBackup_OmitsBothSizes(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		sender:         sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	databaseEntry := sender.calls[0].Databases[0]
	assert.Equal(t, int64(0), databaseEntry.BackupSizeMb)
	assert.Equal(t, int64(0), databaseEntry.RawSizeMb)

	encodedEntry, err := json.Marshal(databaseEntry)
	require.NoError(t, err)
	assert.NotContains(t, string(encodedEntry), "rawSizeMb")
	assert.NotContains(t, string(encodedEntry), "backupSizeMb")
}

func Test_BuildAndSend_WhenLatestBackupLookupFails_ReturnsError(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())
	lookupErr := errors.New("query exploded")

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		backupChecker:  &fakeBackupChecker{latestErr: lookupErr},
		sender:         sender,
	})

	err := service.BuildAndSend(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, lookupErr)
	assert.Empty(t, sender.calls)
}

func Test_BuildAndSend_WhenAgentsRegistered_IncludesCapacityRows(t *testing.T) {
	registeredAgents := []*verification_agents.Agent{
		{
			ID:                uuid.New(),
			Name:              "agent-1",
			MaxCPU:            4,
			MaxRAMGb:          16,
			MaxDiskGb:         100,
			MaxConcurrentJobs: 2,
		},
		{
			ID:                uuid.New(),
			Name:              "agent-2",
			MaxCPU:            8,
			MaxRAMGb:          32,
			MaxDiskGb:         200,
			MaxConcurrentJobs: 4,
		},
	}

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		verificationAgentLister: &fakeVerificationAgentLister{agents: registeredAgents},
		sender:                  sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)

	assert.Equal(t, []VerificationAgentEntry{
		{MaxCPU: 4, MaxRAMGb: 16, MaxDiskGb: 100, MaxConcurrentJobs: 2},
		{MaxCPU: 8, MaxRAMGb: 32, MaxDiskGb: 200, MaxConcurrentJobs: 4},
	}, sender.calls[0].VerificationAgents)
}

func Test_BuildAndSend_WhenNoAgents_VerificationAgentsIsEmpty(t *testing.T) {
	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		sender: sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)

	require.NotNil(t, sender.calls[0].VerificationAgents)
	assert.Empty(t, sender.calls[0].VerificationAgents)

	encodedPayload, err := json.Marshal(sender.calls[0])
	require.NoError(t, err)
	assert.Contains(t, string(encodedPayload), `"verificationAgents":[]`)
}

func Test_BuildAndSend_WhenDbHasAfterBackupConfig_VerificationBlockOmitsIntervalType(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		verificationConfigLister: &fakeVerificationConfigLister{
			enabled: []*verification_config.BackupVerificationConfig{
				{
					DatabaseID:                     db.ID,
					IsScheduledVerificationEnabled: true,
					ScheduleType:                   verification_config.VerificationScheduleAfterBackup,
				},
			},
		},
		sender: sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	databaseEntry := sender.calls[0].Databases[0]
	require.NotNil(t, databaseEntry.Verification)
	assert.True(t, databaseEntry.Verification.IsEnabled)
	assert.Equal(t, "AFTER_BACKUP", databaseEntry.Verification.ScheduleType)
	assert.Empty(t, databaseEntry.Verification.IntervalType)

	encodedEntry, err := json.Marshal(databaseEntry)
	require.NoError(t, err)
	assert.Contains(t, string(encodedEntry), `"verification"`)
	assert.NotContains(t, string(encodedEntry), "intervalType")
}

func Test_BuildAndSend_WhenDbHasIntervalDailyConfig_IncludesIntervalType(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		verificationConfigLister: &fakeVerificationConfigLister{
			enabled: []*verification_config.BackupVerificationConfig{
				{
					DatabaseID:                     db.ID,
					IsScheduledVerificationEnabled: true,
					ScheduleType:                   verification_config.VerificationScheduleInterval,
					VerificationInterval:           intervals.Interval{Type: intervals.IntervalDaily},
				},
			},
		},
		sender: sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	databaseEntry := sender.calls[0].Databases[0]
	require.NotNil(t, databaseEntry.Verification)
	assert.True(t, databaseEntry.Verification.IsEnabled)
	assert.Equal(t, "INTERVAL", databaseEntry.Verification.ScheduleType)
	assert.Equal(t, "DAILY", databaseEntry.Verification.IntervalType)

	encodedEntry, err := json.Marshal(databaseEntry)
	require.NoError(t, err)
	assert.Contains(t, string(encodedEntry), `"intervalType":"DAILY"`)
}

func Test_BuildAndSend_WhenDbHasNoEnabledConfig_VerificationBlockAbsent(t *testing.T) {
	db := postgresDatabase("pg", availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		sender:         sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	databaseEntry := sender.calls[0].Databases[0]
	assert.Nil(t, databaseEntry.Verification)

	encodedEntry, err := json.Marshal(databaseEntry)
	require.NoError(t, err)
	assert.NotContains(t, string(encodedEntry), "verification")
}

func Test_BuildAndSend_WhenVerificationAgentListFails_ReturnsError(t *testing.T) {
	listErr := errors.New("agents query exploded")
	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		verificationAgentLister: &fakeVerificationAgentLister{err: listErr},
		sender:                  sender,
	})

	err := service.BuildAndSend(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, listErr)
	assert.Empty(t, sender.calls)
}

func Test_BuildAndSend_WhenVerificationConfigListFails_ReturnsError(t *testing.T) {
	listErr := errors.New("configs query exploded")
	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		verificationConfigLister: &fakeVerificationConfigLister{err: listErr},
		sender:                   sender,
	})

	err := service.BuildAndSend(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, listErr)
	assert.Empty(t, sender.calls)
}

func Test_BuildAndSend_WhenPhysicalDatabase_EmitsTypeBackupTypeAndFullBackupSizes(t *testing.T) {
	db := physicalDatabase(
		"pg-physical",
		postgresql_physical.BackupTypeFullIncrementalAndWalStream,
		availableStatus(),
	)

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		physicalBackupReader: &fakeLatestPhysicalBackupReader{
			fullBackups: map[uuid.UUID]*physical_models.PhysicalFullBackup{
				db.ID: {BackupSizeMb: new(38400.2), RawSizeMb: new(192000.7)},
			},
		},
		sender: sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	databaseEntry := sender.calls[0].Databases[0]
	assert.Equal(t, "POSTGRES_PHYSICAL", databaseEntry.Type)
	assert.Equal(t, "17", databaseEntry.Version)
	assert.Equal(t, "FULL_INCREMENTAL_WAL_STREAM", databaseEntry.BackupType)
	assert.Equal(t, int64(38401), databaseEntry.BackupSizeMb)
	assert.Equal(t, int64(192001), databaseEntry.RawSizeMb)

	encodedEntry, err := json.Marshal(databaseEntry)
	require.NoError(t, err)
	assert.Contains(t, string(encodedEntry), `"backupType":"FULL_INCREMENTAL_WAL_STREAM"`)
}

func Test_BuildAndSend_WhenPhysicalDatabaseHasNoFullBackup_OmitsSizes(t *testing.T) {
	db := physicalDatabase("pg-physical", postgresql_physical.BackupTypeFullOnly, availableStatus())

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		sender:         sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	databaseEntry := sender.calls[0].Databases[0]
	assert.Equal(t, "FULL", databaseEntry.BackupType)
	assert.Equal(t, int64(0), databaseEntry.BackupSizeMb)
	assert.Equal(t, int64(0), databaseEntry.RawSizeMb)

	encodedEntry, err := json.Marshal(databaseEntry)
	require.NoError(t, err)
	assert.NotContains(t, string(encodedEntry), "rawSizeMb")
	assert.NotContains(t, string(encodedEntry), "backupSizeMb")
}

func Test_BuildAndSend_WhenPhysicalFullBackupLookupFails_ReturnsError(t *testing.T) {
	db := physicalDatabase("pg-physical", postgresql_physical.BackupTypeFullOnly, availableStatus())
	lookupErr := errors.New("physical query exploded")

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister:       &fakeDatabaseLister{databases: []*databases.Database{db}},
		physicalBackupReader: &fakeLatestPhysicalBackupReader{latestFullBackupErr: lookupErr},
		sender:               sender,
	})

	err := service.BuildAndSend(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, lookupErr)
	assert.Empty(t, sender.calls)
}

func Test_BuildAndSend_WhenPhysicalDatabaseHasNoHealthStatusButRecentBackup_DatabaseIsReported(
	t *testing.T,
) {
	db := physicalDatabase("pg-physical", postgresql_physical.BackupTypeFullIncrementalAndWalStream, nil)

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		physicalBackupReader: &fakeLatestPhysicalBackupReader{
			lastBackupTimes: map[uuid.UUID]time.Time{
				db.ID: time.Now().UTC().Add(-2 * time.Hour),
			},
		},
		sender: sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	require.Len(t, sender.calls[0].Databases, 1)

	databaseEntry := sender.calls[0].Databases[0]
	assert.Equal(t, "POSTGRES_PHYSICAL", databaseEntry.Type)
	assert.Equal(t, "FULL_INCREMENTAL_WAL_STREAM", databaseEntry.BackupType)
}

func Test_BuildAndSend_WhenPhysicalDatabaseHasNoHealthStatusAndStaleBackup_DatabaseIsSkipped(
	t *testing.T,
) {
	db := physicalDatabase("pg-physical", postgresql_physical.BackupTypeFullOnly, nil)

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		physicalBackupReader: &fakeLatestPhysicalBackupReader{
			lastBackupTimes: map[uuid.UUID]time.Time{
				db.ID: time.Now().UTC().Add(-activeBackupWindow - time.Hour),
			},
		},
		sender: sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	assert.Empty(t, sender.calls[0].Databases)
}

func Test_BuildAndSend_WhenPhysicalDatabaseHasNoHealthStatusAndNoBackups_DatabaseIsSkipped(
	t *testing.T,
) {
	db := physicalDatabase("pg-physical", postgresql_physical.BackupTypeFullOnly, nil)

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{db}},
		sender:         sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	assert.Empty(t, sender.calls[0].Databases)
}

func Test_BuildAndSend_WhenPhysicalLastBackupTimeLookupFails_ReturnsError(t *testing.T) {
	db := physicalDatabase("pg-physical", postgresql_physical.BackupTypeFullOnly, nil)
	lookupErr := errors.New("last backup time query exploded")

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister:       &fakeDatabaseLister{databases: []*databases.Database{db}},
		physicalBackupReader: &fakeLatestPhysicalBackupReader{lastBackupTimesErr: lookupErr},
		sender:               sender,
	})

	err := service.BuildAndSend(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, lookupErr)
	assert.Empty(t, sender.calls)
}

func Test_BuildAndSend_WhenPayloadSerialized_ContainsDocumentedContractKeys(t *testing.T) {
	physicalDB := physicalDatabase(
		"pg-physical",
		postgresql_physical.BackupTypeFullIncrementalAndWalStream,
		availableStatus(),
	)

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		databaseLister: &fakeDatabaseLister{databases: []*databases.Database{physicalDB}},
		storageLister:  &fakeStorageLister{storages: []*storages.Storage{{Type: storages.StorageTypeLocal}}},
		notifierLister: &fakeNotifierLister{
			notifiers: []*notifiers.Notifier{{NotifierType: notifiers.NotifierTypeEmail}},
		},
		physicalBackupReader: &fakeLatestPhysicalBackupReader{
			fullBackups: map[uuid.UUID]*physical_models.PhysicalFullBackup{
				physicalDB.ID: {BackupSizeMb: new(870.0), RawSizeMb: new(4321.0)},
			},
		},
		verificationAgentLister: &fakeVerificationAgentLister{
			agents: []*verification_agents.Agent{
				{MaxCPU: 4, MaxRAMGb: 8, MaxDiskGb: 100, MaxConcurrentJobs: 2},
			},
		},
		userCounter: &fakeUserCounter{count: 3},
		sender:      sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)

	encodedPayload, err := json.Marshal(sender.calls[0])
	require.NoError(t, err)

	var decodedPayload map[string]any
	require.NoError(t, json.Unmarshal(encodedPayload, &decodedPayload))

	assert.ElementsMatch(t, []string{
		"instanceID",
		"appVersion",
		"os",
		"arch",
		"installedAt",
		"userCount",
		"databases",
		"storages",
		"notifiers",
		"verificationAgents",
	}, slices.Collect(maps.Keys(decodedPayload)))

	databaseEntry, isObject := decodedPayload["databases"].([]any)[0].(map[string]any)
	require.True(t, isObject)

	assert.ElementsMatch(t, []string{
		"type",
		"version",
		"backupType",
		"isSshTunnelEnabled",
		"rawSizeMb",
		"backupSizeMb",
	}, slices.Collect(maps.Keys(databaseEntry)))
}

func Test_BuildAndSend_WhenUsersCounted_PopulatesUserCount(t *testing.T) {
	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		userCounter: &fakeUserCounter{count: 7},
		sender:      sender,
	})

	require.NoError(t, service.BuildAndSend(context.Background()))
	require.Len(t, sender.calls, 1)
	assert.Equal(t, 7, sender.calls[0].UserCount)
}

func Test_BuildAndSend_WhenUserCounterFails_ReturnsError(t *testing.T) {
	countErr := errors.New("users count exploded")

	sender := &fakeSender{}
	service := newServiceUnderTest(t, serviceDependencies{
		userCounter: &fakeUserCounter{err: countErr},
		sender:      sender,
	})

	err := service.BuildAndSend(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, countErr)
	assert.Empty(t, sender.calls)
}

func writeFileForTest(path string) error {
	return os.WriteFile(path, []byte("x"), 0o600)
}
