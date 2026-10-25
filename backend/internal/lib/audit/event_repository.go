package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

type eventRepository struct {
	db *gorm.DB
}

func (r *eventRepository) Insert(ctx context.Context, event Event, createdAt time.Time) error {
	var authorUserID *uuid.UUID
	var authorEmail, authorName *string

	if event.Author != nil {
		authorUserID = &event.Author.UserID
		authorEmail = &event.Author.Email
		authorName = &event.Author.Name
	}

	err := database.Conn(ctx, r.db).Exec(
		`INSERT INTO lib_audit_events
			(id, author_user_id, author_email, author_name, workspace_id, message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uuid.New(), authorUserID, authorEmail, authorName, event.WorkspaceID, event.Message, createdAt,
	).Error
	if err != nil {
		return fmt.Errorf("insert the audit event: %w", err)
	}

	return nil
}

func (r *eventRepository) DeleteCreatedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := database.Conn(ctx, r.db).Exec("DELETE FROM lib_audit_events WHERE created_at < ?", cutoff)
	if result.Error != nil {
		return 0, fmt.Errorf("delete the expired audit events: %w", result.Error)
	}

	return result.RowsAffected, nil
}
