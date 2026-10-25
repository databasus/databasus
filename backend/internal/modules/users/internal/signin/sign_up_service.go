package signin

import (
	"context"
	"fmt"
	"strings"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/modules/users/internal/accounts"
	"databasus-backend/internal/modules/users/internal/identity"
)

type SignUpRequest struct {
	Email           string
	Password        string
	Name            string
	TurnstileAnswer *string
}

// SignUp creates the user and signs them in. An invited user finishes sign-up here even when
// registration is closed; anyone else needs open registration, except the first user, who claims
// the instance.
func (s *Service) SignUp(ctx context.Context, request SignUpRequest) (Session, error) {
	err := validateSignUp(request)
	if err != nil {
		return Session{}, err
	}

	err = s.turnstile.Verify(ctx, request.TurnstileAnswer)
	if err != nil {
		return Session{}, err
	}

	hashedPassword, err := identity.HashPassword(request.Password)
	if err != nil {
		return Session{}, err
	}

	var signedUpUser accounts.UserRecord

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		existingUser, isFound, err := s.accounts.GetRecordByEmail(ctx, request.Email)
		if err != nil {
			return err
		}

		if isFound && existingUser.Status != identity.StatusInvited {
			return accounts.ErrEmailTaken
		}

		if isFound {
			signedUpUser, err = s.accounts.ActivateInvitedUser(ctx, existingUser, request.Name, &hashedPassword)
			if err != nil {
				return err
			}

			return s.recordAuditEvent(ctx, signedUpUser, "Invited user completed registration: "+signedUpUser.Email)
		}

		signedUpUser, err = s.accounts.CreateUserClaimingEmptyInstance(ctx, accounts.NewUser{
			Email:                    request.Email,
			Name:                     request.Name,
			HashedPassword:           &hashedPassword,
			GitHubOAuthID:            nil,
			GoogleOAuthID:            nil,
			OrdinaryRole:             identity.RoleMember,
			CheckOrdinaryUserAllowed: s.checkRegistrationOpen,
		})
		if err != nil {
			return err
		}

		return s.recordAuditEvent(ctx, signedUpUser, "User registered with email: "+signedUpUser.Email)
	})
	if err != nil {
		return Session{}, err
	}

	return s.issueSession(signedUpUser)
}

func validateSignUp(request SignUpRequest) error {
	var fieldErrors []apierror.FieldError

	if !identity.IsEmailAddress(request.Email) {
		fieldErrors = append(fieldErrors, apierror.FieldError{Field: "email", Detail: "is not an email address"})
	}

	if !identity.IsPasswordLongEnough(request.Password) {
		fieldErrors = append(fieldErrors, apierror.FieldError{
			Field:  "password",
			Detail: fmt.Sprintf("must be at least %d characters long", identity.MinPasswordLength),
		})
	}

	if strings.TrimSpace(request.Name) == "" {
		fieldErrors = append(fieldErrors, apierror.FieldError{Field: "name", Detail: "is required"})
	}

	if len(fieldErrors) > 0 {
		return apierror.NewInvalidFieldsProblem(fieldErrors)
	}

	return nil
}
