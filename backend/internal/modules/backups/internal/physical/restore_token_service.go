package physical

import (
	"context"

	"databasus-backend/internal/modules/backups/internal/physical/restorestream"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

func (s *Service) IssueRestoreToken(
	ctx context.Context,
	actor users.Actor,
	subject RestoreTokenSubject,
) (restorestream.IssuedToken, error) {
	databaseID := subject.DatabaseID
	if subject.BackupID != nil {
		resolvedDatabaseID, _, isFound, err := s.resolveBackupDatabaseAndStatus(ctx, *subject.BackupID)
		if err != nil {
			return restorestream.IssuedToken{}, err
		}
		if !isFound {
			return restorestream.IssuedToken{}, ErrBackupNotFound
		}
		databaseID = resolvedDatabaseID
	}

	if _, err := s.authorizeDatabase(ctx, actor, databaseID, workspaces.ActionView); err != nil {
		if subject.BackupID != nil {
			err = toBackupAccessProblem(err)
		}
		return restorestream.IssuedToken{}, err
	}

	if subject.BackupID != nil {
		if _, err := s.chains.ResolveRestoreSetForBackup(ctx, databaseID, *subject.BackupID); err != nil {
			return restorestream.IssuedToken{}, toRestoreProblem(err)
		}
	} else {
		if _, err := s.chains.ResolveRestoreSet(ctx, databaseID, subject.TargetTime); err != nil {
			return restorestream.IssuedToken{}, toRestoreProblem(err)
		}
	}

	return s.restoreTokenService.IssueToken(
		ctx,
		restorestream.Token{
			DatabaseID: databaseID,
			UserID:     actor.UserID,
			BackupID:   subject.BackupID,
			TargetTime: subject.TargetTime,
		},
	)
}
