package database

import (
	"context"
	"database/sql"
	"fmt"

	"gorm.io/gorm"
)

// ExportedSnapshot is a consistent view of the database that another session, such as pg_dump,
// can join, together with the migration versions in effect in that same view.
type ExportedSnapshot struct {
	ID                       string
	AppliedMigrationVersions []int64
}

// WithExportedSnapshot opens a read-only repeatable-read transaction, exports its snapshot and keeps
// it open while use runs, so a dump that joins the snapshot sees exactly the migration history read
// here. The transaction ends when use returns.
func WithExportedSnapshot(ctx context.Context, db *gorm.DB, use func(snapshot ExportedSnapshot) error) error {
	options := &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}

	return db.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		transactionCtx := context.WithValue(ctx, transactionKey{}, transaction)

		var snapshotIDs []string
		if err := transaction.Raw("SELECT pg_export_snapshot()").Scan(&snapshotIDs).Error; err != nil {
			return fmt.Errorf("export the snapshot: %w", err)
		}

		if len(snapshotIDs) != 1 {
			return fmt.Errorf("export the snapshot: %d rows returned", len(snapshotIDs))
		}

		appliedVersions, err := ReadAppliedMigrationVersions(transactionCtx, db)
		if err != nil {
			return err
		}

		return use(ExportedSnapshot{ID: snapshotIDs[0], AppliedMigrationVersions: appliedVersions})
	}, options)
}
