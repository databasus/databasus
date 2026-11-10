package members

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces/internal/access"
	"databasus-backend/internal/modules/workspaces/internal/workspace"
)

// memberUniqueConstraintName allows one member per user and workspace.
const memberUniqueConstraintName = "uk_workspaces_members_user_workspace"

type Deps struct {
	Repositories Repositories
	Transactor   *database.Transactor
	Clock        clock.Source
	AuditLog     *audit.Log
	Users        users.Module
	Authorizer   *access.Authorizer
	Workspaces   *workspace.Service
	Mailer       mailer.Sender
	Logger       *slog.Logger
	// PublicURL is the address the invitation email links to, empty when the operator set none.
	PublicURL string
}

type Service struct {
	repository *memberRepository
	transactor *database.Transactor
	clock      clock.Source
	auditLog   *audit.Log
	users      users.Module
	authorizer *access.Authorizer
	workspaces *workspace.Service
	mailer     mailer.Sender
	logger     *slog.Logger
	publicURL  string
}

func NewService(deps Deps) *Service {
	return &Service{
		repository: deps.Repositories.members,
		transactor: deps.Transactor,
		clock:      deps.Clock,
		auditLog:   deps.AuditLog,
		users:      deps.Users,
		authorizer: deps.Authorizer,
		workspaces: deps.Workspaces,
		mailer:     deps.Mailer,
		logger:     deps.Logger,
		publicURL:  deps.PublicURL,
	}
}

// ListMembers pages through the members of a workspace in the order they joined.
func (s *Service) ListMembers(ctx context.Context, actor users.Actor, query MemberListQuery) (MemberPage, error) {
	limit := defaultMemberPageLimit
	if query.Limit != nil {
		limit = *query.Limit
	}

	if limit < 1 || limit > maxMemberPageLimit {
		return MemberPage{}, apierror.NewInvalidFieldsProblem([]apierror.FieldError{{
			Field:  "limit",
			Detail: fmt.Sprintf("must be between 1 and %d", maxMemberPageLimit),
		}})
	}

	var after *memberCursor

	if query.After != nil {
		cursor, err := decodeMemberCursor(*query.After)
		if err != nil {
			return MemberPage{}, err
		}

		after = &cursor
	}

	err := s.authorizeExistingWorkspace(ctx, actor, query.WorkspaceID, access.ActionView)
	if err != nil {
		return MemberPage{}, err
	}

	pageMembers, err := s.repository.list(ctx, memberListFilter{
		workspaceID: query.WorkspaceID,
		limit:       limit + 1,
		after:       after,
	})
	if err != nil {
		return MemberPage{}, err
	}

	page := MemberPage{Members: nil, NextCursor: nil}

	if len(pageMembers) > limit {
		lastMember := pageMembers[limit-1]
		nextCursor := memberCursor{createdAt: lastMember.CreatedAt, memberID: lastMember.ID}.encode()
		page.NextCursor = &nextCursor
		pageMembers = pageMembers[:limit]
	}

	page.Members, err = s.toListedMembers(ctx, pageMembers)
	if err != nil {
		return MemberPage{}, err
	}

	return page, nil
}

// AddMember adds the user with the address, inviting one when the address has none. The invited
// user and the member are recorded together, and the invitation email goes out only after both.
func (s *Service) AddMember(ctx context.Context, actor users.Actor, addition Addition) (AddedMember, error) {
	if !addition.Role.IsAssignable() {
		return AddedMember{}, ErrRoleNotAssignable
	}

	err := s.authorizeExistingWorkspace(ctx, actor, addition.WorkspaceID, getActionToManage(addition.Role))
	if err != nil {
		return AddedMember{}, err
	}

	var addedMember AddedMember

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		memberUser, isFound, err := s.users.GetUserByEmail(ctx, addition.Email)
		if err != nil {
			return err
		}

		addedMember.Outcome = AdditionOutcomeAdded

		if !isFound {
			memberUser, err = s.users.InviteUser(ctx, actor, users.Invitation{
				Email:       addition.Email,
				WorkspaceID: addition.WorkspaceID,
			})
			if err != nil {
				return err
			}

			addedMember.Outcome = AdditionOutcomeInvited
		}

		addedMember.ListedMember = ListedMember{
			ID:          uuid.New(),
			WorkspaceID: addition.WorkspaceID,
			UserID:      memberUser.ID,
			Role:        addition.Role,
			CreatedAt:   s.clock.Now(),
			Email:       memberUser.Email,
			Name:        memberUser.Name,
			IsInvited:   memberUser.Status == users.StatusInvited,
		}

		err = s.repository.insert(ctx, addedMember.Member)
		if isMemberDuplicate(err) {
			return ErrAlreadyMember
		}

		if err != nil {
			return err
		}

		message := fmt.Sprintf("User added to workspace: %s as %s", memberUser.Email, addition.Role)
		if addedMember.Outcome == AdditionOutcomeInvited {
			message = fmt.Sprintf("User invited to workspace: %s and added as %s", memberUser.Email, addition.Role)
		}

		return s.recordAuditEvent(ctx, actor, addition.WorkspaceID, message)
	})
	if err != nil {
		return AddedMember{}, err
	}

	if addedMember.Outcome == AdditionOutcomeInvited {
		s.sendInvitationEmail(ctx, actor, addedMember.ListedMember)
	}

	return addedMember, nil
}

// ChangeMemberRole checks in a fixed order: the member exists, the actor may manage the member,
// the member is not the actor's own, and the member is not the owner. Permission comes first, so an
// actor without it never learns more from a 409. The checks run on the member's row locked in the
// transaction, so a concurrent ownership transfer cannot make the member the owner in between.
func (s *Service) ChangeMemberRole(
	ctx context.Context,
	actor users.Actor,
	memberID uuid.UUID,
	newRole access.Role,
) (ListedMember, error) {
	if !newRole.IsAssignable() {
		return ListedMember{}, ErrRoleNotAssignable
	}

	var changedMember ListedMember

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		member, err := s.lockMember(ctx, memberID)
		if err != nil {
			return err
		}

		action := getActionToManage(newRole)
		if member.Role == access.RoleAdmin {
			action = access.ActionManageAdmins
		}

		err = s.authorizer.Authorize(ctx, actor, member.WorkspaceID, action)
		if err != nil {
			return err
		}

		if member.UserID == actor.UserID {
			return ErrCannotChangeOwnRole
		}

		if member.Role == access.RoleOwner {
			return ErrOwnerRequiresTransfer
		}

		changedMember, err = s.toListedMember(ctx, member)
		if err != nil {
			return err
		}

		_, err = s.repository.updateRole(ctx, member.ID, newRole)
		if err != nil {
			return err
		}

		changedMember.Role = newRole

		return s.recordAuditEvent(ctx, actor, member.WorkspaceID,
			fmt.Sprintf("Member role changed: %s from %s to %s", changedMember.Email, member.Role, newRole))
	})
	if err != nil {
		return ListedMember{}, err
	}

	return changedMember, nil
}

// RemoveMember ends the user's access to the workspace and keeps their account. It checks the
// member's row locked in the transaction, as ChangeMemberRole does.
func (s *Service) RemoveMember(ctx context.Context, actor users.Actor, memberID uuid.UUID) error {
	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		member, err := s.lockMember(ctx, memberID)
		if err != nil {
			return err
		}

		err = s.authorizer.Authorize(ctx, actor, member.WorkspaceID, getActionToManage(member.Role))
		if err != nil {
			return err
		}

		if member.Role == access.RoleOwner {
			return ErrOwnerRequiresTransfer
		}

		removedMember, err := s.toListedMember(ctx, member)
		if err != nil {
			return err
		}

		err = s.repository.delete(ctx, member.ID)
		if err != nil {
			return err
		}

		return s.recordAuditEvent(ctx, actor, member.WorkspaceID,
			"Member removed from workspace: "+removedMember.Email)
	})
}

// TransferOwnership demotes the current owner to admin and then promotes the new owner in one
// transaction, so the workspace never has two owners. Both rows are locked before either changes,
// so a concurrent removal or role change of the new owner cannot leave the workspace without one.
// Naming the current owner changes nothing.
func (s *Service) TransferOwnership(
	ctx context.Context,
	actor users.Actor,
	workspaceID uuid.UUID,
	newOwnerMemberID uuid.UUID,
) error {
	err := s.authorizeExistingWorkspace(ctx, actor, workspaceID, access.ActionTransferOwnership)
	if err != nil {
		return err
	}

	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		currentOwner, hasOwner, err := s.repository.lockOwner(ctx, workspaceID)
		if err != nil {
			return err
		}

		newOwner, err := s.lockMember(ctx, newOwnerMemberID)
		if err != nil {
			return err
		}

		if newOwner.WorkspaceID != workspaceID {
			return ErrMemberNotFound
		}

		if newOwner.Role == access.RoleOwner {
			return nil
		}

		listedNewOwner, err := s.toListedMember(ctx, newOwner)
		if err != nil {
			return err
		}

		if hasOwner {
			_, err = s.repository.updateRole(ctx, currentOwner.ID, access.RoleAdmin)
			if err != nil {
				return err
			}
		}

		isPromoted, err := s.repository.updateRole(ctx, newOwner.ID, access.RoleOwner)
		if err != nil {
			return err
		}

		if !isPromoted {
			return ErrMemberNotFound
		}

		return s.recordAuditEvent(ctx, actor, workspaceID,
			"Workspace ownership transferred to: "+listedNewOwner.Email)
	})
}

// authorizeExistingWorkspace also answers not found to an instance admin, whom Authorize lets
// through without looking at the workspace.
func (s *Service) authorizeExistingWorkspace(
	ctx context.Context,
	actor users.Actor,
	workspaceID uuid.UUID,
	action access.Action,
) error {
	err := s.authorizer.Authorize(ctx, actor, workspaceID, action)
	if err != nil {
		return err
	}

	foundWorkspaces, err := s.workspaces.GetWorkspaces(ctx, []uuid.UUID{workspaceID})
	if err != nil {
		return err
	}

	if len(foundWorkspaces) == 0 {
		return ErrWorkspaceNotFound
	}

	return nil
}

func (s *Service) lockMember(ctx context.Context, memberID uuid.UUID) (Member, error) {
	member, isFound, err := s.repository.lockByID(ctx, memberID)
	if err != nil {
		return Member{}, err
	}

	if !isFound {
		return Member{}, ErrMemberNotFound
	}

	return member, nil
}

func (s *Service) toListedMember(ctx context.Context, member Member) (ListedMember, error) {
	listedMembers, err := s.toListedMembers(ctx, []Member{member})
	if err != nil {
		return ListedMember{}, err
	}

	return listedMembers[0], nil
}

// toListedMembers reads the users of the members in one call and joins them in Go.
func (s *Service) toListedMembers(ctx context.Context, pageMembers []Member) ([]ListedMember, error) {
	userIDs := make([]uuid.UUID, 0, len(pageMembers))
	for _, member := range pageMembers {
		userIDs = append(userIDs, member.UserID)
	}

	memberUsers, err := s.users.GetUsers(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	usersByID := make(map[uuid.UUID]users.User, len(memberUsers))
	for _, memberUser := range memberUsers {
		usersByID[memberUser.ID] = memberUser
	}

	listedMembers := make([]ListedMember, 0, len(pageMembers))

	for _, member := range pageMembers {
		memberUser, isFound := usersByID[member.UserID]
		if !isFound {
			return nil, fmt.Errorf("%w: member %s", errMemberUserMissing, member.ID)
		}

		listedMembers = append(listedMembers, ListedMember{
			Member:    member,
			Email:     memberUser.Email,
			Name:      memberUser.Name,
			IsInvited: memberUser.Status == users.StatusInvited,
		})
	}

	return listedMembers, nil
}

// sendInvitationEmail runs after the commit. A refused send keeps the invited member: the
// instance admin can share the address of the instance instead.
func (s *Service) sendInvitationEmail(ctx context.Context, actor users.Actor, invitedMember ListedMember) {
	inviter, isFound, err := s.users.GetUser(ctx, actor.UserID)
	if err != nil || !isFound {
		s.logger.ErrorContext(ctx, "failed to read the inviter for the invitation email",
			slog.String("user_id", actor.UserID.String()), slog.Any("error", err))

		return
	}

	foundWorkspaces, err := s.workspaces.GetWorkspaces(ctx, []uuid.UUID{invitedMember.WorkspaceID})
	if err != nil || len(foundWorkspaces) == 0 {
		s.logger.ErrorContext(ctx, "failed to read the workspace for the invitation email",
			slog.String("workspace_id", invitedMember.WorkspaceID.String()), slog.Any("error", err))

		return
	}

	email := newInvitationEmail(foundWorkspaces[0].Name, inviter.Name, string(invitedMember.Role), s.publicURL)

	err = s.mailer.SendEmail(ctx, invitedMember.Email, email.subject, email.body)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to send the invitation email",
			slog.String("email", invitedMember.Email), slog.Any("error", err))
	}
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

// getActionToManage is the permission to add, change or remove a member with the role.
func getActionToManage(role access.Role) access.Action {
	if role == access.RoleAdmin {
		return access.ActionManageAdmins
	}

	return access.ActionManageMembers
}

func isMemberDuplicate(err error) bool {
	var pgError *pgconn.PgError

	return errors.As(err, &pgError) && pgError.ConstraintName == memberUniqueConstraintName
}
