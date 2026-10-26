package telemetry

import (
	"context"
	"log/slog"
	"math"
	"runtime"
	"slices"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/databases"
	verification_config "databasus-backend/internal/features/verification/config"
)

const (
	// activeBackupWindow is how far back a successful backup must have happened
	// for a database with disabled healthcheck to count as "active".
	activeBackupWindow = 7 * 24 * time.Hour

	// maxArrayEntries matches the server-side cap from TELEMETRY.md.
	maxArrayEntries = 200
)

type TelemetryService struct {
	instanceLoader            *InstanceFileLoader
	sender                    TelemetrySender
	databaseService           databaseLister
	storageService            storageLister
	notifierService           notifierLister
	backupService             backupChecker
	physicalBackupService     latestPhysicalBackupReader
	verificationAgentService  verificationAgentLister
	verificationConfigService verificationConfigLister
	userService               userCounter
	appVersion                string
	logger                    *slog.Logger
}

func NewTelemetryService(
	instanceLoader *InstanceFileLoader,
	sender TelemetrySender,
	databaseService databaseLister,
	storageService storageLister,
	notifierService notifierLister,
	backupService backupChecker,
	physicalBackupService latestPhysicalBackupReader,
	verificationAgentService verificationAgentLister,
	verificationConfigService verificationConfigLister,
	userService userCounter,
	appVersion string,
	logger *slog.Logger,
) *TelemetryService {
	return &TelemetryService{
		instanceLoader:            instanceLoader,
		sender:                    sender,
		databaseService:           databaseService,
		storageService:            storageService,
		notifierService:           notifierService,
		backupService:             backupService,
		physicalBackupService:     physicalBackupService,
		verificationAgentService:  verificationAgentService,
		verificationConfigService: verificationConfigService,
		userService:               userService,
		appVersion:                appVersion,
		logger:                    logger,
	}
}

func (s *TelemetryService) BuildAndSend(ctx context.Context) error {
	instance, ok := s.instanceLoader.LoadOrCreate()
	if !ok {
		return nil
	}

	enabledConfigsByDatabaseID, err := s.loadEnabledVerificationConfigs()
	if err != nil {
		return err
	}

	databaseEntries, err := s.collectActiveDatabases(enabledConfigsByDatabaseID)
	if err != nil {
		return err
	}

	storageTypes, err := s.collectStorageTypes()
	if err != nil {
		return err
	}

	notifierTypes, err := s.collectNotifierTypes(ctx)
	if err != nil {
		return err
	}

	verificationAgents, err := s.collectVerificationAgents(ctx)
	if err != nil {
		return err
	}

	userCount, err := s.userService.GetUserCount(ctx)
	if err != nil {
		return err
	}

	req := &CollectRequest{
		InstanceID:         instance.InstanceID,
		AppVersion:         s.appVersion,
		OS:                 runtime.GOOS,
		Arch:               runtime.GOARCH,
		InstalledAt:        instance.InstalledAt,
		UserCount:          int(userCount),
		Databases:          capEntries(databaseEntries),
		Storages:           capEntries(storageTypes),
		Notifiers:          capEntries(notifierTypes),
		VerificationAgents: capEntries(verificationAgents),
	}

	return s.sender.Send(ctx, req)
}

func (s *TelemetryService) loadEnabledVerificationConfigs() (
	map[uuid.UUID]*verification_config.BackupVerificationConfig,
	error,
) {
	enabledConfigs, err := s.verificationConfigService.ListEnabled()
	if err != nil {
		return nil, err
	}

	indexed := make(map[uuid.UUID]*verification_config.BackupVerificationConfig, len(enabledConfigs))
	for _, config := range enabledConfigs {
		indexed[config.DatabaseID] = config
	}

	return indexed, nil
}

func (s *TelemetryService) collectActiveDatabases(
	enabledConfigsByDatabaseID map[uuid.UUID]*verification_config.BackupVerificationConfig,
) ([]DatabaseEntry, error) {
	allDatabases, err := s.databaseService.GetAllDatabases()
	if err != nil {
		return nil, err
	}

	lastPhysicalBackupTimes, err := s.loadLastPhysicalBackupTimes(allDatabases)
	if err != nil {
		return nil, err
	}

	since := time.Now().UTC().Add(-activeBackupWindow)
	entries := make([]DatabaseEntry, 0, len(allDatabases))

	for _, db := range allDatabases {
		isActive, err := s.isDatabaseActive(db, since, lastPhysicalBackupTimes)
		if err != nil {
			return nil, err
		}

		if !isActive {
			continue
		}

		entry, ok := buildDatabaseEntry(db)
		if !ok {
			continue
		}

		if err := s.attachBackupSizes(&entry, db); err != nil {
			return nil, err
		}

		if config, hasConfig := enabledConfigsByDatabaseID[db.ID]; hasConfig {
			entry.Verification = buildVerificationEntry(config)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func buildVerificationEntry(
	config *verification_config.BackupVerificationConfig,
) *DatabaseVerificationEntry {
	entry := &DatabaseVerificationEntry{
		IsEnabled:    true,
		ScheduleType: string(config.ScheduleType),
	}

	if config.ScheduleType == verification_config.VerificationScheduleInterval {
		entry.IntervalType = string(config.VerificationInterval.Type)
	}

	return entry
}

func (s *TelemetryService) collectVerificationAgents(ctx context.Context) ([]VerificationAgentEntry, error) {
	listedAgents, err := s.verificationAgentService.ListAgents(ctx)
	if err != nil {
		return nil, err
	}

	entries := make([]VerificationAgentEntry, 0, len(listedAgents))
	for _, agent := range listedAgents {
		entries = append(entries, VerificationAgentEntry{
			MaxCPU:            agent.MaxCPU,
			MaxRAMGb:          agent.MaxRAMGb,
			MaxDiskGb:         agent.MaxDiskGb,
			MaxConcurrentJobs: agent.MaxConcurrentJobs,
		})
	}

	return entries, nil
}

// Physical databases have no logical backup rows, so their size comes from the
// latest completed physical FULL backup.
func (s *TelemetryService) attachBackupSizes(
	entry *DatabaseEntry,
	db *databases.Database,
) error {
	if db.Type == databases.DatabaseTypePostgresPhysical {
		return s.attachPhysicalBackupSizes(entry, db.ID)
	}

	backup, err := s.backupService.GetLatestCompletedBackup(db.ID)
	if err != nil {
		return err
	}

	if backup == nil {
		return nil
	}

	if backup.BackupSizeMb > 0 {
		entry.BackupSizeMb = int64(math.Ceil(backup.BackupSizeMb))
	}

	if backup.BackupRawDbSizeMb > 0 {
		entry.RawSizeMb = int64(math.Ceil(backup.BackupRawDbSizeMb))
	}

	return nil
}

func (s *TelemetryService) attachPhysicalBackupSizes(
	entry *DatabaseEntry,
	databaseID uuid.UUID,
) error {
	fullBackup, err := s.physicalBackupService.GetLatestCompletedFullBackup(databaseID)
	if err != nil {
		return err
	}

	if fullBackup == nil {
		return nil
	}

	if fullBackup.BackupSizeMb != nil && *fullBackup.BackupSizeMb > 0 {
		entry.BackupSizeMb = int64(math.Ceil(*fullBackup.BackupSizeMb))
	}

	if fullBackup.RawSizeMb != nil && *fullBackup.RawSizeMb > 0 {
		entry.RawSizeMb = int64(math.Ceil(*fullBackup.RawSizeMb))
	}

	return nil
}

// A nil HealthStatus means the healthcheck is switched off for that database, not that it is
// broken, so the backup recency of the owning engine decides instead.
func (s *TelemetryService) isDatabaseActive(
	db *databases.Database,
	since time.Time,
	lastPhysicalBackupTimes map[uuid.UUID]time.Time,
) (bool, error) {
	if db.HealthStatus != nil {
		return *db.HealthStatus == databases.HealthStatusAvailable, nil
	}

	if db.Type == databases.DatabaseTypePostgresPhysical {
		lastBackupTime, hasBackup := lastPhysicalBackupTimes[db.ID]

		return hasBackup && lastBackupTime.After(since), nil
	}

	return s.backupService.HasSuccessfulBackupSince(db.ID, since)
}

// Only databases whose activity check will actually read the map are queried, so an instance
// where every physical database is healthchecked never pays for — nor fails a whole ping on — a
// query it would ignore.
func (s *TelemetryService) loadLastPhysicalBackupTimes(
	allDatabases []*databases.Database,
) (map[uuid.UUID]time.Time, error) {
	physicalDatabaseIDs := make([]uuid.UUID, 0, len(allDatabases))

	for _, db := range allDatabases {
		if db.Type == databases.DatabaseTypePostgresPhysical && db.HealthStatus == nil {
			physicalDatabaseIDs = append(physicalDatabaseIDs, db.ID)
		}
	}

	if len(physicalDatabaseIDs) == 0 {
		return map[uuid.UUID]time.Time{}, nil
	}

	return s.physicalBackupService.GetLastBackupTimesByDatabaseIDs(physicalDatabaseIDs)
}

func buildDatabaseEntry(db *databases.Database) (DatabaseEntry, bool) {
	switch db.Type {
	case databases.DatabaseTypePostgresLogical:
		// The legacy POSTGRES type is the same engine now labelled POSTGRES_LOGICAL;
		// analytics counts the two as one.
		if db.PostgresqlLogical == nil {
			return DatabaseEntry{}, false
		}

		return DatabaseEntry{
			Type:               string(db.Type),
			Version:            string(db.PostgresqlLogical.Version),
			IsSshTunnelEnabled: db.PostgresqlLogical.SshTunnel.IsEnabled,
		}, true
	case databases.DatabaseTypePostgresPhysical:
		if db.PostgresqlPhysical == nil {
			return DatabaseEntry{}, false
		}

		return DatabaseEntry{
			Type:               string(db.Type),
			Version:            string(db.PostgresqlPhysical.Version),
			BackupType:         string(db.PostgresqlPhysical.BackupType),
			IsSshTunnelEnabled: db.PostgresqlPhysical.SshTunnel.IsEnabled,
		}, true
	case databases.DatabaseTypeMysql:
		if db.Mysql == nil {
			return DatabaseEntry{}, false
		}

		return DatabaseEntry{
			Type:               string(db.Type),
			Version:            string(db.Mysql.Version),
			IsSshTunnelEnabled: db.Mysql.SshTunnel.IsEnabled,
		}, true
	case databases.DatabaseTypeMariadb:
		if db.Mariadb == nil {
			return DatabaseEntry{}, false
		}

		return DatabaseEntry{
			Type:               string(db.Type),
			Version:            string(db.Mariadb.Version),
			IsSshTunnelEnabled: db.Mariadb.SshTunnel.IsEnabled,
		}, true
	case databases.DatabaseTypeMongodb:
		if db.Mongodb == nil {
			return DatabaseEntry{}, false
		}

		return DatabaseEntry{
			Type:               string(db.Type),
			Version:            string(db.Mongodb.Version),
			IsSshTunnelEnabled: db.Mongodb.SshTunnel.IsEnabled,
		}, true
	}

	return DatabaseEntry{}, false
}

func (s *TelemetryService) collectStorageTypes() ([]string, error) {
	allStorages, err := s.storageService.GetAllStorages()
	if err != nil {
		return nil, err
	}

	storageTypes := make([]string, 0, len(allStorages))

	for _, storage := range allStorages {
		if storage.Type == "" {
			continue
		}

		storageTypes = append(storageTypes, string(storage.Type))
	}

	slices.Sort(storageTypes)

	return storageTypes, nil
}

func (s *TelemetryService) collectNotifierTypes(ctx context.Context) ([]string, error) {
	allNotifierTypes, err := s.notifierService.GetNotifierTypes(ctx)
	if err != nil {
		return nil, err
	}

	notifierTypes := make([]string, 0, len(allNotifierTypes))
	for _, notifierType := range allNotifierTypes {
		notifierTypes = append(notifierTypes, string(notifierType))
	}

	slices.Sort(notifierTypes)

	return notifierTypes, nil
}

func capEntries[T any](entries []T) []T {
	if len(entries) > maxArrayEntries {
		return entries[:maxArrayEntries]
	}

	return entries
}
