package physical

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

func (s *Service) GetBackups(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	request *GetPhysicalBackupsRequest,
) (*GetPhysicalBackupsResponse, error) {
	if _, err := s.authorizeDatabase(ctx, actor, databaseID, workspaces.ActionView); err != nil {
		return nil, err
	}

	limit := 50
	if request.Limit != nil {
		limit = *request.Limit
	}
	if limit < 1 || limit > 1000 {
		return nil, errInvalidPageLimit
	}
	filter := toBackupListFilter(request)

	after, err := decodePhysicalListCursor(request.Cursor)
	if err != nil {
		return nil, err
	}

	rows, err := s.chains.ListBackups(
		ctx,
		catalog.BackupListQuery{DatabaseID: databaseID, Filter: filter, After: after, Limit: limit + 1},
	)
	if err != nil {
		return nil, err
	}

	totalUsageMb, err := s.chains.GetTotalUsageMBByDatabase(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	hasNextPage := len(rows) > limit
	if hasNextPage {
		rows = rows[:limit]
	}

	items := make([]PhysicalBackupListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, rowToListItem(row))
	}

	response := &GetPhysicalBackupsResponse{
		Backups:      items,
		TotalUsageMb: totalUsageMb,
	}

	if hasNextPage {
		lastRow := rows[len(rows)-1]

		response.NextCursor, err = encodePhysicalListCursor(catalog.BackupListPosition{
			SortTime: lastRow.SortTime,
			ID:       lastRow.ID,
		})
		if err != nil {
			return nil, err
		}
	}

	return response, nil
}

func rowToListItem(row catalog.BackupListRow) PhysicalBackupListItem {
	return PhysicalBackupListItem{
		ID:                        row.ID,
		Type:                      catalog.BackupType(row.Type),
		Status:                    row.Status,
		TimelineID:                row.TimelineID,
		StartLSN:                  row.StartLSN,
		StopLSN:                   row.StopLSN,
		RootFullBackupID:          row.RootFullBackupID,
		ParentIncrementalBackupID: row.ParentIncrementalBackupID,
		FromWalFilename:           row.FromWalFilename,
		ToWalFilename:             row.ToWalFilename,
		WalSegmentsCount:          row.WalSegmentsCount,
		HasGapBefore:              row.HasGapBefore,
		SizeMb:                    row.SizeMb,
		FailMessage:               row.FailMessage,
		CreatedAt:                 row.CreatedAt,
		CompletedAt:               row.CompletedAt,
	}
}

func toBackupListFilter(
	request *GetPhysicalBackupsRequest,
) catalog.BackupListFilter {
	filter := catalog.BackupListFilter{
		Statuses:   request.Statuses,
		BeforeDate: request.BeforeDate,
	}

	for _, backupType := range request.Types {
		filter.Types = append(filter.Types, string(backupType))
	}

	return filter
}
