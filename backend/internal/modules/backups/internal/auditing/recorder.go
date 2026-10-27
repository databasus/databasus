// Package auditing writes the audit events of the backups module with a snapshot of their author.
package auditing

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/modules/users"
)

type Recorder struct {
	auditLog *audit.Log
	users    users.Module
}

func NewRecorder(auditLog *audit.Log, usersModule users.Module) Recorder {
	return Recorder{auditLog: auditLog, users: usersModule}
}

// Record expects the transaction of the action in ctx, so the event commits with it.
func (r Recorder) Record(ctx context.Context, actor users.Actor, workspaceID uuid.UUID, message string) error {
	return r.RecordByUser(ctx, actor.UserID, workspaceID, message)
}

// RecordByUser serves an action whose request carries no session, such as a download by token, where
// the token names the user.
func (r Recorder) RecordByUser(ctx context.Context, userID, workspaceID uuid.UUID, message string) error {
	author, isFound, err := r.users.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	if !isFound {
		return errActorNotFound
	}

	return r.auditLog.Record(ctx, audit.Event{
		Author:      author.GetAuditAuthor(),
		WorkspaceID: &workspaceID,
		Message:     message,
	})
}
