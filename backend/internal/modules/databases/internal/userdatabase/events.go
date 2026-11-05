package userdatabase

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/databases/internal/providers"
)

// Created is published inside the transaction that creates a database, a copy included.
type Created struct {
	DatabaseID uuid.UUID
}

// Copied is published inside the copy's transaction, after Created of the copy.
type Copied struct {
	OriginalDatabaseID uuid.UUID
	CopiedDatabaseID   uuid.UUID
}

// BackupTypeChanged is published inside the transaction that changes the physical backup type of a
// physical PostgreSQL database.
type BackupTypeChanged struct {
	DatabaseID    uuid.UUID
	OldBackupType providers.PhysicalBackupType
	NewBackupType providers.PhysicalBackupType
}

// Deleting is published inside the deletion's transaction, while the database and its settings still
// exist.
type Deleting struct {
	DatabaseID uuid.UUID
}

// The handlers run in the publisher's transaction, which ctx carries; an error rolls it back. An
// effect outside the database registers itself with database.AfterCommit.
type (
	CreatedHandler           func(ctx context.Context, event Created) error
	CopiedHandler            func(ctx context.Context, event Copied) error
	BackupTypeChangedHandler func(ctx context.Context, event BackupTypeChanged) error
	DeletingHandler          func(ctx context.Context, event Deleting) error
)
