package chains

import (
	"context"

	"databasus-backend/internal/modules/databases"
)

func (s *Service) OnDatabaseDeleting(ctx context.Context, event databases.DatabaseDeleting) error {
	references, err := s.repository.GetDatabaseFileReferences(ctx, event.DatabaseID)
	if err != nil {
		return err
	}
	if err := s.storages.RequestFileDeletions(ctx, references); err != nil {
		return err
	}
	return s.repository.DeleteDatabaseRows(ctx, event.DatabaseID)
}
