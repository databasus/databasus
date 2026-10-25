package signin

import (
	"context"
	"log/slog"

	"databasus-backend/internal/modules/users/internal/accounts"
	"databasus-backend/internal/modules/users/internal/identity"
	"databasus-backend/internal/modules/users/internal/oauth"
)

type OAuthSignInRequest struct {
	Code        string
	RedirectURI string
}

type identityProvider interface {
	GetProvider() oauth.Provider
	IsConfigured() bool
	FetchAccount(ctx context.Context, authorization oauth.Authorization) (oauth.Account, error)
}

func (s *Service) SignInWithGitHub(ctx context.Context, request OAuthSignInRequest) (Session, error) {
	return s.signInWithProvider(ctx, s.gitHub, request)
}

func (s *Service) SignInWithGoogle(ctx context.Context, request OAuthSignInRequest) (Session, error) {
	return s.signInWithProvider(ctx, s.google, request)
}

// signInWithProvider issues the session directly, even when the instance requires the second
// factor: the provider has verified the user, and the code would go to the address it vouched for.
func (s *Service) signInWithProvider(
	ctx context.Context,
	provider identityProvider,
	request OAuthSignInRequest,
) (Session, error) {
	if !provider.IsConfigured() {
		return Session{}, ErrOAuthNotConfigured
	}

	account, err := provider.FetchAccount(ctx, oauth.Authorization{
		Code:        request.Code,
		RedirectURI: request.RedirectURI,
	})
	if err != nil {
		s.logger.WarnContext(ctx, "oauth sign-in refused by the provider",
			slog.String("provider", string(provider.GetProvider())), slog.Any("error", err))

		return Session{}, ErrOAuthFailed
	}

	var signedInUser accounts.UserRecord

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		signedInUser, err = s.findOrCreateProviderUser(ctx, provider.GetProvider(), account)

		return err
	})
	if err != nil {
		return Session{}, err
	}

	s.logger.InfoContext(ctx, "user signed in",
		slog.String("user_id", signedInUser.ID.String()), slog.String("provider", string(provider.GetProvider())))

	return s.issueSession(signedInUser)
}

// findOrCreateProviderUser signs in the user linked to the provider's account. Otherwise the user
// with the same address gets the account linked, and an invited one finishes sign-up with it.
// Otherwise a new user is created under the rules of sign-up.
func (s *Service) findOrCreateProviderUser(
	ctx context.Context,
	provider oauth.Provider,
	account oauth.Account,
) (accounts.UserRecord, error) {
	linkedUser, isFound, err := s.accounts.GetRecordByOAuthID(ctx, provider, account.ID)
	if err != nil {
		return accounts.UserRecord{}, err
	}

	if isFound {
		return s.signInLinkedUser(ctx, provider, linkedUser)
	}

	userWithEmail, isFound, err := s.accounts.GetRecordByEmail(ctx, account.Email)
	if err != nil {
		return accounts.UserRecord{}, err
	}

	if isFound {
		return s.linkProviderAccount(ctx, provider, account, userWithEmail)
	}

	return s.createProviderUser(ctx, provider, account)
}

func (s *Service) signInLinkedUser(
	ctx context.Context,
	provider oauth.Provider,
	linkedUser accounts.UserRecord,
) (accounts.UserRecord, error) {
	if !linkedUser.IsActive() {
		return accounts.UserRecord{}, ErrUserDeactivated
	}

	err := s.recordAuditEvent(ctx, linkedUser, "User signed in with "+string(provider)+": "+linkedUser.Email)
	if err != nil {
		return accounts.UserRecord{}, err
	}

	return linkedUser, nil
}

func (s *Service) linkProviderAccount(
	ctx context.Context,
	provider oauth.Provider,
	account oauth.Account,
	user accounts.UserRecord,
) (accounts.UserRecord, error) {
	if user.Status == identity.StatusInactive {
		return accounts.UserRecord{}, ErrUserDeactivated
	}

	var err error

	if user.Status == identity.StatusInvited {
		user, err = s.accounts.ActivateInvitedUser(ctx, user, account.Name, nil)
		if err != nil {
			return accounts.UserRecord{}, err
		}

		err = s.recordAuditEvent(
			ctx,
			user,
			"Invited user completed registration with "+string(provider)+": "+user.Email,
		)
		if err != nil {
			return accounts.UserRecord{}, err
		}
	}

	err = s.accounts.LinkOAuthID(ctx, user.ID, provider, account.ID)
	if err != nil {
		return accounts.UserRecord{}, err
	}

	err = s.recordAuditEvent(ctx, user, string(provider)+" account linked to the user: "+user.Email)
	if err != nil {
		return accounts.UserRecord{}, err
	}

	return user, nil
}

func (s *Service) createProviderUser(
	ctx context.Context,
	provider oauth.Provider,
	account oauth.Account,
) (accounts.UserRecord, error) {
	newUser := accounts.NewUser{
		Email:                    account.Email,
		Name:                     account.Name,
		HashedPassword:           nil,
		GitHubOAuthID:            nil,
		GoogleOAuthID:            nil,
		OrdinaryRole:             identity.RoleMember,
		CheckOrdinaryUserAllowed: s.checkRegistrationOpen,
	}

	switch provider {
	case oauth.ProviderGitHub:
		newUser.GitHubOAuthID = &account.ID
	case oauth.ProviderGoogle:
		newUser.GoogleOAuthID = &account.ID
	}

	createdUser, err := s.accounts.CreateUserClaimingEmptyInstance(ctx, newUser)
	if err != nil {
		return accounts.UserRecord{}, err
	}

	err = s.recordAuditEvent(ctx, createdUser, "User registered with "+string(provider)+": "+createdUser.Email)
	if err != nil {
		return accounts.UserRecord{}, err
	}

	return createdUser, nil
}
