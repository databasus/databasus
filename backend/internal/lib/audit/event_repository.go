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

func (r *eventRepository) List(ctx context.Context, filter eventListFilter) ([]RecordedEvent, error) {
	query := `SELECT id, author_user_id, author_email, author_name, workspace_id, message, created_at
		FROM lib_audit_events
		WHERE TRUE`
	values := map[string]any{"limit": filter.limit}

	if filter.workspaceID != nil {
		query += " AND workspace_id = @workspaceID"
		values["workspaceID"] = *filter.workspaceID
	}

	if filter.authorUserID != nil {
		query += " AND author_user_id = @authorUserID"
		values["authorUserID"] = *filter.authorUserID
	}

	if filter.after != nil {
		query += " AND (created_at, id) < (@afterCreatedAt, @afterID)"
		values["afterCreatedAt"] = filter.after.createdAt
		values["afterID"] = filter.after.eventID
	}

	query += " ORDER BY created_at DESC, id DESC LIMIT @limit"

	var rows []eventRow

	err := database.Conn(ctx, r.db).Raw(query, values).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list the audit events: %w", err)
	}

	listedEvents := make([]RecordedEvent, 0, len(rows))
	for _, row := range rows {
		listedEvents = append(listedEvents, row.toRecordedEvent())
	}

	return listedEvents, nil
}

func (r *eventRepository) DeleteCreatedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := database.Conn(ctx, r.db).Exec("DELETE FROM lib_audit_events WHERE created_at < ?", cutoff)
	if result.Error != nil {
		return 0, fmt.Errorf("delete the expired audit events: %w", result.Error)
	}

	return result.RowsAffected, nil
}

type eventListFilter struct {
	workspaceID  *uuid.UUID
	authorUserID *uuid.UUID
	limit        int
	after        *eventCursor
}

type eventRow struct {
	ID           uuid.UUID
	AuthorUserID *uuid.UUID
	AuthorEmail  *string
	AuthorName   *string
	WorkspaceID  *uuid.UUID
	Message      string
	CreatedAt    time.Time
}

// toRecordedEvent relies on ck_lib_audit_events_author, which keeps the author columns all set or
// all empty.
func (r eventRow) toRecordedEvent() RecordedEvent {
	var author *Author

	if r.AuthorUserID != nil {
		author = &Author{UserID: *r.AuthorUserID, Email: *r.AuthorEmail, Name: *r.AuthorName}
	}

	return RecordedEvent{
		ID:          r.ID,
		Author:      author,
		WorkspaceID: r.WorkspaceID,
		Message:     r.Message,
		CreatedAt:   r.CreatedAt.UTC(),
	}
}
