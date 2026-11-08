// Package config keeps the backup config of a database: one config per database, of the backup method
// its type implies, with its defaults, its validation, its storage and its reactions to database and
// storage events.
package config

import (
	"slices"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages"
)

// Method is how a database is backed up. A physical PostgreSQL database is backed up physically and
// every other database logically.
type Method string

const (
	MethodLogical  Method = "LOGICAL"
	MethodPhysical Method = "PHYSICAL"
)

type Encryption string

const (
	EncryptionNone      Encryption = "NONE"
	EncryptionEncrypted Encryption = "ENCRYPTED"
)

func (e Encryption) isValid() bool {
	switch e {
	case EncryptionNone, EncryptionEncrypted:
		return true
	default:
		return false
	}
}

// NotificationTrigger is a backup outcome that sends a notification. A logical config takes only a
// failed and a successful backup.
type NotificationTrigger string

const (
	NotificationTriggerBackupFailed  NotificationTrigger = "BACKUP_FAILED"
	NotificationTriggerBackupSuccess NotificationTrigger = "BACKUP_SUCCESS"
	NotificationTriggerChainBroken   NotificationTrigger = "CHAIN_BROKEN"
	NotificationTriggerWalGap        NotificationTrigger = "WAL_GAP"
)

// BackupConfig holds what the configs of both methods share.
type BackupConfig struct {
	Cleanup           storages.CleanupSummary
	StorageGeneration int64
	DatabaseID        uuid.UUID
	IsBackupsEnabled  bool
	// StorageID is nil while the config names no storage.
	StorageID            *uuid.UUID
	Encryption           Encryption
	NotificationTriggers []NotificationTrigger
}

func (c BackupConfig) IsNotifyingOn(trigger NotificationTrigger) bool {
	return slices.Contains(c.NotificationTriggers, trigger)
}

// AnyBackupConfig is a LogicalBackupConfig or a PhysicalBackupConfig.
type AnyBackupConfig interface {
	GetBackupConfig() BackupConfig
	GetMethod() Method
}

type GfsRetention struct {
	Hours  int
	Days   int
	Weeks  int
	Months int
	Years  int
}

func (r GfsRetention) HasAnyPeriod() bool {
	return r.Hours > 0 || r.Days > 0 || r.Weeks > 0 || r.Months > 0 || r.Years > 0
}

func (r GfsRetention) IsZero() bool {
	return r == GfsRetention{}
}

func validateCommonFields(config BackupConfig, allowedTriggers []NotificationTrigger) error {
	if !config.Encryption.isValid() {
		return newInvalidFieldError("encryption", "must be NONE or ENCRYPTED")
	}

	for _, trigger := range config.NotificationTriggers {
		if !slices.Contains(allowedTriggers, trigger) {
			return newInvalidSettingError("notificationTriggers", "holds a trigger the backup method does not send")
		}
	}

	return nil
}

func NormalizeTriggers(triggers []NotificationTrigger) []NotificationTrigger {
	uniqueTriggers := make([]NotificationTrigger, 0, len(triggers))
	for _, trigger := range triggers {
		if !slices.Contains(uniqueTriggers, trigger) {
			uniqueTriggers = append(uniqueTriggers, trigger)
		}
	}

	return uniqueTriggers
}
