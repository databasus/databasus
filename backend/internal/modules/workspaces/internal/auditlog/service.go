// Package auditlog serves the audit log views: the global one, a user's and a workspace's. It reads
// the events through lib/audit and adds the names of their workspaces, which only this module knows.
package auditlog

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces/internal/access"
	"databasus-backend/internal/modules/workspaces/internal/workspace"
)

type Deps struct {
	AuditLog   *audit.Log
	Authorizer *access.Authorizer
	Workspaces *workspace.Service
}

type Service struct {
	auditLog   *audit.Log
	authorizer *access.Authorizer
	workspaces *workspace.Service
}

func NewService(deps Deps) *Service {
	return &Service{auditLog: deps.AuditLog, authorizer: deps.Authorizer, workspaces: deps.Workspaces}
}

// ListEvents pages through the events of the view the filters choose, the newest first.
func (s *Service) ListEvents(ctx context.Context, actor users.Actor, query EventListQuery) (EventPage, error) {
	err := s.authorizeView(ctx, actor, query)
	if err != nil {
		return EventPage{}, err
	}

	limit := defaultEventPageLimit
	if query.Limit != nil {
		limit = *query.Limit
	}

	if limit < 1 || limit > maxEventPageLimit {
		return EventPage{}, apierror.NewInvalidFieldsProblem([]apierror.FieldError{{
			Field:  "limit",
			Detail: fmt.Sprintf("must be between 1 and %d", maxEventPageLimit),
		}})
	}

	after := ""
	if query.After != nil {
		after = *query.After
	}

	recordedPage, err := s.auditLog.ListEvents(ctx, audit.EventListQuery{
		WorkspaceID:  query.WorkspaceID,
		AuthorUserID: query.AuthorUserID,
		Limit:        limit,
		After:        after,
	})
	if err != nil {
		return EventPage{}, err
	}

	listedEvents, err := s.addWorkspaceNames(ctx, recordedPage.Events)
	if err != nil {
		return EventPage{}, err
	}

	page := EventPage{Events: listedEvents, NextCursor: nil}
	if recordedPage.NextCursor != "" {
		page.NextCursor = &recordedPage.NextCursor
	}

	return page, nil
}

func (s *Service) authorizeView(ctx context.Context, actor users.Actor, query EventListQuery) error {
	switch {
	case query.WorkspaceID != nil:
		return s.authorizer.Authorize(ctx, actor, *query.WorkspaceID, access.ActionView)
	case actor.IsAdmin():
		return nil
	case query.AuthorUserID == nil:
		return ErrGlobalViewForInstanceAdmins
	case *query.AuthorUserID != actor.UserID:
		return ErrAnotherUsersView
	default:
		return nil
	}
}

// addWorkspaceNames reads the workspaces of the whole page in one query.
func (s *Service) addWorkspaceNames(ctx context.Context, recordedEvents []audit.RecordedEvent) ([]ListedEvent, error) {
	workspaceIDs := []uuid.UUID{}
	isCollected := map[uuid.UUID]bool{}

	for _, event := range recordedEvents {
		if event.WorkspaceID != nil && !isCollected[*event.WorkspaceID] {
			isCollected[*event.WorkspaceID] = true
			workspaceIDs = append(workspaceIDs, *event.WorkspaceID)
		}
	}

	existingWorkspaces, err := s.workspaces.GetWorkspaces(ctx, workspaceIDs)
	if err != nil {
		return nil, err
	}

	namesByWorkspaceID := make(map[uuid.UUID]string, len(existingWorkspaces))
	for _, existingWorkspace := range existingWorkspaces {
		namesByWorkspaceID[existingWorkspace.ID] = existingWorkspace.Name
	}

	listedEvents := make([]ListedEvent, 0, len(recordedEvents))

	for _, event := range recordedEvents {
		listedEvent := ListedEvent{RecordedEvent: event, Workspace: nil}

		if event.WorkspaceID != nil {
			listedEvent.Workspace = &EventWorkspace{ID: *event.WorkspaceID, Name: nil}

			if name, isFound := namesByWorkspaceID[*event.WorkspaceID]; isFound {
				listedEvent.Workspace.Name = &name
			}
		}

		listedEvents = append(listedEvents, listedEvent)
	}

	return listedEvents, nil
}
