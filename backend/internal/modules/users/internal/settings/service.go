package settings

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/lib/ratelimit"
	"databasus-backend/internal/modules/users/internal/identity"
)

// UserReader reads the users the settings rules name. The accounts package implements it; this
// package cannot import accounts, because accounts asks the settings who may invite.
type UserReader interface {
	GetUser(ctx context.Context, userID uuid.UUID) (identity.User, bool, error)
	ListAdmins(ctx context.Context) ([]identity.User, error)
}

type Deps struct {
	Repositories Repositories
	Transactor   *database.Transactor
	Users        UserReader
	AuditLog     *audit.Log
	Mailer       mailer.Sender
	RateLimiter  ratelimit.Counter
	Logger       *slog.Logger
}

type Service struct {
	repository  *settingsRepository
	transactor  *database.Transactor
	users       UserReader
	auditLog    *audit.Log
	mailer      mailer.Sender
	rateLimiter ratelimit.Counter
	logger      *slog.Logger
}

func NewService(deps Deps) *Service {
	return &Service{
		repository:  deps.Repositories.settings,
		transactor:  deps.Transactor,
		users:       deps.Users,
		auditLog:    deps.AuditLog,
		mailer:      deps.Mailer,
		rateLimiter: deps.RateLimiter,
		logger:      deps.Logger,
	}
}

func (s *Service) GetSettings(ctx context.Context) (Settings, error) {
	return s.repository.getOrCreate(ctx)
}

func (s *Service) GetInstanceSettings(ctx context.Context) (InstanceSettings, error) {
	settings, err := s.repository.getOrCreate(ctx)
	if err != nil {
		return InstanceSettings{}, err
	}

	return s.toInstanceSettings(settings), nil
}

func (s *Service) GetTwoFactorRequirement(ctx context.Context) (TwoFactorRequirement, error) {
	settings, err := s.repository.getOrCreate(ctx)
	if err != nil {
		return "", err
	}

	return settings.TwoFactorRequirement, nil
}

func (s *Service) CanInviteUsers(ctx context.Context, actor identity.Actor) (bool, error) {
	if actor.IsAdmin() {
		return true, nil
	}

	settings, err := s.repository.getOrCreate(ctx)
	if err != nil {
		return false, err
	}

	return settings.IsAllowMemberInvitations, nil
}

func (s *Service) CanCreateWorkspaces(ctx context.Context, actor identity.Actor) (bool, error) {
	if actor.IsAdmin() {
		return true, nil
	}

	settings, err := s.repository.getOrCreate(ctx)
	if err != nil {
		return false, err
	}

	return settings.IsMemberAllowedToCreateWorkspaces, nil
}

// UpdateInstanceSettings refuses to require the emailed code unless every active administrator can
// receive one, because that requirement is what locks an instance out. Turning it off or moving to
// the authenticator app is never refused for mail, so an administrator can still leave an instance
// whose mail server is gone.
func (s *Service) UpdateInstanceSettings(
	ctx context.Context,
	actor identity.Actor,
	switches Switches,
) (InstanceSettings, error) {
	if !actor.IsAdmin() {
		return InstanceSettings{}, ErrNotAdmin
	}

	if !switches.TwoFactorRequirement.IsValid() {
		return InstanceSettings{}, ErrTwoFactorRequirementInvalid
	}

	author, err := s.getActorUser(ctx, actor)
	if err != nil {
		return InstanceSettings{}, err
	}

	var updatedSettings Settings

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		currentSettings, err := s.repository.getOrCreate(ctx)
		if err != nil {
			return err
		}

		isBecomingEmail := switches.TwoFactorRequirement == TwoFactorRequirementEmail &&
			currentSettings.TwoFactorRequirement != TwoFactorRequirementEmail
		if isBecomingEmail {
			err = s.checkCodesCanReachEveryAdmin(ctx)
			if err != nil {
				return err
			}
		}

		err = s.repository.updateSwitches(ctx, currentSettings.ID, switches)
		if err != nil {
			return err
		}

		for _, message := range describeChangedSwitches(currentSettings, switches) {
			err = s.auditLog.Record(
				ctx,
				audit.Event{Author: author.GetAuditAuthor(), WorkspaceID: nil, Message: message},
			)
			if err != nil {
				return err
			}
		}

		updatedSettings = applySwitches(currentSettings, switches)

		return nil
	})
	if err != nil {
		return InstanceSettings{}, err
	}

	return s.toInstanceSettings(updatedSettings), nil
}

// DisableTwoFactorAuthFromConsole is the way back into an instance whose mail server has died, so
// it takes no actor and asks no permission: whoever runs it already runs commands inside the
// instance. The audit event keeps the host from being a way to move a security switch unobserved.
// Connected authenticator apps stay: they belong to the accounts, not to the requirement.
func (s *Service) DisableTwoFactorAuthFromConsole(ctx context.Context) (bool, error) {
	isChanged := false

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		currentSettings, err := s.repository.getOrCreate(ctx)
		if err != nil {
			return err
		}

		if !currentSettings.TwoFactorRequirement.IsRequired() {
			return nil
		}

		err = s.repository.updateTwoFactorRequirement(ctx, currentSettings.ID, TwoFactorRequirementOff)
		if err != nil {
			return err
		}

		isChanged = true

		return s.auditLog.Record(ctx, audit.Event{
			Author:      nil,
			WorkspaceID: nil,
			Message: fmt.Sprintf("twoFactorRequirement: %s -> %s (switched off from the host console)",
				currentSettings.TwoFactorRequirement, TwoFactorRequirementOff),
		})
	})

	return isChanged, err
}

func (s *Service) checkCodesCanReachEveryAdmin(ctx context.Context) error {
	if !s.mailer.IsConfigured() {
		return ErrTwoFactorWithoutMailServer
	}

	admins, err := s.users.ListAdmins(ctx)
	if err != nil {
		return err
	}

	for _, admin := range admins {
		if admin.IsActive() && !identity.IsEmailAddress(admin.Email) {
			return newTwoFactorAdminEmailMissingProblem(admin.Email)
		}
	}

	return nil
}

func (s *Service) getActorUser(ctx context.Context, actor identity.Actor) (identity.User, error) {
	user, isFound, err := s.users.GetUser(ctx, actor.UserID)
	if err != nil {
		return identity.User{}, err
	}

	if !isFound {
		return identity.User{}, fmt.Errorf("the signed-in user %s does not exist", actor.UserID)
	}

	return user, nil
}

func (s *Service) toInstanceSettings(settings Settings) InstanceSettings {
	return InstanceSettings{
		IsAllowExternalRegistrations:      settings.IsAllowExternalRegistrations,
		IsAllowMemberInvitations:          settings.IsAllowMemberInvitations,
		IsMemberAllowedToCreateWorkspaces: settings.IsMemberAllowedToCreateWorkspaces,
		TwoFactorRequirement:              settings.TwoFactorRequirement,
		IsEmailConfigured:                 s.mailer.IsConfigured(),
	}
}

func applySwitches(settings Settings, switches Switches) Settings {
	settings.IsAllowExternalRegistrations = switches.IsAllowExternalRegistrations
	settings.IsAllowMemberInvitations = switches.IsAllowMemberInvitations
	settings.IsMemberAllowedToCreateWorkspaces = switches.IsMemberAllowedToCreateWorkspaces
	settings.TwoFactorRequirement = switches.TwoFactorRequirement

	return settings
}

// describeChangedSwitches names each switch the update changes, one audit message per switch.
func describeChangedSwitches(current Settings, proposed Switches) []string {
	changes := []struct {
		name       string
		isCurrent  bool
		isProposed bool
	}{
		{"isAllowExternalRegistrations", current.IsAllowExternalRegistrations, proposed.IsAllowExternalRegistrations},
		{"isAllowMemberInvitations", current.IsAllowMemberInvitations, proposed.IsAllowMemberInvitations},
		{
			"isMemberAllowedToCreateWorkspaces",
			current.IsMemberAllowedToCreateWorkspaces,
			proposed.IsMemberAllowedToCreateWorkspaces,
		},
	}

	messages := make([]string, 0, len(changes)+1)

	for _, change := range changes {
		if change.isCurrent != change.isProposed {
			messages = append(messages, fmt.Sprintf("%s: %t -> %t", change.name, change.isCurrent, change.isProposed))
		}
	}

	if current.TwoFactorRequirement != proposed.TwoFactorRequirement {
		messages = append(messages, fmt.Sprintf("twoFactorRequirement: %s -> %s",
			current.TwoFactorRequirement, proposed.TwoFactorRequirement))
	}

	return messages
}
