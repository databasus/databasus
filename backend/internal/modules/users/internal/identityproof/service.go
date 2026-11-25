package identityproof

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/modules/users/internal/accounts"
	"databasus-backend/internal/modules/users/internal/authenticator"
	"databasus-backend/internal/modules/users/internal/identity"
	"databasus-backend/internal/modules/users/internal/settings"
)

const (
	codeRequestInterval = time.Minute

	// expiredCodeRetention keeps used and expired codes briefly, so the one-a-minute rule still sees
	// a code that expired a moment ago.
	expiredCodeRetention = time.Hour
)

type Deps struct {
	Repositories  Repositories
	Transactor    *database.Transactor
	Clock         clock.Source
	Accounts      *accounts.Service
	Directory     *accounts.Directory
	Authenticator *authenticator.Service
	Settings      *settings.Service
	Mailer        mailer.Sender
	AuditLog      *audit.Log
	Logger        *slog.Logger
}

type Service struct {
	codes         *identityCodeRepository
	transactor    *database.Transactor
	clock         clock.Source
	accounts      *accounts.Service
	directory     *accounts.Directory
	authenticator *authenticator.Service
	settings      *settings.Service
	mailer        mailer.Sender
	auditLog      *audit.Log
	logger        *slog.Logger
}

func NewService(deps Deps) *Service {
	return &Service{
		codes:         deps.Repositories.codes,
		transactor:    deps.Transactor,
		clock:         deps.Clock,
		accounts:      deps.Accounts,
		directory:     deps.Directory,
		authenticator: deps.Authenticator,
		settings:      deps.Settings,
		mailer:        deps.Mailer,
		auditLog:      deps.AuditLog,
		logger:        deps.Logger,
	}
}

// SendCode asks for root admin's second factor for the binding. Without the two-factor requirement
// there is nothing to confirm. With a connected app nothing is sent and the app code is expected;
// otherwise a code is emailed for the binding in the current session. A newer emailed code
// replaces older ones, and one code a minute may be sent.
func (s *Service) SendCode(ctx context.Context, actor identity.Actor, binding Binding) (Confirmation, error) {
	sessionSha256, hasSession := identity.GetSessionSha256(ctx)
	if !hasSession {
		return Confirmation{}, errNoSession
	}

	rootAdmin, err := s.directory.RequireRootAdmin(ctx, actor)
	if err != nil {
		return Confirmation{}, err
	}

	isRequired, err := s.isTwoFactorRequired(ctx)
	if err != nil {
		return Confirmation{}, err
	}

	if !isRequired {
		return Confirmation{}, ErrProofUnavailable
	}

	_, hasApp, err := s.authenticator.GetConnectedApp(ctx, actor.UserID)
	if err != nil {
		return Confirmation{}, err
	}

	if hasApp {
		return Confirmation{Method: MethodAuthenticatorApp, ExpiresAt: nil}, nil
	}

	expiresAt, err := s.sendEmailedCode(ctx, rootAdmin, sessionSha256, binding)
	if err != nil {
		return Confirmation{}, err
	}

	return Confirmation{Method: MethodEmail, ExpiresAt: &expiresAt}, nil
}

// Verify consumes root admin's app code or recovery code once when root admin has an app, and the
// emailed code for the binding in the current session otherwise. Any refusal answers
// identity_proof_invalid without telling which part was wrong, apart from the per-account limit on
// wrong app codes.
func (s *Service) Verify(ctx context.Context, actor identity.Actor, binding Binding, code *string) error {
	sessionSha256, hasSession := identity.GetSessionSha256(ctx)
	if !hasSession {
		return errNoSession
	}

	rootAdmin, err := s.directory.RequireRootAdmin(ctx, actor)
	if err != nil {
		return err
	}

	if code == nil || *code == "" {
		return ErrProofInvalid
	}

	_, hasApp, err := s.authenticator.GetConnectedApp(ctx, rootAdmin.ID)
	if err != nil {
		return err
	}

	if hasApp {
		return s.verifyAppCode(ctx, rootAdmin, *code)
	}

	return s.verifyCode(ctx, actor, sessionSha256, binding, *code)
}

func (s *Service) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	return s.codes.deleteExpiredBefore(ctx, now.Add(-expiredCodeRetention))
}

func (s *Service) sendEmailedCode(
	ctx context.Context,
	rootAdmin identity.User,
	sessionSha256 string,
	binding Binding,
) (time.Time, error) {
	record, err := s.getRecord(ctx, rootAdmin.ID)
	if err != nil {
		return time.Time{}, err
	}

	if !s.canEmail(record) {
		return time.Time{}, ErrProofUnavailable
	}

	now := s.clock.Now()

	newestCreatedAt, err := s.codes.getNewestCreatedAt(ctx, rootAdmin.ID)
	if err != nil {
		return time.Time{}, err
	}

	if newestCreatedAt != nil && now.Sub(*newestCreatedAt) < codeRequestInterval {
		return time.Time{}, ErrRateLimited
	}

	code, err := identity.GenerateSixDigitCode()
	if err != nil {
		return time.Time{}, err
	}

	hashedCode, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return time.Time{}, err
	}

	expiresAt := now.Add(codeLifetime)

	err = s.codes.insert(ctx, IdentityCode{
		ID:                 uuid.New(),
		UserID:             rootAdmin.ID,
		SessionSha256:      sessionSha256,
		Purpose:            binding.Purpose,
		Subject:            binding.Subject,
		RecipientEmail:     record.Email,
		HashedCode:         string(hashedCode),
		ExpiresAt:          expiresAt,
		IsUsed:             false,
		FailedAttemptCount: 0,
		CreatedAt:          now,
	})
	if err != nil {
		return time.Time{}, err
	}

	if err := s.mailer.SendEmail(
		ctx,
		record.Email,
		identityCodeSubject,
		identityCodeBody(code, binding.Purpose),
	); err != nil {
		s.logger.WarnContext(ctx, "identity confirmation code was not delivered", slog.Any("error", err))

		return time.Time{}, ErrProofUnavailable
	}

	err = s.auditLog.Record(ctx, audit.Event{
		Author:      rootAdmin.GetAuditAuthor(),
		WorkspaceID: nil,
		Message:     "Identity confirmation code sent for " + describePurpose(binding.Purpose),
	})
	if err != nil {
		return time.Time{}, err
	}

	return expiresAt, nil
}

func (s *Service) verifyAppCode(ctx context.Context, rootAdmin identity.User, code string) error {
	isAccepted, err := s.authenticator.CheckCode(ctx, rootAdmin, code)
	if err != nil {
		return err
	}

	if !isAccepted {
		s.logger.WarnContext(ctx, "identity confirmation refused: wrong app code")

		return ErrProofInvalid
	}

	return nil
}

func (s *Service) verifyCode(
	ctx context.Context,
	actor identity.Actor,
	sessionSha256 string,
	binding Binding,
	code string,
) error {
	isAccepted := false

	err := s.transactor.Transaction(ctx, func(txCtx context.Context) error {
		storedCode, isFound, err := s.codes.getNewestForUpdate(txCtx, actor.UserID, sessionSha256, binding)
		if err != nil || !isFound {
			return err
		}

		record, err := s.getRecord(txCtx, actor.UserID)
		if err != nil {
			return err
		}

		if !storedCode.IsUsable(s.clock.Now()) || storedCode.RecipientEmail != record.Email {
			return nil
		}

		if bcrypt.CompareHashAndPassword([]byte(storedCode.HashedCode), []byte(code)) != nil {
			return s.codes.incrementFailedAttemptCount(txCtx, storedCode.ID)
		}

		isAccepted = true

		return s.codes.markUsed(txCtx, storedCode.ID)
	})
	if err != nil {
		return err
	}

	if !isAccepted {
		s.logger.WarnContext(ctx, "identity confirmation refused: unusable code")

		return ErrProofInvalid
	}

	return nil
}

func (s *Service) getRecord(ctx context.Context, userID uuid.UUID) (accounts.UserRecord, error) {
	record, isFound, err := s.accounts.GetRecord(ctx, userID)
	if err != nil {
		return accounts.UserRecord{}, err
	}

	if !isFound {
		return accounts.UserRecord{}, accounts.ErrUserNotFound
	}

	return record, nil
}

func (s *Service) isTwoFactorRequired(ctx context.Context) (bool, error) {
	requirement, err := s.settings.GetTwoFactorRequirement(ctx)
	if err != nil {
		return false, err
	}

	return requirement.IsRequired(), nil
}

func (s *Service) canEmail(record accounts.UserRecord) bool {
	return s.mailer.IsConfigured() && identity.IsEmailAddress(record.Email)
}

func describePurpose(purpose Purpose) string {
	switch purpose {
	case PurposeMasterKeyExport:
		return "a master key export"
	case PurposeInstanceRecovery:
		return "applying a recovery"
	}

	return "an operation"
}
