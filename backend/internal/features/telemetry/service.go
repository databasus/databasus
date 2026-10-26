package telemetry

import (
	"context"
	"log/slog"
	"math"
	"runtime"
	"slices"
	"time"

	"github.com/google/uuid"

	verification_config "databasus-backend/internal/features/verification/config"
	"databasus-backend/internal/modules/databases"
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
	storageService            storageTypeLister
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
	storageService storageTypeLister,
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

	databaseEntries, err := s.collectActiveDatabases(ctx, enabledConfigsByDatabaseID)
	if err != nil {
		return err
	}

	storageTypes, err := s.collectStorageTypes(ctx)
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
	ctx context.Context,
	enabledConfigsByDatabaseID map[uuid.UUID]*verification_config.BackupVerificationConfig,
) ([]DatabaseEntry, error) {
	allDatabases, err := s.databaseService.GetTelemetry(ctx)
	if err != nil {
		return nil, err
	}

	since := time.Now().UTC().Add(-activeBackupWindow)
	entries := make([]DatabaseEntry, 0, len(allDatabases))

	for _, db := range allDatabases {
		isActive, err := s.isDatabaseActive(db, since)
		if err != nil {
			return nil, err
		}

		if !isActive {
			continue
		}

		entry := buildDatabaseEntry(db)

		if err := s.attachBackupSizes(&entry, db); err != nil {
			return nil, err
		}

		if config, hasConfig := enabledConfigsByDatabaseID[db.DatabaseID]; hasConfig {
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
	db databases.TelemetryEntry,
) error {
	if db.Type == databases.TypePostgresqlPhysical {
		return s.attachPhysicalBackupSizes(entry, db.DatabaseID)
	}

	backup, err := s.backupService.GetLatestCompletedBackup(db.DatabaseID)
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
func (s *TelemetryService) isDatabaseActive(db databases.TelemetryEntry, since time.Time) (bool, error) {
	if db.HealthStatus != nil {
		return *db.HealthStatus == databases.HealthStatusAvailable, nil
	}

	if db.Type == databases.TypePostgresqlPhysical {
		return db.LastBackupTime != nil && db.LastBackupTime.After(since), nil
	}

	return s.backupService.HasSuccessfulBackupSince(db.DatabaseID, since)
}

func buildDatabaseEntry(db databases.TelemetryEntry) DatabaseEntry {
	return DatabaseEntry{
		Type:               string(db.Type),
		Version:            db.Version,
		BackupType:         string(db.PhysicalBackupType),
		IsSshTunnelEnabled: db.IsSSHTunnelEnabled,
	}
}

func (s *TelemetryService) collectStorageTypes(ctx context.Context) ([]string, error) {
	allStorageTypes, err := s.storageService.GetStorageTypes(ctx)
	if err != nil {
		return nil, err
	}

	storageTypes := make([]string, 0, len(allStorageTypes))
	for _, storageType := range allStorageTypes {
		storageTypes = append(storageTypes, string(storageType))
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
