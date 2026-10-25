package accounts

import (
	"context"
	"fmt"
	"strings"

	"databasus-backend/internal/modules/users/internal/identity"
)

// noAdminsMessage answers an instance nobody has signed up on. It is a sentence rather than an
// empty listing, because the owner who runs the command is finding out why they cannot sign in.
const noAdminsMessage = "This instance has no administrator: no account has been created on it yet."

type ConsoleUserRequest struct {
	Email    string
	Password string
	// Role is admin or member in any case.
	Role string
	// Name is empty for the local part of the address.
	Name string
}

type ConsoleUserOutcome struct {
	User identity.User
	// IsCreated is false when the same user already existed and nothing changed.
	IsCreated        bool
	IsBootstrapAdmin bool
}

// ConsolePasswordChange names the address and the new password, so the two strings cannot be
// swapped by position.
type ConsolePasswordChange struct {
	Email    string
	Password string
}

// CreateUserFromConsole creates a user from inside the running instance, so it skips the
// registration setting and may create an instance admin without the bootstrap administrator:
// whoever runs it can already reset that user's password. It never changes an existing user, and
// an exact match is reported rather than refused, so a deployment can run it on every rollout.
//
// Nothing here logs. The outcome and the errors are all the command prints, which keeps the
// password out of its output.
func (s *Service) CreateUserFromConsole(ctx context.Context, request ConsoleUserRequest) (ConsoleUserOutcome, error) {
	role, err := validateConsoleUserRequest(request)
	if err != nil {
		return ConsoleUserOutcome{}, err
	}

	hashedPassword, err := identity.HashPassword(request.Password)
	if err != nil {
		return ConsoleUserOutcome{}, err
	}

	var outcome ConsoleUserOutcome

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		existingUser, isFound, err := s.repository.getByEmail(ctx, request.Email)
		if err != nil {
			return err
		}

		if isFound {
			if !isSameConsoleUser(existingUser, role) {
				return ErrConsoleUserExistsInAnotherState
			}

			outcome = ConsoleUserOutcome{
				User:             existingUser.ToUser(),
				IsCreated:        false,
				IsBootstrapAdmin: existingUser.IsBootstrapAdmin,
			}

			return nil
		}

		// An instance never becomes empty again once it has a user, so a sign-up landing after this
		// check can only make the refusal stricter than needed, never let a member claim the instance.
		userCount, err := s.repository.countAll(ctx)
		if err != nil {
			return err
		}

		if userCount == 0 && role != identity.RoleAdmin {
			return ErrConsoleFirstUserMustBeAdmin
		}

		createdUser, err := s.CreateUserClaimingEmptyInstance(ctx, NewUser{
			Email:                    request.Email,
			Name:                     getConsoleUserName(request),
			HashedPassword:           &hashedPassword,
			GitHubOAuthID:            nil,
			GoogleOAuthID:            nil,
			OrdinaryRole:             role,
			CheckOrdinaryUserAllowed: allowConsoleUser,
		})
		if err != nil {
			return err
		}

		err = s.recordAuditEvent(ctx, createdUser,
			fmt.Sprintf("Account created from the host console: %s (role %s)", createdUser.Email, createdUser.Role))
		if err != nil {
			return err
		}

		outcome = ConsoleUserOutcome{
			User:             createdUser.ToUser(),
			IsCreated:        true,
			IsBootstrapAdmin: createdUser.IsBootstrapAdmin,
		}

		return nil
	})

	return outcome, err
}

// ChangePasswordFromConsole accepts any stored address, including the placeholder address of an
// upgraded instance, because the console is how an owner who cannot sign in gets back.
func (s *Service) ChangePasswordFromConsole(ctx context.Context, change ConsolePasswordChange) error {
	if !identity.IsPasswordLongEnough(change.Password) {
		return ErrConsolePasswordTooShort
	}

	hashedPassword, err := identity.HashPassword(change.Password)
	if err != nil {
		return err
	}

	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		user, isFound, err := s.repository.getByEmail(ctx, change.Email)
		if err != nil {
			return err
		}

		if !isFound {
			return ErrConsoleUserNotFound
		}

		return s.ReplacePassword(ctx, user, hashedPassword, "Password changed")
	})
}

// RenderAdminListing prints one line per instance admin. The active state is part of it, because a
// deactivated administrator otherwise reads as an address that should work. Nothing else is
// printed: no password, no hash, no token. The listing writes no audit event.
func (s *Service) RenderAdminListing(ctx context.Context) (string, error) {
	admins, err := s.repository.listAdmins(ctx)
	if err != nil {
		return "", err
	}

	if len(admins) == 0 {
		return noAdminsMessage, nil
	}

	lines := make([]string, 0, len(admins))
	for _, admin := range admins {
		lines = append(lines, renderAdminLine(admin))
	}

	return strings.Join(lines, "\n"), nil
}

func validateConsoleUserRequest(request ConsoleUserRequest) (identity.Role, error) {
	if !identity.IsEmailAddress(request.Email) {
		return "", ErrConsoleEmailInvalid
	}

	if !identity.IsPasswordLongEnough(request.Password) {
		return "", ErrConsolePasswordTooShort
	}

	role := identity.Role(strings.ToUpper(request.Role))
	if !role.IsValid() {
		return "", ErrConsoleRoleInvalid
	}

	return role, nil
}

// isSameConsoleUser counts a user without a password as another state: reporting it as ready would
// tell a deployment the address signs in with a password it does not hold.
func isSameConsoleUser(user UserRecord, role identity.Role) bool {
	return user.IsActive() && user.Role == role && user.HasPassword()
}

func getConsoleUserName(request ConsoleUserRequest) string {
	if request.Name != "" {
		return request.Name
	}

	localPart, _, _ := strings.Cut(request.Email, "@")

	return localPart
}

func allowConsoleUser(context.Context) error {
	return nil
}

func renderAdminLine(admin UserRecord) string {
	activeState := "inactive"
	if admin.IsActive() {
		activeState = "active"
	}

	line := fmt.Sprintf("%s | %s | created %s | %s",
		admin.Email, admin.Name, admin.CreatedAt.UTC().Format("2006-01-02"), activeState)

	if admin.IsBootstrapAdmin {
		line += " | bootstrap administrator"
	}

	return line
}
