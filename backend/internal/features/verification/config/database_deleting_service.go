package verification_config

import (
	"context"

	"databasus-backend/internal/modules/databases"
)

func (s *VerificationConfigService) OnDatabaseDeleting(ctx context.Context, event databases.DatabaseDeleting) error {
	return s.verificationConfigRepository.DeleteByDatabaseID(ctx, event.DatabaseID)
}
