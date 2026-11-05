package configs

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

type Repositories struct{ configs *configRepository }

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{configs: &configRepository{db: db}}
}

type (
	configRepository struct{ db *gorm.DB }
	configRow        struct {
		DatabaseID                     uuid.UUID
		IsScheduledVerificationEnabled bool
		ScheduleType                   VerificationScheduleType
		IntervalType                   schedule.Type
		TimeOfDay                      *string
		Weekday                        *int
		DayOfMonth                     *int
		CronExpression                 *string
		SendNotificationsOn            pq.StringArray `gorm:"type:text[]"`
	}
)

func (configRow) TableName() string { return "verification_configs" }
func (r *configRepository) get(ctx context.Context, id uuid.UUID) (Config, bool, error) {
	var rows []configRow
	err := database.Conn(ctx, r.db).Where("database_id = ?", id).Limit(1).Find(&rows).Error
	if err != nil {
		return Config{}, false, fmt.Errorf("get verification config: %w", err)
	}
	if len(rows) == 0 {
		return Config{}, false, nil
	}
	return rows[0].toConfig(), true, nil
}

func (r *configRepository) save(ctx context.Context, config Config, now time.Time) error {
	notifications := make([]string, len(config.NotificationTriggers))
	for index, trigger := range config.NotificationTriggers {
		notifications[index] = string(trigger)
	}
	err := database.Conn(ctx, r.db).
		Exec(`INSERT INTO verification_configs (database_id,is_scheduled_verification_enabled,schedule_type,interval_type,time_of_day,weekday,day_of_month,cron_expression,send_notifications_on,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT (database_id) DO UPDATE SET is_scheduled_verification_enabled=EXCLUDED.is_scheduled_verification_enabled,schedule_type=EXCLUDED.schedule_type,interval_type=EXCLUDED.interval_type,time_of_day=EXCLUDED.time_of_day,weekday=EXCLUDED.weekday,day_of_month=EXCLUDED.day_of_month,cron_expression=EXCLUDED.cron_expression,send_notifications_on=EXCLUDED.send_notifications_on,updated_at=EXCLUDED.updated_at`, config.DatabaseID, config.IsAutomaticVerificationEnabled, config.ScheduleType, config.Schedule.Type, config.Schedule.TimeOfDay, config.Schedule.Weekday, config.Schedule.DayOfMonth, config.Schedule.CronExpression, pq.StringArray(notifications), now, now).
		Error
	if err != nil {
		return fmt.Errorf("save verification config: %w", err)
	}
	return nil
}

func (r *configRepository) listEnabled(ctx context.Context) ([]Config, error) {
	rows := []configRow{}
	err := database.Conn(ctx, r.db).Where("is_scheduled_verification_enabled = ?", true).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list enabled verification configs: %w", err)
	}
	configs := make([]Config, len(rows))
	for index, row := range rows {
		configs[index] = row.toConfig()
	}
	return configs, nil
}

func (r *configRepository) delete(ctx context.Context, id uuid.UUID) error {
	err := database.Conn(ctx, r.db).Where("database_id = ?", id).Delete(&configRow{}).Error
	if err != nil {
		return fmt.Errorf("delete verification config: %w", err)
	}
	return nil
}

func (r configRow) toConfig() Config {
	notifications := make([]VerificationNotificationType, len(r.SendNotificationsOn))
	for index, trigger := range r.SendNotificationsOn {
		notifications[index] = VerificationNotificationType(trigger)
	}
	return Config{
		DatabaseID:                     r.DatabaseID,
		IsAutomaticVerificationEnabled: r.IsScheduledVerificationEnabled,
		ScheduleType:                   r.ScheduleType,
		Schedule: schedule.Schedule{
			Type:           r.IntervalType,
			TimeOfDay:      r.TimeOfDay,
			Weekday:        r.Weekday,
			DayOfMonth:     r.DayOfMonth,
			CronExpression: r.CronExpression,
		},
		NotificationTriggers: notifications,
	}
}
