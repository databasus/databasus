package workspace

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces/internal/access"
)

const maxWorkspaceNameLength = 255

type Deps struct {
	Repositories Repositories
	Transactor   *database.Transactor
	Clock        clock.Source
	AuditLog     *audit.Log
	Users        users.Module
	Authorizer   *access.Authorizer
}

type Service struct {
	repository       *workspaceRepository
	transactor       *database.Transactor
	clock            clock.Source
	auditLog         *audit.Log
	users            users.Module
	authorizer       *access.Authorizer
	deletingHandlers []DeletingHandler
}

func NewService(deps Deps) *Service {
	return &Service{
		repository:       deps.Repositories.workspaces,
		transactor:       deps.Transactor,
		clock:            deps.Clock,
		auditLog:         deps.AuditLog,
		users:            deps.Users,
		authorizer:       deps.Authorizer,
		deletingHandlers: nil,
	}
}

// SubscribeToDeleting adds a handler after the ones subscribed before it. internal/app subscribes
// while it builds the application, before any request runs.
func (s *Service) SubscribeToDeleting(handler DeletingHandler) {
	s.deletingHandlers = append(s.deletingHandlers, handler)
}

// GetWorkspaces leaves out the IDs that match no workspace.
func (s *Service) GetWorkspaces(ctx context.Context, workspaceIDs []uuid.UUID) ([]Workspace, error) {
	if len(workspaceIDs) == 0 {
		return []Workspace{}, nil
	}

	return s.repository.listByIDs(ctx, workspaceIDs)
}

// ListWorkspaces pages through the workspaces the actor is a member of, or through every workspace
// for an instance admin, ordered by name.
func (s *Service) ListWorkspaces(
	ctx context.Context,
	actor users.Actor,
	query WorkspaceListQuery,
) (WorkspacePage, error) {
	limit := defaultWorkspacePageLimit
	if query.Limit != nil {
		limit = *query.Limit
	}

	if limit < 1 || limit > maxWorkspacePageLimit {
		return WorkspacePage{}, apierror.NewInvalidFieldsProblem([]apierror.FieldError{{
			Field:  "limit",
			Detail: fmt.Sprintf("must be between 1 and %d", maxWorkspacePageLimit),
		}})
	}

	var after *database.NameCursor

	if query.After != nil {
		cursor, err := database.DecodeNameCursor(*query.After)
		if err != nil {
			return WorkspacePage{}, err
		}

		after = &cursor
	}

	listedWorkspaces, err := s.repository.list(ctx, workspaceListFilter{
		userID:           actor.UserID,
		isEveryWorkspace: actor.IsAdmin(),
		limit:            limit + 1,
		after:            after,
	})
	if err != nil {
		return WorkspacePage{}, err
	}

	page := WorkspacePage{Workspaces: listedWorkspaces[:min(len(listedWorkspaces), limit)], NextCursor: nil}

	if len(listedWorkspaces) > limit {
		lastWorkspace := listedWorkspaces[limit-1]
		nextCursor := database.NameCursor{Name: lastWorkspace.Name, ID: lastWorkspace.ID}.Encode()
		page.NextCursor = &nextCursor
	}

	return page, nil
}

// CreateWorkspace makes the actor the owner of the new workspace. A user who is not an instance
// admin needs the instance settings to let members create workspaces.
func (s *Service) CreateWorkspace(ctx context.Context, actor users.Actor, name string) (ListedWorkspace, error) {
	err := validateName(name)
	if err != nil {
		return ListedWorkspace{}, err
	}

	canCreateWorkspaces, err := s.users.CanCreateWorkspaces(ctx, actor)
	if err != nil {
		return ListedWorkspace{}, err
	}

	if !canCreateWorkspaces {
		return ListedWorkspace{}, ErrCreationNotAllowed
	}

	ownerRole := access.RoleOwner
	createdWorkspace := ListedWorkspace{
		ID: uuid.New(), Name: name, CreatedAt: s.clock.Now(),
		CurrentUserRole: &ownerRole,
	}

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		err := s.repository.insert(ctx, createdWorkspace.Workspace)
		if err != nil {
			return err
		}

		err = s.repository.insertOwner(ctx, createdWorkspace.ID, actor.UserID, createdWorkspace.CreatedAt)
		if err != nil {
			return err
		}

		return s.recordAuditEvent(ctx, actor, createdWorkspace.ID, "Workspace created: "+name)
	})
	if err != nil {
		return ListedWorkspace{}, err
	}

	return createdWorkspace, nil
}

func (s *Service) GetWorkspace(ctx context.Context, actor users.Actor, workspaceID uuid.UUID) (ListedWorkspace, error) {
	err := s.authorizer.Authorize(ctx, actor, workspaceID, access.ActionView)
	if err != nil {
		return ListedWorkspace{}, err
	}

	return s.getForActor(ctx, actor, workspaceID)
}

func (s *Service) UpdateWorkspace(
	ctx context.Context,
	actor users.Actor,
	workspaceID uuid.UUID,
	name string,
) (ListedWorkspace, error) {
	err := validateName(name)
	if err != nil {
		return ListedWorkspace{}, err
	}

	err = s.authorizer.Authorize(ctx, actor, workspaceID, access.ActionUpdateWorkspace)
	if err != nil {
		return ListedWorkspace{}, err
	}

	var updatedWorkspace ListedWorkspace

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		existingWorkspace, err := s.getForActor(ctx, actor, workspaceID)
		if err != nil {
			return err
		}

		err = s.repository.updateName(ctx, workspaceID, name)
		if err != nil {
			return err
		}

		message := "Workspace updated: " + name
		if existingWorkspace.Name != name {
			message = fmt.Sprintf("Workspace updated and renamed from '%s' to '%s'", existingWorkspace.Name, name)
		}

		updatedWorkspace = existingWorkspace
		updatedWorkspace.Name = name

		return s.recordAuditEvent(ctx, actor, workspaceID, message)
	})
	if err != nil {
		return ListedWorkspace{}, err
	}

	return updatedWorkspace, nil
}

// DeleteWorkspace runs every subscriber of Deleting and deletes the workspace in one transaction,
// so a failure anywhere leaves the workspace as it was. A row of another module that still
// references the workspace, such as a database added after its subscriber looked, refuses the
// deletion with ErrWorkspaceNotEmpty.
func (s *Service) DeleteWorkspace(ctx context.Context, actor users.Actor, workspaceID uuid.UUID) error {
	err := s.authorizer.Authorize(ctx, actor, workspaceID, access.ActionDeleteWorkspace)
	if err != nil {
		return err
	}

	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		deletedWorkspace, isFound, err := s.repository.lockForDeletion(ctx, workspaceID)
		if err != nil {
			return err
		}

		if !isFound {
			return ErrWorkspaceNotFound
		}

		for _, handler := range s.deletingHandlers {
			err := handler(ctx, Deleting{WorkspaceID: workspaceID})
			if err != nil {
				return err
			}
		}

		err = s.repository.delete(ctx, workspaceID)
		if database.IsForeignKeyViolation(err) {
			return ErrWorkspaceNotEmpty
		}

		if err != nil {
			return err
		}

		return s.recordAuditEvent(ctx, actor, workspaceID, "Workspace deleted: "+deletedWorkspace.Name)
	})
}

func (s *Service) getForActor(ctx context.Context, actor users.Actor, workspaceID uuid.UUID) (ListedWorkspace, error) {
	listedWorkspace, isFound, err := s.repository.getForUser(ctx, workspaceID, actor.UserID)
	if err != nil {
		return ListedWorkspace{}, err
	}

	if !isFound {
		return ListedWorkspace{}, ErrWorkspaceNotFound
	}

	return listedWorkspace, nil
}

func (s *Service) recordAuditEvent(
	ctx context.Context,
	actor users.Actor,
	workspaceID uuid.UUID,
	message string,
) error {
	author, isFound, err := s.users.GetUser(ctx, actor.UserID)
	if err != nil {
		return err
	}

	if !isFound {
		return ErrActorNotFound
	}

	return s.auditLog.Record(ctx, audit.Event{
		Author:      author.GetAuditAuthor(),
		WorkspaceID: &workspaceID,
		Message:     message,
	})
}

func validateName(name string) error {
	nameLength := utf8.RuneCountInString(name)
	if nameLength < 1 || nameLength > maxWorkspaceNameLength {
		return ErrInvalidName
	}

	return nil
}
