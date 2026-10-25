//go:build testhelpers

package audit_logs

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
)

const testWorkspaceEventLimit = 100

// ListTestWorkspaceEvents reads the newest events of the workspace for the tests of the old
// features, which have no contract client to call listAuditEvents with.
func ListTestWorkspaceEvents(ctx context.Context, workspaceID uuid.UUID) ([]audit.RecordedEvent, error) {
	page, err := getAuditLog().ListEvents(ctx, audit.EventListQuery{
		WorkspaceID:  &workspaceID,
		AuthorUserID: nil,
		Limit:        testWorkspaceEventLimit,
		After:        "",
	})
	if err != nil {
		return nil, err
	}

	return page.Events, nil
}
