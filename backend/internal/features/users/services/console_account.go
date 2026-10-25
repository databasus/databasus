package users_services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	users_dto "databasus-backend/internal/features/users/dto"
	users_enums "databasus-backend/internal/features/users/enums"
	users_errors "databasus-backend/internal/features/users/errors"
	users_models "databasus-backend/internal/features/users/models"
	"databasus-backend/internal/modules/users"
)

// The library and the tag the registration binding evaluates, so the console
// accepts and refuses exactly the addresses sign-up does.
var consoleAccountValidator = validator.New()

type ConsoleAccountOutcome struct {
	User      *users_models.User
	IsCreated bool
}

// CreateUserFromConsole provisions an account from inside the running instance,
// so it skips the registration policy and may hand out the administrator role
// without the bootstrap administrator: whoever runs it can already reset that
// account's password. It never changes an existing account, and an exact match
// is reported rather than refused, so a deployment can run it on every rollout.
//
// Nothing here logs. The rendered outcome and the returned errors are all the
// command emits, which is what keeps the password out of its output.
func (s *UserService) CreateUserFromConsole(
	ctx context.Context,
	request *users_dto.CreateConsoleAccountRequest,
) (*ConsoleAccountOutcome, error) {
	role, err := validateConsoleAccountRequest(request)
	if err != nil {
		return nil, err
	}

	existingUser, err := s.userRepository.GetUserByEmail(ctx, request.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	if existingUser != nil {
		if !isSameConsoleAccount(existingUser, role) {
			return nil, users_errors.ErrConsoleAccountExistsInAnotherState
		}

		return &ConsoleAccountOutcome{User: existingUser, IsCreated: false}, nil
	}

	// An instance never becomes empty again once it holds an account, so a
	// registration landing after this check can only make the refusal stricter
	// than needed, never let a member take the instance.
	isInstanceUnclaimed, err := s.isInstanceUnclaimed()
	if err != nil {
		return nil, err
	}

	if isInstanceUnclaimed && role != users.RoleAdmin {
		return nil, users_errors.ErrConsoleAccountFirstMustBeAdmin
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	hashedPasswordStr := string(hashedPassword)
	name := getConsoleAccountName(request)

	user, err := s.createAccountClaimingEmptyInstance(
		ctx,
		allowConsoleAccount,
		func(isTakingTheInstance bool) *users_models.User {
			return buildConsoleUser(request.Email, name, hashedPasswordStr, role, isTakingTheInstance)
		},
	)
	if err != nil {
		return nil, err
	}

	s.writeAuditLog(ctx, audit_logs_models.AuditEntry{
		Message: fmt.Sprintf(
			"Account created from the host console: %s (role %s)",
			user.Email,
			user.Role,
		),
		UserID:      &user.ID,
		WorkspaceID: nil,
	})

	s.writeInstanceClaimedAuditLog(ctx, user)

	return &ConsoleAccountOutcome{User: user, IsCreated: true}, nil
}

// RenderConsoleAccountOutcome is the line the operator reads. A rerun says the
// password was not changed, because that is how a deployment learns a rerun did
// not rotate its secret.
func RenderConsoleAccountOutcome(outcome *ConsoleAccountOutcome) string {
	user := outcome.User

	if !outcome.IsCreated {
		return fmt.Sprintf(
			"Account %s already exists with the %s role. Nothing changed, and its password was not changed.",
			user.Email,
			user.Role,
		)
	}

	line := fmt.Sprintf("Created account %s with the %s role.", user.Email, user.Role)
	if user.IsRootAdmin {
		line += " It is the administrator of this instance."
	}

	return line
}

func validateConsoleAccountRequest(
	request *users_dto.CreateConsoleAccountRequest,
) (users.Role, error) {
	if err := consoleAccountValidator.Var(request.Email, "required,email"); err != nil {
		return "", users_errors.ErrConsoleAccountEmailInvalid
	}

	if !isPasswordLongEnough(request.Password) {
		return "", users_errors.ErrPasswordTooShort
	}

	role := users.Role(strings.ToUpper(request.Role))
	if !role.IsValid() {
		return "", users_errors.ErrConsoleAccountRoleInvalid
	}

	return role, nil
}

// A password-less account counts as another state: reporting it as ready would
// tell a deployment the address signs in with a password it does not hold.
func isSameConsoleAccount(user *users_models.User, role users.Role) bool {
	return user.IsActiveUser() && user.Role == role && user.HasPassword()
}

func getConsoleAccountName(request *users_dto.CreateConsoleAccountRequest) string {
	if request.Name != "" {
		return request.Name
	}

	localPart, _, _ := strings.Cut(request.Email, "@")

	return localPart
}

func allowConsoleAccount(context.Context) error {
	return nil
}

func buildConsoleUser(
	email, name, hashedPassword string,
	role users.Role,
	isTakingTheInstance bool,
) *users_models.User {
	return &users_models.User{
		ID:                   uuid.New(),
		Email:                email,
		Name:                 name,
		HashedPassword:       &hashedPassword,
		PasswordCreationTime: time.Now().UTC(),
		Role:                 role,
		Status:               users_enums.UserStatusActive,
		IsRootAdmin:          isTakingTheInstance,
		CreatedAt:            time.Now().UTC(),
	}
}
