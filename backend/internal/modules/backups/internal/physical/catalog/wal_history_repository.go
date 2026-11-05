package catalog

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

type WalHistoryRepository struct{ db *gorm.DB }

func NewWalHistoryRepository(db *gorm.DB) *WalHistoryRepository { return &WalHistoryRepository{db: db} }

func (r *WalHistoryRepository) Insert(ctx context.Context,
	file *WalHistoryFile, now time.Time,
) error {
	if file.DatabaseID == uuid.Nil || file.StorageID == uuid.Nil {
		return errors.New("database ID and storage ID are required")
	}

	if file.ID == uuid.Nil {
		file.ID = uuid.New()
	}

	if file.CreatedAt.IsZero() {
		file.CreatedAt = now
	}

	return database.Conn(ctx, r.db).Create(file).Error
}

func (r *WalHistoryRepository) FindByDatabaseTimeline(ctx context.Context, databaseID uuid.UUID,
	timelineID int,
) (*WalHistoryFile, bool, error) {
	var file WalHistoryFile

	err := database.Conn(ctx, r.db).
		Where("database_id = ? AND timeline_id = ?", databaseID, timelineID).
		First(&file).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return &file, true, nil
}

func (r *WalHistoryRepository) FindAllByDatabase(ctx context.Context, databaseID uuid.UUID,
) ([]*WalHistoryFile, error) {
	var files []*WalHistoryFile

	if err := database.Conn(ctx, r.db).
		Where("database_id = ?", databaseID).
		Order("timeline_id ASC").
		Find(&files).Error; err != nil {
		return nil, err
	}

	return files, nil
}

func (r *WalHistoryRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	return database.Conn(ctx, r.db).Delete(&WalHistoryFile{}, "id = ?", id).Error
}

func (r *WalHistoryRepository) FindAllByDatabaseTimeline(
	ctx context.Context,
	databaseID uuid.UUID,
	timelineID int,
) ([]*WalHistoryFile, error) {
	var files []*WalHistoryFile
	err := database.Conn(ctx, r.db).
		Where("database_id = ? AND timeline_id = ?", databaseID, timelineID).
		Find(&files).
		Error
	return files, err
}
