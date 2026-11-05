package userdatabase

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/modules/users"
)

// auditRecorder writes the audit events of the package with a snapshot of their author.
type auditRecorder struct {
	auditLog *audit.Log
	users    users.Module
}

func (r auditRecorder) record(ctx context.Context, actor users.Actor, workspaceID uuid.UUID, message string) error {
	author, err := r.getAuthor(ctx, actor)
	if err != nil {
		return err
	}

	return r.auditLog.Record(ctx, audit.Event{Author: author, WorkspaceID: &workspaceID, Message: message})
}

func (r auditRecorder) getAuthor(ctx context.Context, actor users.Actor) (*audit.Author, error) {
	user, isFound, err := r.users.GetUser(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}

	if !isFound {
		return nil, ErrActorNotFound
	}

	return user.GetAuditAuthor(), nil
}
