package config

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/schedule"
)

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// services never hold it.
type Repositories struct {
	Configs *Repository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{Configs: &Repository{db: db}}
}

// Repository keeps the shared fields of a config in backups_configs and the fields of its method
// in backups_configs_logical or backups_configs_physical.
type Repository struct {
	db *gorm.DB
}

// RowLock is the lock a read takes on the config's row until the caller's transaction ends.
type RowLock string

const (
	ForUpdateRowLock RowLock = " FOR UPDATE"
	ForShareRowLock  RowLock = " FOR SHARE"
)

type configListFilter struct {
	// databaseIDs limits the list to these databases when set.
	databaseIDs []uuid.UUID
	// storageID limits the list to the configs that back up into this storage when set.
	storageID *uuid.UUID
	afterID   *uuid.UUID
	limit     int
}

type backupConfigRow struct {
	StorageGeneration    int64          `gorm:"column:storage_generation"`
	DatabaseID           uuid.UUID      `gorm:"column:database_id"`
	Method               Method         `gorm:"column:method"`
	IsBackupsEnabled     bool           `gorm:"column:is_backups_enabled"`
	StorageID            *uuid.UUID     `gorm:"column:storage_id"`
	Encryption           Encryption     `gorm:"column:encryption"`
	NotificationTriggers pq.StringArray `gorm:"column:notification_triggers;type:text[]"`
}

type scheduleColumns struct {
	Type           string  `gorm:"column:interval_type"`
	TimeOfDay      *string `gorm:"column:time_of_day"`
	Weekday        *int    `gorm:"column:weekday"`
	DayOfMonth     *int    `gorm:"column:day_of_month"`
	CronExpression *string `gorm:"column:cron_expression"`
}

type logicalConfigRow struct {
	Common              backupConfigRow        `gorm:"embedded"`
	Schedule            scheduleColumns        `gorm:"embedded"`
	RetentionPolicy     LogicalRetentionPolicy `gorm:"column:retention_policy_type"`
	RetentionPeriod     RetentionPeriod        `gorm:"column:retention_time_period"`
	RetentionCount      int                    `gorm:"column:retention_count"`
	RetentionGfs        gfsColumns             `gorm:"embedded;embeddedPrefix:retention_gfs_"`
	IsRetryIfFailed     bool                   `gorm:"column:is_retry_if_failed"`
	MaxFailedTriesCount int                    `gorm:"column:max_failed_tries_count"`
}

type physicalConfigRow struct {
	Common                      backupConfigRow            `gorm:"embedded"`
	FullSchedule                scheduleColumns            `gorm:"embedded;embeddedPrefix:full_"`
	IncrementalSchedule         scheduleColumns            `gorm:"embedded;embeddedPrefix:incremental_"`
	RetentionPolicy             PhysicalRetentionPolicy    `gorm:"column:retention"`
	ChainsRetentionCount        int                        `gorm:"column:chains_retention_count"`
	FullBackupsRetentionPolicy  FullBackupsRetentionPolicy `gorm:"column:full_backups_retention_policy"`
	FullBackupsRetentionCount   int                        `gorm:"column:full_backups_retention_count"`
	FullBackupsRetentionGfs     gfsColumns                 `gorm:"embedded;embeddedPrefix:full_backups_retention_gfs_"`
	WalLagThresholdBytes        int64                      `gorm:"column:wal_lag_threshold_bytes"`
	ForceFullRequestedAt        *time.Time                 `gorm:"column:force_full_requested_at"`
	ForceIncrementalRequestedAt *time.Time                 `gorm:"column:force_incremental_requested_at"`
}

type gfsColumns struct {
	Hours  int `gorm:"column:hours"`
	Days   int `gorm:"column:days"`
	Weeks  int `gorm:"column:weeks"`
	Months int `gorm:"column:months"`
	Years  int `gorm:"column:years"`
}

const (
	backupConfigColumns = "backups_configs.database_id, backups_configs.method, backups_configs.is_backups_enabled, " +
		"backups_configs.storage_generation, backups_configs.storage_id, backups_configs.encryption, backups_configs.notification_triggers"
	logicalConfigQuery = "SELECT " + backupConfigColumns + ", interval_type, time_of_day, weekday, day_of_month, " +
		"cron_expression, retention_policy_type, retention_time_period, retention_count, retention_gfs_hours, " +
		"retention_gfs_days, retention_gfs_weeks, retention_gfs_months, retention_gfs_years, is_retry_if_failed, " +
		"max_failed_tries_count FROM backups_configs " +
		"JOIN backups_configs_logical ON backups_configs_logical.database_id = backups_configs.database_id"
	physicalConfigQuery = "SELECT " + backupConfigColumns + ", full_interval_type, full_time_of_day, full_weekday, " +
		"full_day_of_month, full_cron_expression, incremental_interval_type, incremental_time_of_day, " +
		"incremental_weekday, incremental_day_of_month, incremental_cron_expression, retention, " +
		"chains_retention_count, full_backups_retention_policy, full_backups_retention_count, " +
		"full_backups_retention_gfs_hours, full_backups_retention_gfs_days, full_backups_retention_gfs_weeks, " +
		"full_backups_retention_gfs_months, full_backups_retention_gfs_years, wal_lag_threshold_bytes, " +
		"force_full_requested_at, force_incremental_requested_at FROM backups_configs " +
		"JOIN backups_configs_physical ON backups_configs_physical.database_id = backups_configs.database_id"
)

// LockDatabaseConfig serializes the writes of one database's config until the caller's transaction
// ends. A row lock cannot do it before the first save, when the config has no row yet.
func (r *Repository) LockDatabaseConfig(ctx context.Context, databaseID uuid.UUID) error {
	err := database.Conn(ctx, r.db).
		Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", "backups_configs:"+databaseID.String()).
		Error
	if err != nil {
		return fmt.Errorf("lock the backup config: %w", err)
	}

	return nil
}

func (r *Repository) GetBackupConfig(
	ctx context.Context,
	databaseID uuid.UUID,
	lock RowLock,
) (BackupConfig, bool, error) {
	var rows []backupConfigRow

	err := database.Conn(ctx, r.db).
		Raw("SELECT "+backupConfigColumns+" FROM backups_configs WHERE database_id = ?"+string(lock), databaseID).
		Scan(&rows).Error
	if err != nil {
		return BackupConfig{}, false, fmt.Errorf("get the backup config: %w", err)
	}

	if len(rows) == 0 {
		return BackupConfig{}, false, nil
	}

	return rows[0].toConfig(), true, nil
}

func (r *Repository) ListPhysical(ctx context.Context, databaseIDs []uuid.UUID) ([]PhysicalBackupConfig, error) {
	return r.queryPhysical(ctx, physicalConfigQuery+" WHERE backups_configs.database_id IN ?", databaseIDs)
}

// UpsertLogical expects the caller's transaction in ctx, so the shared row never exists without the
// row of its method.
func (r *Repository) UpsertLogical(ctx context.Context, config LogicalBackupConfig) error {
	err := r.upsertBackupConfig(ctx, MethodLogical, config.BackupConfig)
	if err != nil {
		return err
	}

	err = database.Conn(ctx, r.db).Exec(`INSERT INTO backups_configs_logical (database_id, interval_type,
		time_of_day, weekday, day_of_month, cron_expression, retention_policy_type, retention_time_period,
		retention_count, retention_gfs_hours, retention_gfs_days, retention_gfs_weeks, retention_gfs_months,
		retention_gfs_years, is_retry_if_failed, max_failed_tries_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (database_id) DO UPDATE SET
			interval_type = EXCLUDED.interval_type,
			time_of_day = EXCLUDED.time_of_day,
			weekday = EXCLUDED.weekday,
			day_of_month = EXCLUDED.day_of_month,
			cron_expression = EXCLUDED.cron_expression,
			retention_policy_type = EXCLUDED.retention_policy_type,
			retention_time_period = EXCLUDED.retention_time_period,
			retention_count = EXCLUDED.retention_count,
			retention_gfs_hours = EXCLUDED.retention_gfs_hours,
			retention_gfs_days = EXCLUDED.retention_gfs_days,
			retention_gfs_weeks = EXCLUDED.retention_gfs_weeks,
			retention_gfs_months = EXCLUDED.retention_gfs_months,
			retention_gfs_years = EXCLUDED.retention_gfs_years,
			is_retry_if_failed = EXCLUDED.is_retry_if_failed,
			max_failed_tries_count = EXCLUDED.max_failed_tries_count`,
		config.DatabaseID,
		string(config.Schedule.Type), config.Schedule.TimeOfDay, config.Schedule.Weekday,
		config.Schedule.DayOfMonth, config.Schedule.CronExpression,
		config.Retention.Policy, config.Retention.Period, config.Retention.Count,
		config.Retention.Gfs.Hours, config.Retention.Gfs.Days, config.Retention.Gfs.Weeks,
		config.Retention.Gfs.Months, config.Retention.Gfs.Years,
		config.IsRetryIfFailed, config.MaxFailedTriesCount,
	).Error
	if err != nil {
		return fmt.Errorf("upsert the logical backup config: %w", err)
	}

	return nil
}

// UpsertPhysical keeps the force requests, which only their own writes change.
func (r *Repository) UpsertPhysical(ctx context.Context, config PhysicalBackupConfig) error {
	err := r.upsertBackupConfig(ctx, MethodPhysical, config.BackupConfig)
	if err != nil {
		return err
	}

	incrementalSchedule := toScheduleColumns(config.IncrementalBackupSchedule)

	err = database.Conn(ctx, r.db).Exec(`INSERT INTO backups_configs_physical (database_id, full_interval_type,
		full_time_of_day, full_weekday, full_day_of_month, full_cron_expression, incremental_interval_type,
		incremental_time_of_day, incremental_weekday, incremental_day_of_month, incremental_cron_expression,
		retention, chains_retention_count, full_backups_retention_policy, full_backups_retention_count,
		full_backups_retention_gfs_hours, full_backups_retention_gfs_days, full_backups_retention_gfs_weeks,
		full_backups_retention_gfs_months, full_backups_retention_gfs_years, wal_lag_threshold_bytes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (database_id) DO UPDATE SET
			full_interval_type = EXCLUDED.full_interval_type,
			full_time_of_day = EXCLUDED.full_time_of_day,
			full_weekday = EXCLUDED.full_weekday,
			full_day_of_month = EXCLUDED.full_day_of_month,
			full_cron_expression = EXCLUDED.full_cron_expression,
			incremental_interval_type = EXCLUDED.incremental_interval_type,
			incremental_time_of_day = EXCLUDED.incremental_time_of_day,
			incremental_weekday = EXCLUDED.incremental_weekday,
			incremental_day_of_month = EXCLUDED.incremental_day_of_month,
			incremental_cron_expression = EXCLUDED.incremental_cron_expression,
			retention = EXCLUDED.retention,
			chains_retention_count = EXCLUDED.chains_retention_count,
			full_backups_retention_policy = EXCLUDED.full_backups_retention_policy,
			full_backups_retention_count = EXCLUDED.full_backups_retention_count,
			full_backups_retention_gfs_hours = EXCLUDED.full_backups_retention_gfs_hours,
			full_backups_retention_gfs_days = EXCLUDED.full_backups_retention_gfs_days,
			full_backups_retention_gfs_weeks = EXCLUDED.full_backups_retention_gfs_weeks,
			full_backups_retention_gfs_months = EXCLUDED.full_backups_retention_gfs_months,
			full_backups_retention_gfs_years = EXCLUDED.full_backups_retention_gfs_years,
			wal_lag_threshold_bytes = EXCLUDED.wal_lag_threshold_bytes`,
		config.DatabaseID,
		string(config.FullBackupSchedule.Type), config.FullBackupSchedule.TimeOfDay,
		config.FullBackupSchedule.Weekday, config.FullBackupSchedule.DayOfMonth,
		config.FullBackupSchedule.CronExpression,
		incrementalSchedule.Type, incrementalSchedule.TimeOfDay, incrementalSchedule.Weekday,
		incrementalSchedule.DayOfMonth, incrementalSchedule.CronExpression,
		config.Retention.Policy, config.Retention.ChainsCount,
		config.Retention.FullBackups.Policy, config.Retention.FullBackups.Count,
		config.Retention.FullBackups.Gfs.Hours, config.Retention.FullBackups.Gfs.Days,
		config.Retention.FullBackups.Gfs.Weeks, config.Retention.FullBackups.Gfs.Months,
		config.Retention.FullBackups.Gfs.Years,
		config.WalLagThresholdBytes,
	).Error
	if err != nil {
		return fmt.Errorf("upsert the physical backup config: %w", err)
	}

	return nil
}

func (r *Repository) GetLastSelectedStorageID(
	ctx context.Context,
	databaseID uuid.UUID,
) (uuid.UUID, bool, error) {
	var rows []struct{ LastSelectedStorageID *uuid.UUID }
	err := database.Conn(ctx, r.db).
		Raw("SELECT last_selected_storage_id FROM backups_configs WHERE database_id=?", databaseID).
		Scan(&rows).Error
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("read the last selected backup storage: %w", err)
	}
	if len(rows) == 0 || rows[0].LastSelectedStorageID == nil {
		return uuid.Nil, false, nil
	}
	return *rows[0].LastSelectedStorageID, true, nil
}

func (r *Repository) IncrementStorageGeneration(ctx context.Context, databaseID uuid.UUID) error {
	if err := database.Conn(ctx, r.db).
		Exec("UPDATE backups_configs SET storage_generation=storage_generation+1 WHERE database_id=?", databaseID).
		Error; err != nil {
		return fmt.Errorf("advance storage generation: %w", err)
	}
	return nil
}

func (r *Repository) ResetPhysicalRequests(
	ctx context.Context,
	reset PhysicalRequestsReset,
) error {
	if err := database.Conn(ctx, r.db).
		Exec("UPDATE backups_configs_physical SET force_incremental_requested_at=NULL,force_full_requested_at=? WHERE database_id=?", reset.RequestedAt, reset.DatabaseID).
		Error; err != nil {
		return fmt.Errorf("reset physical backup requests: %w", err)
	}
	return nil
}

func (r *Repository) LockPhysicalHistory(ctx context.Context, databaseID uuid.UUID) error {
	return database.Conn(ctx, r.db).
		Exec("SELECT pg_advisory_xact_lock(hashtextextended('physical-backup:' || ?::text,0))", databaseID).
		Error
}

func (r *Repository) list(ctx context.Context, filter configListFilter) ([]backupConfigRow, error) {
	query := "SELECT " + backupConfigColumns + " FROM backups_configs WHERE TRUE"
	values := map[string]any{"limit": filter.limit}

	if filter.databaseIDs != nil {
		query += " AND database_id IN @databaseIDs"
		values["databaseIDs"] = filter.databaseIDs
	}

	if filter.storageID != nil {
		query += " AND storage_id = @storageID"
		values["storageID"] = *filter.storageID
	}

	if filter.afterID != nil {
		query += " AND database_id > @afterID"
		values["afterID"] = *filter.afterID
	}

	query += " ORDER BY database_id LIMIT @limit"

	var rows []backupConfigRow

	err := database.Conn(ctx, r.db).Raw(query, values).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list the backup configs: %w", err)
	}

	return rows, nil
}

func (r *Repository) listDatabaseIDsByStorage(ctx context.Context, storageID uuid.UUID) ([]uuid.UUID, error) {
	var databaseIDs []uuid.UUID

	err := database.Conn(ctx, r.db).
		Raw("SELECT database_id FROM backups_configs WHERE storage_id = ?", storageID).
		Scan(&databaseIDs).Error
	if err != nil {
		return nil, fmt.Errorf("list the databases that back up into the storage: %w", err)
	}

	return databaseIDs, nil
}

func (r *Repository) listLogical(ctx context.Context, databaseIDs []uuid.UUID) ([]LogicalBackupConfig, error) {
	return r.queryLogical(ctx, logicalConfigQuery+" WHERE backups_configs.database_id IN ?", databaseIDs)
}

func (r *Repository) listEnabledLogical(ctx context.Context) ([]LogicalBackupConfig, error) {
	return r.queryLogical(ctx, logicalConfigQuery+" WHERE backups_configs.is_backups_enabled")
}

func (r *Repository) listEnabledPhysical(ctx context.Context) ([]PhysicalBackupConfig, error) {
	return r.queryPhysical(ctx, physicalConfigQuery+" WHERE backups_configs.is_backups_enabled")
}

func (r *Repository) listPhysicalWithPendingRequest(ctx context.Context) ([]PhysicalBackupConfig, error) {
	return r.queryPhysical(ctx, physicalConfigQuery+" WHERE backups_configs.is_backups_enabled "+
		"AND backups_configs.storage_id IS NOT NULL "+
		"AND (force_full_requested_at IS NOT NULL OR force_incremental_requested_at IS NOT NULL)")
}

// delete removes the config and, through the module's own cascade, the row of its method.
func (r *Repository) delete(ctx context.Context, databaseID uuid.UUID) error {
	err := database.Conn(ctx, r.db).Exec("DELETE FROM backups_configs WHERE database_id = ?", databaseID).Error
	if err != nil {
		return fmt.Errorf("delete the backup config: %w", err)
	}

	return nil
}

func (r *Repository) updateForceFullRequestedAt(
	ctx context.Context,
	databaseID uuid.UUID,
	requestedAt time.Time,
) (bool, error) {
	return r.updateForceRequest(ctx, "force_full_requested_at", databaseID, &requestedAt, nil)
}

func (r *Repository) clearForceFullRequestedAt(
	ctx context.Context,
	databaseID uuid.UUID,
	requestedAt *time.Time,
) (bool, error) {
	return r.updateForceRequest(ctx, "force_full_requested_at", databaseID, nil, requestedAt)
}

func (r *Repository) updateForceIncrementalRequestedAt(
	ctx context.Context,
	databaseID uuid.UUID,
	requestedAt time.Time,
) (bool, error) {
	return r.updateForceRequest(ctx, "force_incremental_requested_at", databaseID, &requestedAt, nil)
}

func (r *Repository) clearForceIncrementalRequestedAt(
	ctx context.Context,
	databaseID uuid.UUID,
	requestedAt *time.Time,
) (bool, error) {
	return r.updateForceRequest(ctx, "force_incremental_requested_at", databaseID, nil, requestedAt)
}

// updateForceRequest writes value into the force request column. A clear given the request it served
// leaves a newer request in place.
func (r *Repository) updateForceRequest(
	ctx context.Context,
	column string,
	databaseID uuid.UUID,
	value *time.Time,
	servedRequest *time.Time,
) (bool, error) {
	// Only the two constant force-request column names reach this helper; values stay bound parameters.
	// nosemgrep: go.lang.security.audit.database.string-formatted-query.string-formatted-query
	query := "UPDATE backups_configs_physical SET " + column + " = ? WHERE database_id = ?"
	values := []any{value, databaseID}

	if servedRequest != nil {
		query += " AND " + column + " = ?"
		values = append(values, *servedRequest)
	}

	update := database.Conn(ctx, r.db).Exec(query, values...)
	if update.Error != nil {
		return false, fmt.Errorf("update %s: %w", column, update.Error)
	}

	return update.RowsAffected > 0, nil
}

func (r *Repository) upsertBackupConfig(ctx context.Context, method Method, config BackupConfig) error {
	err := database.Conn(ctx, r.db).Exec(`INSERT INTO backups_configs (database_id, method, is_backups_enabled,
		storage_id, last_selected_storage_id, encryption, notification_triggers) VALUES (?, ?, ?, ?, ?, ?, ?::text[])
		ON CONFLICT (database_id) DO UPDATE SET
			is_backups_enabled = EXCLUDED.is_backups_enabled,
			storage_id = EXCLUDED.storage_id,
			last_selected_storage_id = COALESCE(EXCLUDED.storage_id, backups_configs.last_selected_storage_id),
			encryption = EXCLUDED.encryption,
			notification_triggers = EXCLUDED.notification_triggers`,
		config.DatabaseID, method, config.IsBackupsEnabled, config.StorageID, config.StorageID, config.Encryption,
		toTriggerColumn(config.NotificationTriggers),
	).Error
	if err != nil {
		return fmt.Errorf("upsert the backup config: %w", err)
	}

	return nil
}

func (r *Repository) queryLogical(
	ctx context.Context,
	query string,
	values ...any,
) ([]LogicalBackupConfig, error) {
	var rows []logicalConfigRow

	err := database.Conn(ctx, r.db).Raw(query+" ORDER BY backups_configs.database_id", values...).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list the logical backup configs: %w", err)
	}

	configs := make([]LogicalBackupConfig, 0, len(rows))
	for _, row := range rows {
		configs = append(configs, row.toConfig())
	}

	return configs, nil
}

func (r *Repository) queryPhysical(
	ctx context.Context,
	query string,
	values ...any,
) ([]PhysicalBackupConfig, error) {
	var rows []physicalConfigRow

	err := database.Conn(ctx, r.db).Raw(query+" ORDER BY backups_configs.database_id", values...).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list the physical backup configs: %w", err)
	}

	configs := make([]PhysicalBackupConfig, 0, len(rows))
	for _, row := range rows {
		configs = append(configs, row.toConfig())
	}

	return configs, nil
}

func (r backupConfigRow) toConfig() BackupConfig {
	triggers := make([]NotificationTrigger, 0, len(r.NotificationTriggers))
	for _, trigger := range r.NotificationTriggers {
		triggers = append(triggers, NotificationTrigger(trigger))
	}

	return BackupConfig{
		StorageGeneration:    r.StorageGeneration,
		DatabaseID:           r.DatabaseID,
		IsBackupsEnabled:     r.IsBackupsEnabled,
		StorageID:            r.StorageID,
		Encryption:           r.Encryption,
		NotificationTriggers: triggers,
	}
}

func (r logicalConfigRow) toConfig() LogicalBackupConfig {
	return LogicalBackupConfig{
		BackupConfig: r.Common.toConfig(),
		Schedule:     r.Schedule.toSchedule(),
		Retention: LogicalRetention{
			Policy: r.RetentionPolicy,
			Period: r.RetentionPeriod,
			Count:  r.RetentionCount,
			Gfs:    r.RetentionGfs.toRetention(),
		},
		IsRetryIfFailed:     r.IsRetryIfFailed,
		MaxFailedTriesCount: r.MaxFailedTriesCount,
	}
}

// toConfig leaves BackupType empty: the database owns it, and the service reads it from there.
func (r physicalConfigRow) toConfig() PhysicalBackupConfig {
	var incrementalSchedule *schedule.Schedule
	if r.IncrementalSchedule.Type != "" {
		readSchedule := r.IncrementalSchedule.toSchedule()
		incrementalSchedule = &readSchedule
	}

	return PhysicalBackupConfig{
		BackupConfig:              r.Common.toConfig(),
		BackupType:                "",
		FullBackupSchedule:        r.FullSchedule.toSchedule(),
		IncrementalBackupSchedule: incrementalSchedule,
		Retention: PhysicalRetention{
			Policy:      r.RetentionPolicy,
			ChainsCount: r.ChainsRetentionCount,
			FullBackups: FullBackupsRetention{
				Policy: r.FullBackupsRetentionPolicy,
				Count:  r.FullBackupsRetentionCount,
				Gfs:    r.FullBackupsRetentionGfs.toRetention(),
			},
		},
		WalLagThresholdBytes:        r.WalLagThresholdBytes,
		ForceFullRequestedAt:        r.ForceFullRequestedAt,
		ForceIncrementalRequestedAt: r.ForceIncrementalRequestedAt,
	}
}

func (c scheduleColumns) toSchedule() schedule.Schedule {
	return schedule.Schedule{
		Type:           schedule.Type(c.Type),
		TimeOfDay:      c.TimeOfDay,
		Weekday:        c.Weekday,
		DayOfMonth:     c.DayOfMonth,
		CronExpression: c.CronExpression,
	}
}

func (c gfsColumns) toRetention() GfsRetention {
	return GfsRetention(c)
}

// toScheduleColumns writes a missing schedule as the empty type the incremental columns allow.
func toScheduleColumns(writtenSchedule *schedule.Schedule) scheduleColumns {
	if writtenSchedule == nil {
		return scheduleColumns{Type: "", TimeOfDay: nil, Weekday: nil, DayOfMonth: nil, CronExpression: nil}
	}

	return scheduleColumns{
		Type:           string(writtenSchedule.Type),
		TimeOfDay:      writtenSchedule.TimeOfDay,
		Weekday:        writtenSchedule.Weekday,
		DayOfMonth:     writtenSchedule.DayOfMonth,
		CronExpression: writtenSchedule.CronExpression,
	}
}

func toTriggerColumn(triggers []NotificationTrigger) pq.StringArray {
	column := make(pq.StringArray, 0, len(triggers))
	for _, trigger := range triggers {
		column = append(column, string(trigger))
	}

	return column
}

type PhysicalRequestsReset struct {
	DatabaseID  uuid.UUID
	RequestedAt time.Time
}
