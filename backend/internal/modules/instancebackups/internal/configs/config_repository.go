package configs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

const configColumns = "schedule_type, schedule_time_of_day, schedule_weekday, schedule_day_of_month, " +
	"schedule_cron_expression, retention_days, protection_state, next_selection_generation, is_run_requested, " +
	"setup_deferred_until, acknowledged_master_key_sha256, master_key_acknowledged_at, " +
	"recovery_paused_operation_id, recovery_paused_at, archive_namespace"

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// services never hold it.
type Repositories struct {
	configs      *configRepository
	destinations *destinationRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{configs: &configRepository{db: db}, destinations: &destinationRepository{db: db}}
}

type configRepository struct {
	db *gorm.DB
}

func (r *configRepository) get(ctx context.Context) (Config, error) {
	return r.read(ctx, "")
}

// getForUpdate serializes every change of the settings and of the selected destinations.
func (r *configRepository) getForUpdate(ctx context.Context) (Config, error) {
	return r.read(ctx, " FOR UPDATE")
}

func (r *configRepository) read(ctx context.Context, lock string) (Config, error) {
	var configs []Config

	err := database.Conn(ctx, r.db).
		Raw("SELECT " + configColumns + " FROM instancebackups_configs WHERE id" + lock).
		Scan(&configs).Error
	if err != nil {
		return Config{}, fmt.Errorf("get the Databasus backup settings: %w", err)
	}

	if len(configs) == 0 {
		return Config{}, errConfigMissing
	}

	return configs[0], nil
}

func (r *configRepository) updateScheduleAndRetention(ctx context.Context, config Config) error {
	err := database.Conn(ctx, r.db).Exec(`UPDATE instancebackups_configs SET
		schedule_type = ?, schedule_time_of_day = ?, schedule_weekday = ?, schedule_day_of_month = ?,
		schedule_cron_expression = ?, retention_days = ?
		WHERE id`,
		config.ScheduleType, config.ScheduleTimeOfDay, config.ScheduleWeekday, config.ScheduleDayOfMonth,
		config.ScheduleCronExpression, config.RetentionDays,
	).Error
	if err != nil {
		return fmt.Errorf("update the Databasus backup schedule: %w", err)
	}

	return nil
}

func (r *configRepository) updateProtectionState(ctx context.Context, state ProtectionState) error {
	err := database.Conn(ctx, r.db).
		Exec("UPDATE instancebackups_configs SET protection_state = ? WHERE id", state).Error
	if err != nil {
		return fmt.Errorf("update the Databasus backup protection state: %w", err)
	}

	return nil
}

func (r *configRepository) updateNextSelectionGeneration(ctx context.Context, generation int64) error {
	err := database.Conn(ctx, r.db).
		Exec("UPDATE instancebackups_configs SET next_selection_generation = ? WHERE id", generation).Error
	if err != nil {
		return fmt.Errorf("update the next Databasus backup selection generation: %w", err)
	}

	return nil
}

func (r *configRepository) updateIsRunRequested(ctx context.Context, isRunRequested bool) error {
	err := database.Conn(ctx, r.db).
		Exec("UPDATE instancebackups_configs SET is_run_requested = ? WHERE id", isRunRequested).Error
	if err != nil {
		return fmt.Errorf("update the Databasus backup run request: %w", err)
	}

	return nil
}

func (r *configRepository) updateSetupDeferredUntil(ctx context.Context, deferredUntil *time.Time) error {
	err := database.Conn(ctx, r.db).
		Exec("UPDATE instancebackups_configs SET setup_deferred_until = ? WHERE id", deferredUntil).Error
	if err != nil {
		return fmt.Errorf("update the Databasus backup setup reminder: %w", err)
	}

	return nil
}

// updateMasterKeyAcknowledgment sets both columns in one statement, because the schema requires
// them to be set together.
func (r *configRepository) updateMasterKeyAcknowledgment(
	ctx context.Context,
	masterKeySha256 *string,
	acknowledgedAt *time.Time,
) error {
	err := database.Conn(ctx, r.db).Exec(`UPDATE instancebackups_configs
		SET acknowledged_master_key_sha256 = ?, master_key_acknowledged_at = ? WHERE id`,
		masterKeySha256, acknowledgedAt,
	).Error
	if err != nil {
		return fmt.Errorf("update the master key acknowledgment: %w", err)
	}

	return nil
}

// countRows reads how many settings rows a database of any schema version holds, without creating
// the table.
func (r *configRepository) countRows(ctx context.Context) (int64, error) {
	isPresent, err := database.HasTable(ctx, r.db, "public.instancebackups_configs")
	if err != nil || !isPresent {
		return 0, err
	}

	var count int64

	err = database.Conn(ctx, r.db).Raw("SELECT count(*) FROM instancebackups_configs").Scan(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count the Databasus backup settings: %w", err)
	}

	return count, nil
}

// updateRecoveryPause sets both columns in one statement, because the schema requires them to be
// set together.
func (r *configRepository) updateRecoveryPause(
	ctx context.Context,
	operationID *uuid.UUID,
	pausedAt *time.Time,
) error {
	err := database.Conn(ctx, r.db).Exec(`UPDATE instancebackups_configs
		SET recovery_paused_operation_id = ?, recovery_paused_at = ? WHERE id`,
		operationID, pausedAt,
	).Error
	if err != nil {
		return fmt.Errorf("update the recovery pause: %w", err)
	}

	return nil
}
