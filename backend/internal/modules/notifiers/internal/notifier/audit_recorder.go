package notifier

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
	author, isFound, err := r.users.GetUser(ctx, actor.UserID)
	if err != nil {
		return err
	}

	if !isFound {
		return ErrActorNotFound
	}

	return r.auditLog.Record(ctx, audit.Event{
		Author:      author.GetAuditAuthor(),
		WorkspaceID: &workspaceID,
		Message:     message,
	})
}
