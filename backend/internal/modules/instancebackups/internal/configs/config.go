// Package configs owns the Databasus backup settings: the shared schedule and retention, the
// selected destinations with their selection generations, the protection lifecycle, the setup
// reminder and the acknowledgment that root admin saved the current master key.
package configs

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/schedule"
)

// ProtectionState is the persisted lifecycle of protection. Health combines it with the outcomes
// of each destination.
type ProtectionState string

const (
	// ProtectionStateNotConfigured is a new installation that never selected a destination.
	ProtectionStateNotConfigured ProtectionState = "NOT_CONFIGURED"
	// ProtectionStateInitializing waits for the first successful copy after the first selection.
	ProtectionStateInitializing ProtectionState = "INITIALIZING"
	// ProtectionStateProtected had a successful copy since destinations were selected.
	ProtectionStateProtected ProtectionState = "PROTECTED"
	// ProtectionStateStopped lost its last destination.
	ProtectionStateStopped ProtectionState = "STOPPED"
	// ProtectionStateResuming waits for the first successful copy after a stop.
	ProtectionStateResuming ProtectionState = "RESUMING"
)

const (
	DefaultRetentionDays = 30
)

// Config is the singleton row of the settings.
type Config struct {
	ScheduleType                schedule.Type   `gorm:"column:schedule_type"`
	ScheduleTimeOfDay           *string         `gorm:"column:schedule_time_of_day"`
	ScheduleWeekday             *int            `gorm:"column:schedule_weekday"`
	ScheduleDayOfMonth          *int            `gorm:"column:schedule_day_of_month"`
	ScheduleCronExpression      *string         `gorm:"column:schedule_cron_expression"`
	RetentionDays               int             `gorm:"column:retention_days"`
	ProtectionState             ProtectionState `gorm:"column:protection_state"`
	NextSelectionGeneration     int64           `gorm:"column:next_selection_generation"`
	IsRunRequested              bool            `gorm:"column:is_run_requested"`
	AcknowledgedMasterKeySha256 *string         `gorm:"column:acknowledged_master_key_sha256"`
	MasterKeyAcknowledgedAt     *time.Time      `gorm:"column:master_key_acknowledged_at"`
	// RecoveryPausedOperationID is the recovery that published this installation and keeps it
	// paused until root admin activates it.
	RecoveryPausedOperationID *uuid.UUID `gorm:"column:recovery_paused_operation_id"`
	RecoveryPausedAt          *time.Time `gorm:"column:recovery_paused_at"`
	// ArchiveNamespace prefixes the file names of this installation's Databasus backups, so
	// retention never touches another installation's files in a shared storage. It is random,
	// so IsDefault leaves it out.
	ArchiveNamespace uuid.UUID `gorm:"column:archive_namespace"`
}

func (c Config) GetSchedule() schedule.Schedule {
	return schedule.Schedule{
		Type:           c.ScheduleType,
		TimeOfDay:      c.ScheduleTimeOfDay,
		Weekday:        c.ScheduleWeekday,
		DayOfMonth:     c.ScheduleDayOfMonth,
		CronExpression: c.ScheduleCronExpression,
	}
}

// IsDefault reports whether the row still holds the values a migration creates it with, so an
// installation whose owner never changed them counts as automatically initialized.
func (c Config) IsDefault() bool {
	return c.ScheduleType == schedule.TypeHourly &&
		c.ScheduleTimeOfDay == nil &&
		c.ScheduleWeekday == nil &&
		c.ScheduleDayOfMonth == nil &&
		c.ScheduleCronExpression == nil &&
		c.RetentionDays == DefaultRetentionDays &&
		c.ProtectionState == ProtectionStateNotConfigured &&
		c.NextSelectionGeneration == 1 &&
		!c.IsRunRequested &&
		c.AcknowledgedMasterKeySha256 == nil &&
		c.MasterKeyAcknowledgedAt == nil &&
		c.RecoveryPausedOperationID == nil &&
		c.RecoveryPausedAt == nil
}

func (c Config) IsRecoveryPaused() bool {
	return c.RecoveryPausedOperationID != nil
}

// Settings are the fields root admin edits.
type Settings struct {
	Schedule      schedule.Schedule
	RetentionDays int
	StorageIDs    []uuid.UUID
}

// Destination is one current selection of an existing storage. It owns no provider settings.
type Destination struct {
	StorageID                uuid.UUID  `gorm:"column:storage_id"`
	SelectionGeneration      int64      `gorm:"column:selection_generation"`
	SelectedAt               time.Time  `gorm:"column:selected_at"`
	OutstandingDueAt         *time.Time `gorm:"column:outstanding_due_at"`
	LastSuccessfulSnapshotAt *time.Time `gorm:"column:last_successful_snapshot_at"`
	LastError                *string    `gorm:"column:last_error"`
}

func (Destination) TableName() string {
	return "instancebackups_destinations"
}

// GetOutstandingDueAt is the earliest scheduled occurrence the destination still owes a copy for. A
// due occurrence nothing recorded yet, such as while no job runs, counts from the destination's last
// successful snapshot or its selection.
func (d Destination) GetOutstandingDueAt(destinationSchedule schedule.Schedule, now time.Time) *time.Time {
	if d.OutstandingDueAt != nil {
		return d.OutstandingDueAt
	}

	nextOccurrence := destinationSchedule.GetNextOccurrence(d.getCoveredUntil())
	if nextOccurrence.IsZero() || nextOccurrence.After(now) {
		return nil
	}

	return &nextOccurrence
}

func (d Destination) getCoveredUntil() time.Time {
	if d.LastSuccessfulSnapshotAt != nil && d.LastSuccessfulSnapshotAt.After(d.SelectedAt) {
		return *d.LastSuccessfulSnapshotAt
	}

	return d.SelectedAt
}

// DeliveryOutcome is the result of one copy at the destination it was made for. A nil LastError is a
// success, whose snapshot covers every occurrence the destination owed up to SnapshotAt.
type DeliveryOutcome struct {
	StorageID           uuid.UUID
	SelectionGeneration int64
	SnapshotAt          time.Time
	LastError           *string
}
