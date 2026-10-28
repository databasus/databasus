package physical

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

func (s *Service) GenerateRestoreToken(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	request *GenerateRestoreTokenRequest,
) (string, error) {
	if _, err := s.authorizeDatabase(ctx, actor, databaseID, workspaces.ActionView); err != nil {
		return "", err
	}

	if _, err := s.chains.ResolveRestoreSet(ctx, databaseID, request.TargetTime); err != nil {
		return "", err
	}

	return s.restoreTokenService.GenerateRestoreToken(ctx, databaseID, actor.UserID, request.TargetTime)
}

func (s *Service) GenerateBackupRestoreToken(
	ctx context.Context,
	actor users.Actor,
	backupID uuid.UUID,
) (string, error) {
	databaseID, _, found, err := s.resolveBackupDatabaseAndStatus(ctx, backupID)
	if err != nil {
		return "", err
	}
	if !found {
		return "", ErrBackupNotFound
	}

	if _, err := s.authorizeDatabase(ctx, actor, databaseID, workspaces.ActionView); err != nil {
		return "", err
	}

	if _, err := s.chains.ResolveRestoreSetForBackup(ctx, databaseID, backupID); err != nil {
		return "", err
	}

	return s.restoreTokenService.GenerateBackupRestoreToken(ctx, databaseID, actor.UserID, backupID)
}
