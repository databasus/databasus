package restores_core

import (
	"time"

	"github.com/google/uuid"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
)

type Restore struct {
	ID     uuid.UUID     `json:"id"     gorm:"column:id;type:uuid;primaryKey"`
	Status RestoreStatus `json:"status" gorm:"column:status;type:text;not null"`

	BackupID uuid.UUID `json:"backupId" gorm:"column:backup_id;type:uuid;not null"`
	Backup   *backups_core_logical.LogicalBackup

	PostgresqlLogicalDatabase *PostgresqlTargetDatabase `json:"postgresqlDatabase" gorm:"-"`
	MysqlDatabase             *MysqlTargetDatabase      `json:"mysqlDatabase"      gorm:"-"`
	MariadbDatabase           *MariadbTargetDatabase    `json:"mariadbDatabase"    gorm:"-"`
	MongodbDatabase           *MongodbTargetDatabase    `json:"mongodbDatabase"    gorm:"-"`

	FailMessage *string `json:"failMessage" gorm:"column:fail_message"`

	RestoreDurationMs int64     `json:"restoreDurationMs" gorm:"column:restore_duration_ms;default:0"`
	CreatedAt         time.Time `json:"createdAt"         gorm:"column:created_at;default:now()"`
}
