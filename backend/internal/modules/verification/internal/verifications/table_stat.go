package verifications

import "github.com/google/uuid"

type TableStat struct {
	ID             uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	VerificationID uuid.UUID `gorm:"column:run_id;type:uuid;not null"`
	SchemaName     string    `gorm:"column:schema_name;type:text;not null"`
	Name           string    `gorm:"column:name;type:text;not null"`
	RowCount       int64     `gorm:"column:row_count;type:bigint;not null"`
}

func (TableStat) TableName() string {
	return "verification_run_table_stats"
}
