package report

import (
	"context"
	"log/slog"
	"math"
	"runtime"
	"slices"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/verification"
)

const (
	activeBackupWindow = 7 * 24 * time.Hour

	maxArrayEntries = 200
)

type Deps struct {
	Users        users.Module
	Databases    databases.Module
	Storages     storages.Module
	Notifiers    notifiers.Module
	Backups      backups.Module
	Verification verification.Module
	Clock        clock.Source
	Logger       *slog.Logger
	Version      string
	ReceiverURL  string
	IdentityPath string
	IsDisabled   bool
}

type Service struct {
	deps      Deps
	identity  *IdentityLoader
	receiver  *Receiver
	nextDueAt *time.Time
	backoff   time.Duration
}

func New(deps Deps) *Service {
	return &Service{
		deps:      deps,
		identity:  NewIdentityLoader(deps),
		receiver:  NewReceiver(deps),
		nextDueAt: nil,
		backoff:   time.Minute,
	}
}

func (s *Service) buildAndSend(ctx context.Context) error {
	instance, isLoaded := s.identity.LoadOrCreate(ctx)
	if !isLoaded {
		return nil
	}

	enabledConfigsByDatabaseID, err := s.loadEnabledVerificationConfigs(ctx)
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

	userCount, err := s.deps.Users.GetUserCount(ctx)
	if err != nil {
		return err
	}

	req := &CollectRequest{
		InstanceID:         instance.InstanceID,
		AppVersion:         s.deps.Version,
		OS:                 runtime.GOOS,
		Arch:               runtime.GOARCH,
		InstalledAt:        instance.InstalledAt,
		UserCount:          int(userCount),
		Databases:          capEntries(databaseEntries),
		Storages:           capEntries(storageTypes),
		Notifiers:          capEntries(notifierTypes),
		VerificationAgents: capEntries(verificationAgents),
	}

	return s.receiver.Send(ctx, req)
}

func (s *Service) loadEnabledVerificationConfigs(ctx context.Context) (
	map[uuid.UUID]verification.VerificationConfig,
	error,
) {
	enabledConfigs, err := s.deps.Verification.ListEnabledConfigs(ctx)
	if err != nil {
		return nil, err
	}

	indexed := make(map[uuid.UUID]verification.VerificationConfig, len(enabledConfigs))
	for _, config := range enabledConfigs {
		indexed[config.DatabaseID] = config
	}

	return indexed, nil
}

func (s *Service) collectActiveDatabases(
	ctx context.Context,
	enabledConfigsByDatabaseID map[uuid.UUID]verification.VerificationConfig,
) ([]DatabaseEntry, error) {
	allDatabases, err := s.deps.Databases.GetTelemetry(ctx)
	if err != nil {
		return nil, err
	}

	since := s.deps.Clock.Now().UTC().Add(-activeBackupWindow)
	entries := make([]DatabaseEntry, 0, len(allDatabases))

	backupMethods := getBackupMethods()
	logicalIDs := make([]uuid.UUID, 0, len(allDatabases))
	physicalIDs := make([]uuid.UUID, 0, len(allDatabases))
	for _, database := range allDatabases {
		if backupMethods[database.Type] == backups.MethodPhysical {
			physicalIDs = append(physicalIDs, database.DatabaseID)
		} else {
			logicalIDs = append(logicalIDs, database.DatabaseID)
		}
	}

	logicalBackups, err := s.deps.Backups.GetLatestCompletedLogicalBackups(ctx, logicalIDs)
	if err != nil {
		return nil, err
	}
	physicalBackups, err := s.deps.Backups.GetLatestCompletedPhysicalFullBackups(ctx, physicalIDs)
	if err != nil {
		return nil, err
	}

	for _, database := range allDatabases {
		logicalBackup, hasLogicalBackup := logicalBackups[database.DatabaseID]

		isActive := false
		switch {
		case database.HealthStatus != nil:
			isActive = *database.HealthStatus == databases.HealthStatusAvailable
		case backupMethods[database.Type] == backups.MethodPhysical:
			isActive = database.LastBackupTime != nil && database.LastBackupTime.After(since)
		default:
			isActive = hasLogicalBackup && !logicalBackup.CreatedAt.Before(since)
		}
		if !isActive {
			continue
		}

		entry := buildDatabaseEntry(database)
		if backupMethods[database.Type] == backups.MethodPhysical {
			if backup, hasBackup := physicalBackups[database.DatabaseID]; hasBackup {
				if backup.BackupSizeMb != nil && *backup.BackupSizeMb > 0 {
					entry.BackupSizeMb = int64(math.Ceil(*backup.BackupSizeMb))
				}
				if backup.RawSizeMb != nil && *backup.RawSizeMb > 0 {
					entry.RawSizeMb = int64(math.Ceil(*backup.RawSizeMb))
				}
			}
		} else if hasLogicalBackup {
			if logicalBackup.BackupSizeMb > 0 {
				entry.BackupSizeMb = int64(math.Ceil(logicalBackup.BackupSizeMb))
			}
			if logicalBackup.BackupRawDbSizeMb > 0 {
				entry.RawSizeMb = int64(math.Ceil(logicalBackup.BackupRawDbSizeMb))
			}
		}

		if config, hasConfig := enabledConfigsByDatabaseID[database.DatabaseID]; hasConfig {
			entry.Verification = buildVerificationEntry(config)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func buildVerificationEntry(
	config verification.VerificationConfig,
) *DatabaseVerificationEntry {
	entry := &DatabaseVerificationEntry{
		IsEnabled:    true,
		ScheduleType: string(config.ScheduleType),
	}

	if config.ScheduleType == verification.ScheduleInterval {
		entry.IntervalType = string(config.Schedule.Type)
	}

	return entry
}

func (s *Service) collectVerificationAgents(ctx context.Context) ([]VerificationAgentEntry, error) {
	listedAgents, err := s.deps.Verification.ListAgents(ctx)
	if err != nil {
		return nil, err
	}

	entries := make([]VerificationAgentEntry, 0, len(listedAgents))
	for _, agent := range listedAgents {
		entries = append(entries, VerificationAgentEntry{
			MaxCPU:            agent.MaxCPU,
			MaxRAMMb:          agent.MaxRAMMb,
			MaxDiskGb:         agent.MaxDiskGb,
			MaxConcurrentJobs: agent.MaxConcurrentJobs,
		})
	}

	return entries, nil
}

func buildDatabaseEntry(db databases.TelemetryEntry) DatabaseEntry {
	return DatabaseEntry{
		Type:               string(db.Type),
		Version:            db.Version,
		BackupType:         string(db.PhysicalBackupType),
		IsSSHTunnelEnabled: db.IsSSHTunnelEnabled,
	}
}

func (s *Service) collectStorageTypes(ctx context.Context) ([]string, error) {
	allStorageTypes, err := s.deps.Storages.GetStorageTypes(ctx)
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

func (s *Service) collectNotifierTypes(ctx context.Context) ([]string, error) {
	allNotifierTypes, err := s.deps.Notifiers.GetNotifierTypes(ctx)
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
