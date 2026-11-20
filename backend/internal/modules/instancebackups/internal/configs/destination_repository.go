package configs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

type destinationRepository struct {
	db *gorm.DB
}

func (r *destinationRepository) list(ctx context.Context) ([]Destination, error) {
	destinations := []Destination{}

	err := database.Conn(ctx, r.db).Order("storage_id").Find(&destinations).Error
	if err != nil {
		return nil, fmt.Errorf("list the Databasus backup destinations: %w", err)
	}

	return destinations, nil
}

func (r *destinationRepository) insert(ctx context.Context, destination Destination) error {
	err := database.Conn(ctx, r.db).Exec(`INSERT INTO instancebackups_destinations
		(storage_id, selection_generation, selected_at, outstanding_due_at) VALUES (?, ?, ?, ?)`,
		destination.StorageID, destination.SelectionGeneration, destination.SelectedAt, destination.OutstandingDueAt,
	).Error
	if err != nil {
		return fmt.Errorf("insert the Databasus backup destination: %w", err)
	}

	return nil
}

func (r *destinationRepository) deleteByStorageIDs(ctx context.Context, storageIDs []uuid.UUID) error {
	if len(storageIDs) == 0 {
		return nil
	}

	err := database.Conn(ctx, r.db).
		Exec("DELETE FROM instancebackups_destinations WHERE storage_id IN ?", storageIDs).Error
	if err != nil {
		return fmt.Errorf("delete the Databasus backup destinations: %w", err)
	}

	return nil
}

func (r *destinationRepository) hasAnyOf(ctx context.Context, storageIDs []uuid.UUID) (bool, error) {
	var isSelected bool

	err := database.Conn(ctx, r.db).Raw(
		"SELECT EXISTS (SELECT 1 FROM instancebackups_destinations WHERE storage_id IN ?)", storageIDs,
	).Scan(&isSelected).Error
	if err != nil {
		return false, fmt.Errorf("check the Databasus backup destinations: %w", err)
	}

	return isSelected, nil
}

// getCurrentForUpdate locks the destination while it is still the selection the generation names,
// so a publication and a deselection or reselection never interleave.
func (r *destinationRepository) getCurrentForUpdate(
	ctx context.Context,
	storageID uuid.UUID,
	generation int64,
) (Destination, bool, error) {
	var destinations []Destination

	err := database.Conn(ctx, r.db).Raw(`SELECT * FROM instancebackups_destinations
		WHERE storage_id = ? AND selection_generation = ? FOR UPDATE`, storageID, generation,
	).Scan(&destinations).Error
	if err != nil {
		return Destination{}, false, fmt.Errorf("lock the Databasus backup destination: %w", err)
	}

	if len(destinations) == 0 {
		return Destination{}, false, nil
	}

	return destinations[0], true, nil
}

func (r *destinationRepository) recordSuccess(ctx context.Context, outcome DeliveryOutcome) error {
	err := database.Conn(ctx, r.db).Exec(`UPDATE instancebackups_destinations
		SET last_successful_snapshot_at = GREATEST(COALESCE(last_successful_snapshot_at, ?), ?),
			last_error = NULL,
			outstanding_due_at = CASE WHEN outstanding_due_at <= ? THEN NULL ELSE outstanding_due_at END
		WHERE storage_id = ?`,
		outcome.SnapshotAt, outcome.SnapshotAt, outcome.SnapshotAt, outcome.StorageID,
	).Error
	if err != nil {
		return fmt.Errorf("record the Databasus backup copy: %w", err)
	}

	return nil
}

func (r *destinationRepository) recordFailure(ctx context.Context, storageID uuid.UUID, lastError string) error {
	err := database.Conn(ctx, r.db).Exec(
		"UPDATE instancebackups_destinations SET last_error = ? WHERE storage_id = ?", lastError, storageID,
	).Error
	if err != nil {
		return fmt.Errorf("record the Databasus backup copy failure: %w", err)
	}

	return nil
}

// markOutstanding records the scheduled occurrence the destination owes a copy for, keeping an
// earlier occurrence that is still uncovered.
func (r *destinationRepository) markOutstanding(ctx context.Context, storageID uuid.UUID, dueAt time.Time) error {
	err := database.Conn(ctx, r.db).Exec(
		"UPDATE instancebackups_destinations SET outstanding_due_at = COALESCE(outstanding_due_at, ?) WHERE storage_id = ?",
		dueAt, storageID,
	).
		Error
	if err != nil {
		return fmt.Errorf("record the owed Databasus backup copies: %w", err)
	}

	return nil
}
