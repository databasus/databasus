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
	"databasus-backend/internal/lib/ratelimit"
	"databasus-backend/internal/modules/users/internal/accounts"
	"databasus-backend/internal/modules/users/internal/identity"
)

const (
	codeRequestInterval = time.Minute

	passwordRateLimitScope  = "identity-proof-password"
	passwordRateLimit       = 5
	passwordRateLimitWindow = 10 * time.Minute

	// expiredCodeRetention keeps used and expired codes briefly, so the one-a-minute rule still sees
	// a code that expired a moment ago.
	expiredCodeRetention = time.Hour
)

type Deps struct {
	Repositories Repositories
	Transactor   *database.Transactor
	Clock        clock.Source
	Accounts     *accounts.Service
	Directory    *accounts.Directory
	Mailer       mailer.Sender
	RateLimiter  ratelimit.Counter
	AuditLog     *audit.Log
	Logger       *slog.Logger
}

type Service struct {
	codes       *identityCodeRepository
	transactor  *database.Transactor
	clock       clock.Source
	accounts    *accounts.Service
	directory   *accounts.Directory
	mailer      mailer.Sender
	rateLimiter ratelimit.Counter
	auditLog    *audit.Log
	logger      *slog.Logger
}

func NewService(deps Deps) *Service {
	return &Service{
		codes:       deps.Repositories.codes,
		transactor:  deps.Transactor,
		clock:       deps.Clock,
		accounts:    deps.Accounts,
		directory:   deps.Directory,
		mailer:      deps.Mailer,
		rateLimiter: deps.RateLimiter,
		auditLog:    deps.AuditLog,
		logger:      deps.Logger,
	}
}

// GetMethods lists the confirmations root admin can give now. An empty list leaves only the host
// file, which the interface explains.
func (s *Service) GetMethods(ctx context.Context, actor identity.Actor) ([]Method, error) {
	if _, err := s.directory.RequireRootAdmin(ctx, actor); err != nil {
		return nil, err
	}

	record, err := s.getRecord(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}

	methods := []Method{}
	if record.HasPassword() {
		methods = append(methods, MethodPassword)
	}

	if s.canEmail(record) {
		methods = append(methods, MethodEmailCode)
	}

	return methods, nil
}

// SendCode emails root admin a code for the binding in the current session. A newer code replaces
// older ones, and one code a minute may be sent.
func (s *Service) SendCode(ctx context.Context, actor identity.Actor, binding Binding) (time.Time, error) {
	sessionSha256, hasSession := identity.GetSessionSha256(ctx)
	if !hasSession {
		return time.Time{}, errNoSession
	}

	rootAdmin, err := s.directory.RequireRootAdmin(ctx, actor)
	if err != nil {
		return time.Time{}, err
	}

	record, err := s.getRecord(ctx, actor.UserID)
	if err != nil {
		return time.Time{}, err
	}

	if !s.canEmail(record) {
		return time.Time{}, ErrProofUnavailable
	}

	now := s.clock.Now()

	newestCreatedAt, err := s.codes.getNewestCreatedAt(ctx, actor.UserID)
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
		UserID:             actor.UserID,
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

// Verify accepts the proof once for the binding in the current session. Any refusal answers
// identity_proof_invalid without telling which part was wrong.
func (s *Service) Verify(ctx context.Context, actor identity.Actor, binding Binding, proof Proof) error {
	sessionSha256, hasSession := identity.GetSessionSha256(ctx)
	if !hasSession {
		return errNoSession
	}

	if _, err := s.directory.RequireRootAdmin(ctx, actor); err != nil {
		return err
	}

	isPassword := proof.Password != nil && *proof.Password != ""
	isCode := proof.Code != nil && *proof.Code != ""

	switch {
	case isPassword && !isCode:
		return s.verifyPassword(ctx, actor, *proof.Password)
	case isCode && !isPassword:
		return s.verifyCode(ctx, actor, sessionSha256, binding, *proof.Code)
	default:
		return ErrProofInvalid
	}
}

// DeleteExpired removes codes nobody can use any longer.
func (s *Service) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	return s.codes.deleteExpiredBefore(ctx, now.Add(-expiredCodeRetention))
}

func (s *Service) verifyPassword(ctx context.Context, actor identity.Actor, password string) error {
	isAllowed, err := s.rateLimiter.RecordAttemptAndCheckIsAllowed(ctx, ratelimit.Attempt{
		Scope:      passwordRateLimitScope,
		Identifier: actor.UserID.String(),
		Limit:      passwordRateLimit,
		Window:     passwordRateLimitWindow,
	})
	if err != nil || !isAllowed {
		return ErrRateLimited
	}

	record, err := s.getRecord(ctx, actor.UserID)
	if err != nil {
		return err
	}

	if !record.HasPassword() {
		return ErrProofInvalid
	}

	if bcrypt.CompareHashAndPassword([]byte(*record.HashedPassword), []byte(password)) != nil {
		s.logger.WarnContext(ctx, "identity confirmation refused: incorrect password")

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
