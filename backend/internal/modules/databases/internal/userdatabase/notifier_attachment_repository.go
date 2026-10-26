package userdatabase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

// notifierAttachmentRepository keeps which notifiers a database uses in databases_notifiers.
type notifierAttachmentRepository struct {
	db *gorm.DB
}

type notifierAttachmentRow struct {
	DatabaseID uuid.UUID
	NotifierID uuid.UUID
}

// replace expects the caller's transaction in ctx. Inserting a row takes FOR KEY SHARE on the
// notifier's row, which keeps a transfer of that notifier waiting until the transaction ends.
func (r *notifierAttachmentRepository) replace(
	ctx context.Context,
	databaseID uuid.UUID,
	notifierIDs []uuid.UUID,
) error {
	conn := database.Conn(ctx, r.db)

	err := conn.Exec("DELETE FROM databases_notifiers WHERE database_id = ?", databaseID).Error
	if err != nil {
		return fmt.Errorf("delete the notifier attachments of the database: %w", err)
	}

	if len(notifierIDs) == 0 {
		return nil
	}

	err = conn.Exec(
		`INSERT INTO databases_notifiers (database_id, notifier_id)
		SELECT ?, notifier_id FROM unnest(?::uuid[]) AS attached (notifier_id)
		ON CONFLICT DO NOTHING`,
		databaseID, pq.Array(notifierIDs),
	).Error
	if database.IsForeignKeyViolation(err) {
		return ErrNotifierNotInWorkspace
	}

	if err != nil {
		return fmt.Errorf("insert the notifier attachments of the database: %w", err)
	}

	return nil
}

// listByDatabaseIDs leaves out the databases without attachments.
func (r *notifierAttachmentRepository) listByDatabaseIDs(
	ctx context.Context,
	databaseIDs []uuid.UUID,
) (map[uuid.UUID][]uuid.UUID, error) {
	var rows []notifierAttachmentRow

	err := database.Conn(ctx, r.db).Raw(
		"SELECT database_id, notifier_id FROM databases_notifiers WHERE database_id IN ? ORDER BY notifier_id",
		databaseIDs,
	).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list the notifier attachments: %w", err)
	}

	notifierIDsByDatabaseID := make(map[uuid.UUID][]uuid.UUID, len(databaseIDs))
	for _, row := range rows {
		notifierIDsByDatabaseID[row.DatabaseID] = append(notifierIDsByDatabaseID[row.DatabaseID], row.NotifierID)
	}

	return notifierIDsByDatabaseID, nil
}
