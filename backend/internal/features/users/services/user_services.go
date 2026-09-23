package users_services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"databasus-backend/internal/config"
	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	"databasus-backend/internal/features/encryption/secrets"
	users_dto "databasus-backend/internal/features/users/dto"
	users_enums "databasus-backend/internal/features/users/enums"
	users_errors "databasus-backend/internal/features/users/errors"
	users_interfaces "databasus-backend/internal/features/users/interfaces"
	users_models "databasus-backend/internal/features/users/models"
	users_repositories "databasus-backend/internal/features/users/repositories"
	"databasus-backend/internal/util/ratelimiter"
)

// bootstrapAdminIndexName is the partial unique index that admits a single
// bootstrap administrator. Naming it is what lets a lost race be told apart from
// a duplicate address, which must keep failing as it always has.
const bootstrapAdminIndexName = "idx_users_is_root_admin"

type UserService struct {
	userRepository          *users_repositories.UserRepository
	secretKeyService        *secrets.SecretKeyService
	settingsService         *SettingsService
	auditLogWriter          users_interfaces.AuditLogWriter
	emailSender             users_interfaces.EmailSender
	passwordResetRepository *users_repositories.PasswordResetRepository
	twoFactorRepository     *users_repositories.TwoFactorRepository
	rateLimiter             ratelimiter.Counter
	logger                  *slog.Logger
}

func (s *UserService) SetAuditLogWriter(writer users_interfaces.AuditLogWriter) {
	s.auditLogWriter = writer
}

func (s *UserService) SetEmailSender(sender users_interfaces.EmailSender) {
	s.emailSender = sender
}

func (s *UserService) SignUp(ctx context.Context, request *users_dto.SignUpRequestDTO) (*users_models.User, error) {
	existingUser, err := s.userRepository.GetUserByEmail(ctx, request.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	if existingUser != nil && existingUser.Status != users_enums.UserStatusInvited {
		return nil, errors.New("user with this email already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	hashedPasswordStr := string(hashedPassword)

	// If user exists with INVITED status, activate them and set password
	if existingUser != nil && existingUser.Status == users_enums.UserStatusInvited {
		if err := s.userRepository.UpdateUserPassword(
			existingUser.ID,
			hashedPasswordStr,
		); err != nil {
			return nil, fmt.Errorf("failed to set password: %w", err)
		}

		if err := s.userRepository.UpdateUserStatus(
			existingUser.ID,
			users_enums.UserStatusActive,
		); err != nil {
			return nil, fmt.Errorf("failed to activate user: %w", err)
		}

		name := request.Name
		if err := s.userRepository.UpdateUserInfo(existingUser.ID, &name, nil); err != nil {
			return nil, fmt.Errorf("failed to update name: %w", err)
		}

		// Fetch updated user to ensure we have the latest data
		updatedUser, err := s.userRepository.GetUserByID(ctx, existingUser.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get updated user: %w", err)
		}

		s.writeAuditLog(ctx, audit_logs_models.AuditEntry{
			Message:     fmt.Sprintf("Invited user completed registration: %s", updatedUser.Email),
			UserID:      &updatedUser.ID,
			WorkspaceID: nil,
		})

		return updatedUser, nil
	}

	user, err := s.createAccountClaimingEmptyInstance(
		ctx,
		func(isTakingTheInstance bool) *users_models.User {
			return buildRegisteredUser(request, hashedPasswordStr, isTakingTheInstance)
		},
	)
	if err != nil {
		return nil, err
	}

	s.writeAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("User registered with email: %s", user.Email),
		UserID:      &user.ID,
		WorkspaceID: nil,
	})

	s.writeInstanceClaimedAuditLog(ctx, user)

	return user, nil
}

func isBootstrapAdminTaken(err error) bool {
	var pgError *pgconn.PgError

	return errors.As(err, &pgError) && pgError.ConstraintName == bootstrapAdminIndexName
}

func (s *UserService) SignIn(
	ctx context.Context,
	request *users_dto.SignInRequestDTO,
) (*users_dto.SignInOutcome, error) {
	// Failed sign-ins are the only signal that someone is guessing credentials, so every branch that
	// refuses one is logged. The handler's addresses are masked centrally by the log handler.
	user, err := s.userRepository.GetUserByEmail(ctx, request.Email)
	if err != nil {
		s.logger.WarnContext(ctx, "sign-in refused: email lookup failed", "email", request.Email, "error", err)

		return nil, errors.New("user with this email does not exist")
	}

	if user == nil {
		s.logger.WarnContext(ctx, "sign-in refused: no user with this email", "email", request.Email)

		return nil, errors.New("user with this email does not exist")
	}

	if user.Status == users_enums.UserStatusInvited {
		s.logger.WarnContext(ctx, "sign-in refused: the account has not completed sign-up",
			"user_id", user.ID)

		return nil, errors.New("user account is not passed sign up yet")
	}

	if user.Status != users_enums.UserStatusActive {
		s.logger.WarnContext(ctx, "sign-in refused: the account is deactivated", "user_id", user.ID)

		return nil, errors.New("user account is deactivated")
	}

	// An account created through an external identity provider has no password
	// hash at all. Refuse it the way a wrong password is refused, rather than
	// dereferencing nil and taking the process down.
	if !user.HasPassword() {
		s.logger.WarnContext(ctx, "sign-in refused: the account has no password", "user_id", user.ID)

		return nil, errors.New("password is incorrect")
	}

	err = bcrypt.CompareHashAndPassword([]byte(*user.HashedPassword), []byte(request.Password))
	if err != nil {
		s.logger.WarnContext(ctx, "sign-in refused: incorrect password", "user_id", user.ID)

		return nil, errors.New("password is incorrect")
	}

	settings, err := s.settingsService.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	if settings.IsTwoFactorAuthRequired {
		pendingSignIn, err := s.StartTwoFactorSignIn(ctx, user)
		if err != nil {
			return nil, err
		}

		return &users_dto.SignInOutcome{PendingSignIn: pendingSignIn}, nil
	}

	response, err := s.completeSignIn(ctx, user)
	if err != nil {
		return nil, err
	}

	return &users_dto.SignInOutcome{CompletedSignIn: response}, nil
}

func (s *UserService) GetUserFromToken(ctx context.Context, token string) (*users_models.User, error) {
	secretKey, err := s.secretKeyService.GetSecretKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get secret key: %w", err)
	}

	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok && parsedToken.Valid {
		userIDStr, ok := claims["sub"].(string)
		if !ok {
			return nil, errors.New("invalid token claims")
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return nil, errors.New("invalid token claims")
		}

		user, err := s.userRepository.GetUserByID(ctx, userID)
		if err != nil {
			return nil, err
		}

		// Check if user is active
		if !user.IsActiveUser() {
			return nil, errors.New("user account is deactivated")
		}

		if passwordCreationTimeUnix, ok := claims["passwordCreationTime"].(float64); ok {
			tokenPasswordTime := time.Unix(int64(passwordCreationTimeUnix), 0)

			tokenTimeSeconds := tokenPasswordTime.Truncate(time.Second)
			userTimeSeconds := user.PasswordCreationTime.Truncate(time.Second)

			if !tokenTimeSeconds.Equal(userTimeSeconds) {
				return nil, errors.New("password has been changed, please sign in again")
			}
		} else {
			return nil, errors.New("invalid token claims: missing password creation time")
		}

		return user, nil
	}

	return nil, errors.New("invalid token")
}

func (s *UserService) GenerateAccessToken(
	ctx context.Context,
	user *users_models.User,
) (*users_dto.SignInResponseDTO, error) {
	secretKey, err := s.secretKeyService.GetSecretKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get secret key: %w", err)
	}

	tenYearsExpiration := time.Now().UTC().Add(time.Hour * 24 * 365 * 10)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":                  user.ID.String(),
		"exp":                  tenYearsExpiration.Unix(),
		"iat":                  time.Now().UTC().Unix(),
		"role":                 string(user.Role),
		"passwordCreationTime": user.PasswordCreationTime.Unix(),
	})

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &users_dto.SignInResponseDTO{
		UserID: user.ID,
		Email:  user.Email,
		Token:  tokenString,
	}, nil
}

func (s *UserService) GetUsersCount() (int64, error) {
	return s.userRepository.GetUsersCount()
}

// The entry screen asks this to decide between offering a sign-in and offering
// the registration that claims the instance.
func (s *UserService) HasAnyUser() (bool, error) {
	isUnclaimed, err := s.isInstanceUnclaimed()
	if err != nil {
		return false, err
	}

	return !isUnclaimed, nil
}

// IsSignUpAllowed reports whether a new account may currently be created.
// An empty instance always accepts a registration, because the first account is
// how it gains an administrator, whatever the registration policy says.
func (s *UserService) IsSignUpAllowed(ctx context.Context) (bool, error) {
	isUnclaimed, err := s.isInstanceUnclaimed()
	if err != nil {
		return false, err
	}

	if isUnclaimed {
		return true, nil
	}

	settings, err := s.settingsService.GetSettings(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get settings: %w", err)
	}

	return settings.IsAllowExternalRegistrations, nil
}

func (s *UserService) ChangeUserPasswordByEmail(ctx context.Context, email, newPassword string) error {
	user, err := s.userRepository.GetUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return errors.New("user with this email does not exist")
	}

	return s.ChangeUserPassword(ctx, user.ID, newPassword)
}

func (s *UserService) ChangeUserPassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	if err := s.userRepository.UpdateUserPassword(userID, string(hashedPassword)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	s.writeAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     "Password changed",
		UserID:      &userID,
		WorkspaceID: nil,
	})

	return nil
}

func (s *UserService) InviteUser(
	ctx context.Context,
	request *users_dto.InviteUserRequestDTO,
	invitedBy *users_models.User,
) (*users_dto.InviteUserResponseDTO, error) {
	// Get settings to check permissions
	settings, err := s.settingsService.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	// Check if user has permission to invite
	if !invitedBy.CanInviteUsers(settings) {
		return nil, users_errors.ErrInsufficientPermissionsToInviteUsers
	}

	// Check if user already exists
	existingUser, err := s.userRepository.GetUserByEmail(ctx, request.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	user := &users_models.User{
		ID:                   uuid.New(),
		Email:                request.Email,
		Name:                 "User",
		HashedPassword:       nil,
		PasswordCreationTime: time.Now().UTC(),
		Role:                 users_enums.UserRoleMember,
		Status:               users_enums.UserStatusInvited,
		CreatedAt:            time.Now().UTC(),
	}

	if err := s.userRepository.CreateUser(user); err != nil {
		return nil, fmt.Errorf("failed to create invited user: %w", err)
	}

	message := fmt.Sprintf("User invited: %s", request.Email)
	if request.IntendedWorkspaceID != nil {
		message += " for workspace"
	}
	s.writeAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     message,
		UserID:      &invitedBy.ID,
		WorkspaceID: request.IntendedWorkspaceID,
	})

	return &users_dto.InviteUserResponseDTO{
		ID:                    user.ID,
		Email:                 user.Email,
		IntendedWorkspaceID:   request.IntendedWorkspaceID,
		IntendedWorkspaceRole: request.IntendedWorkspaceRole,
		CreatedAt:             user.CreatedAt,
	}, nil
}

func (s *UserService) GetUserByID(ctx context.Context, userID uuid.UUID) (*users_models.User, error) {
	return s.userRepository.GetUserByID(ctx, userID)
}

func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*users_models.User, error) {
	return s.userRepository.GetUserByEmail(ctx, email)
}

func (s *UserService) GetCurrentUserProfile(
	ctx context.Context,
	user *users_models.User,
) *users_dto.UserProfileResponseDTO {
	return &users_dto.UserProfileResponseDTO{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		IsActive:  user.IsActiveUser(),
		CreatedAt: user.CreatedAt,
	}
}

func (s *UserService) UpdateUserInfo(
	ctx context.Context,
	userID uuid.UUID,
	request *users_dto.UpdateUserInfoRequestDTO,
) error {
	user, err := s.userRepository.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	oldEmail := user.Email
	oldName := user.Name

	if request.Email != nil && *request.Email != user.Email {
		existingUser, err := s.userRepository.GetUserByEmail(ctx, *request.Email)
		if err != nil {
			return fmt.Errorf("failed to check email: %w", err)
		}
		if existingUser != nil {
			return errors.New("email is already taken by another user")
		}
	}

	if err := s.userRepository.UpdateUserInfo(userID, request.Name, request.Email); err != nil {
		return fmt.Errorf("failed to update user info: %w", err)
	}

	var auditMessages []string
	if request.Email != nil && *request.Email != oldEmail {
		auditMessages = append(
			auditMessages,
			fmt.Sprintf("Email changed from '%s' to '%s'", oldEmail, *request.Email),
		)
	}
	if request.Name != nil && *request.Name != oldName {
		auditMessages = append(
			auditMessages,
			fmt.Sprintf("Name changed from '%s' to '%s'", oldName, *request.Name),
		)
	}

	if len(auditMessages) > 0 {
		for _, message := range auditMessages {
			s.writeAuditLog(ctx, audit_logs_models.AuditEntry{
				Message:     message,
				UserID:      &userID,
				WorkspaceID: nil,
			})
		}
	} else {
		s.writeAuditLog(ctx, audit_logs_models.AuditEntry{
			Message:     "User info updated",
			UserID:      &userID,
			WorkspaceID: nil,
		})
	}

	return nil
}

func (s *UserService) HandleGitHubOAuth(
	ctx context.Context,
	code, redirectUri string,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	return s.handleGitHubOAuthWithEndpoint(
		ctx,
		code,
		redirectUri,
		github.Endpoint,
		"https://api.github.com/user",
	)
}

func (s *UserService) HandleGoogleOAuth(
	ctx context.Context,
	code, redirectUri string,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	return s.handleGoogleOAuthWithEndpoint(
		ctx,
		code,
		redirectUri,
		google.Endpoint,
		"https://www.googleapis.com/oauth2/v2/userinfo",
	)
}

func (s *UserService) SendResetPasswordCode(ctx context.Context, email string) error {
	user, err := s.userRepository.GetUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Silently succeed for non-existent users to prevent enumeration attacks
	if user == nil {
		return nil
	}

	// Only active users can reset passwords
	if user.Status != users_enums.UserStatusActive {
		return errors.New("only active users can reset their password")
	}

	// Check rate limiting - max 3 codes per hour
	oneHourAgo := time.Now().UTC().Add(-1 * time.Hour)
	recentCount, err := s.passwordResetRepository.CountRecentCodesByUserID(user.ID, oneHourAgo)
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}

	if recentCount >= 3 {
		return errors.New("too many password reset attempts, please try again later")
	}

	code, err := generateSixDigitCode()
	if err != nil {
		return err
	}

	hashedCode, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash code: %w", err)
	}

	// Store in database with 1 hour expiration
	resetCode := &users_models.PasswordResetCode{
		ID:         uuid.New(),
		UserID:     user.ID,
		HashedCode: string(hashedCode),
		ExpiresAt:  time.Now().UTC().Add(1 * time.Hour),
		IsUsed:     false,
		CreatedAt:  time.Now().UTC(),
	}

	if err := s.passwordResetRepository.CreateResetCode(resetCode); err != nil {
		return fmt.Errorf("failed to create reset code: %w", err)
	}

	// Send email with code
	if s.emailSender != nil {
		subject := "Password Reset Code"
		body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin: 0; padding: 0; font-family: Arial, sans-serif; background-color: #f4f4f4;">
    <div style="max-width: 600px; margin: 0 auto; background-color: #ffffff; padding: 20px;">
        <h2 style="color: #333333; margin-bottom: 20px;">Password Reset Request</h2>
        <p style="color: #666666; line-height: 1.6; margin-bottom: 20px;">
            You have requested to reset your password. Please use the following code to complete the password reset process:
        </p>
        <div style="background-color: #f8f9fa; border: 2px solid #e9ecef; border-radius: 8px; padding: 20px; text-align: center; margin: 30px 0;">
            <h1 style="color: #2c3e50; font-size: 36px; margin: 0; letter-spacing: 8px; font-family: monospace;">%s</h1>
        </div>
        <p style="color: #666666; line-height: 1.6; margin-bottom: 20px;">
            This code will expire in <strong>1 hour</strong>.
        </p>
        <p style="color: #666666; line-height: 1.6; margin-bottom: 20px;">
            If you did not request a password reset, please ignore this email. Your password will remain unchanged.
        </p>
        <hr style="border: none; border-top: 1px solid #e9ecef; margin: 30px 0;">
        <p style="color: #999999; font-size: 12px; line-height: 1.6;">
            This is an automated message. Please do not reply to this email.
        </p>
    </div>
</body>
</html>
`, code)

		if err := s.emailSender.SendEmail(ctx, user.Email, subject, body); err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
	}

	// Audit log
	s.writeAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Password reset code sent to: %s", user.Email),
		UserID:      &user.ID,
		WorkspaceID: nil,
	})

	return nil
}

func (s *UserService) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	user, err := s.userRepository.GetUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return errors.New("user with this email does not exist")
	}

	// Get valid reset code for user
	resetCode, err := s.passwordResetRepository.GetValidCodeByUserID(user.ID)
	if err != nil {
		return errors.New("invalid or expired reset code")
	}

	// Verify code matches
	err = bcrypt.CompareHashAndPassword([]byte(resetCode.HashedCode), []byte(code))
	if err != nil {
		return errors.New("invalid reset code")
	}

	// Mark code as used
	if err := s.passwordResetRepository.MarkCodeAsUsed(resetCode.ID); err != nil {
		return fmt.Errorf("failed to mark code as used: %w", err)
	}

	// Update user password
	if err := s.ChangeUserPassword(ctx, user.ID, newPassword); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Audit log
	s.writeAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     "Password reset via email code",
		UserID:      &user.ID,
		WorkspaceID: nil,
	})

	return nil
}

// The audit trail records a sign-in where access is actually granted, which is
// the password step with one factor and the code step with two.
func (s *UserService) completeSignIn(
	ctx context.Context,
	user *users_models.User,
) (*users_dto.SignInResponseDTO, error) {
	s.logger.InfoContext(ctx, "user signed in", "user_id", user.ID)

	response, err := s.GenerateAccessToken(ctx, user)
	if err != nil {
		return nil, err
	}

	s.writeAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("User signed in with email: %s", user.Email),
		UserID:      &user.ID,
		WorkspaceID: nil,
	})

	return response, nil
}

// Every audit entry this service writes goes through here, because the console
// path reaches the service without installing a writer and a missing entry must
// not take the process down.
func (s *UserService) writeAuditLog(ctx context.Context, entry audit_logs_models.AuditEntry) {
	if s.auditLogWriter == nil {
		return
	}

	s.auditLogWriter.WriteAuditLog(ctx, entry)
}

func (s *UserService) isInstanceUnclaimed() (bool, error) {
	usersCount, err := s.userRepository.GetUsersCount()
	if err != nil {
		return false, fmt.Errorf("failed to get users count: %w", err)
	}

	return usersCount == 0, nil
}

func (s *UserService) checkExternalRegistrationAllowed(ctx context.Context) error {
	settings, err := s.settingsService.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	if !settings.IsAllowExternalRegistrations {
		return errors.New("external registration is disabled")
	}

	return nil
}

// Both entry points that insert a new account share this path. An instance
// holding no account hands the account the administrator role and records it as
// the bootstrap administrator, and the question is asked before the registration
// policy rather than inside the insert, because an empty instance whose policy
// forbids registration would otherwise never gain an administrator at all.
//
// Losing the race for that record is not a failure: the partial unique index
// admits one flagged account, so the loser is built again as an ordinary
// account - now under the registration policy it was allowed to skip.
func (s *UserService) createAccountClaimingEmptyInstance(
	ctx context.Context,
	build func(isTakingTheInstance bool) *users_models.User,
) (*users_models.User, error) {
	isTakingTheInstance, err := s.isInstanceUnclaimed()
	if err != nil {
		return nil, err
	}

	if !isTakingTheInstance {
		if err := s.checkExternalRegistrationAllowed(ctx); err != nil {
			return nil, err
		}
	}

	user := build(isTakingTheInstance)

	err = s.userRepository.CreateUser(user)
	if err == nil {
		return user, nil
	}

	// Anything but the contested bootstrap record - a duplicate address above all -
	// stays the failure it has always been.
	if !isTakingTheInstance || !isBootstrapAdminTaken(err) {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := s.checkExternalRegistrationAllowed(ctx); err != nil {
		return nil, err
	}

	user = build(false)

	if err := s.userRepository.CreateUser(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func buildRegisteredUser(
	request *users_dto.SignUpRequestDTO,
	hashedPassword string,
	isTakingTheInstance bool,
) *users_models.User {
	return &users_models.User{
		ID:                   uuid.New(),
		Email:                request.Email,
		Name:                 request.Name,
		HashedPassword:       &hashedPassword,
		PasswordCreationTime: time.Now().UTC(),
		Role:                 roleForNewAccount(isTakingTheInstance),
		Status:               users_enums.UserStatusActive,
		IsRootAdmin:          isTakingTheInstance,
		CreatedAt:            time.Now().UTC(),
	}
}

func buildOAuthUser(
	email, name string,
	githubOAuthID, googleOAuthID *string,
	isTakingTheInstance bool,
) *users_models.User {
	return &users_models.User{
		ID:                   uuid.New(),
		Email:                email,
		Name:                 name,
		HashedPassword:       nil,
		PasswordCreationTime: time.Now().UTC(),
		Role:                 roleForNewAccount(isTakingTheInstance),
		Status:               users_enums.UserStatusActive,
		GitHubOAuthID:        githubOAuthID,
		GoogleOAuthID:        googleOAuthID,
		IsRootAdmin:          isTakingTheInstance,
		CreatedAt:            time.Now().UTC(),
	}
}

func roleForNewAccount(isTakingTheInstance bool) users_enums.UserRole {
	if isTakingTheInstance {
		return users_enums.UserRoleAdmin
	}

	return users_enums.UserRoleMember
}

// The log has to explain an administrator nobody appointed.
func (s *UserService) writeInstanceClaimedAuditLog(ctx context.Context, user *users_models.User) {
	if !user.IsRootAdmin {
		return
	}

	s.writeAuditLog(ctx, audit_logs_models.AuditEntry{
		Message: fmt.Sprintf(
			"First account on the instance now administers it: %s",
			user.Email,
		),
		UserID:      &user.ID,
		WorkspaceID: nil,
	})
}

func (s *UserService) handleGitHubOAuthWithEndpoint(
	ctx context.Context,
	code, redirectUri string,
	endpoint oauth2.Endpoint,
	userAPIURL string,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	env := config.GetEnv()

	oauthConfig := &oauth2.Config{
		ClientID:     env.GitHubClientID,
		ClientSecret: env.GitHubClientSecret,
		RedirectURL:  redirectUri,
		Endpoint:     endpoint,
		Scopes:       []string{"user:email"},
	}

	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		s.logger.ErrorContext(ctx, "oauth code exchange failed", "error", err)

		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	client := oauthConfig.Client(ctx, token)
	githubReq, err := http.NewRequestWithContext(ctx, "GET", userAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}
	resp, err := client.Do(githubReq)
	if err != nil {
		s.logger.ErrorContext(ctx, "oauth user info request failed", "error", err)

		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var githubUser struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Login string `json:"login"`
	}

	if err := json.Unmarshal(body, &githubUser); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	email := githubUser.Email
	if email == "" {
		email, err = s.fetchGitHubPrimaryEmail(ctx, client, userAPIURL)
		if err != nil {
			return nil, err
		}
	}

	name := githubUser.Name
	if name == "" {
		name = githubUser.Login
	}

	oauthID := fmt.Sprintf("%d", githubUser.ID)
	return s.getOrCreateUserFromOAuth(ctx, oauthID, email, name, "github")
}

func (s *UserService) handleGoogleOAuthWithEndpoint(
	ctx context.Context,
	code, redirectUri string,
	endpoint oauth2.Endpoint,
	userAPIURL string,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	env := config.GetEnv()

	oauthConfig := &oauth2.Config{
		ClientID:     env.GoogleClientID,
		ClientSecret: env.GoogleClientSecret,
		RedirectURL:  redirectUri,
		Endpoint:     endpoint,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}

	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		s.logger.ErrorContext(ctx, "oauth code exchange failed", "error", err)

		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	client := oauthConfig.Client(ctx, token)
	googleReq, err := http.NewRequestWithContext(ctx, "GET", userAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}
	resp, err := client.Do(googleReq)
	if err != nil {
		s.logger.ErrorContext(ctx, "oauth user info request failed", "error", err)

		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var googleUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	if err := json.Unmarshal(body, &googleUser); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	if googleUser.Email == "" {
		return nil, errors.New("google account has no email")
	}

	name := googleUser.Name
	if name == "" {
		name = "User"
	}

	return s.getOrCreateUserFromOAuth(ctx, googleUser.ID, googleUser.Email, name, "google")
}

func (s *UserService) getOrCreateUserFromOAuth(
	ctx context.Context,
	oauthID, email, name, provider string,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	var existingUser *users_models.User
	var err error

	if provider == "github" {
		existingUser, err = s.userRepository.GetUserByGitHubOAuthID(oauthID)
	} else {
		existingUser, err = s.userRepository.GetUserByGoogleOAuthID(oauthID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to check OAuth ID: %w", err)
	}

	if existingUser != nil {
		tokenResponse, err := s.GenerateAccessToken(ctx, existingUser)
		if err != nil {
			return nil, err
		}

		s.writeAuditLog(ctx, audit_logs_models.AuditEntry{
			Message:     fmt.Sprintf("User signed in via %s", provider),
			UserID:      &existingUser.ID,
			WorkspaceID: nil,
		})

		return &users_dto.OAuthCallbackResponseDTO{
			UserID:    tokenResponse.UserID,
			Email:     existingUser.Email,
			Token:     tokenResponse.Token,
			IsNewUser: false,
		}, nil
	}

	userByEmail, err := s.userRepository.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}

	if userByEmail != nil {
		if userByEmail.Status == users_enums.UserStatusInvited {
			if err := s.userRepository.UpdateUserStatus(
				userByEmail.ID,
				users_enums.UserStatusActive,
			); err != nil {
				return nil, fmt.Errorf("failed to activate user: %w", err)
			}

			if err := s.userRepository.UpdateUserInfo(userByEmail.ID, &name, nil); err != nil {
				return nil, fmt.Errorf("failed to update name: %w", err)
			}
		}

		oauthColumn := "github_oauth_id"
		if provider == "google" {
			oauthColumn = "google_oauth_id"
		}

		if err := s.userRepository.LinkOAuthID(userByEmail.ID, oauthColumn, oauthID); err != nil {
			return nil, fmt.Errorf("failed to link OAuth ID: %w", err)
		}

		user, err := s.userRepository.GetUserByID(ctx, userByEmail.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get updated user: %w", err)
		}

		tokenResponse, err := s.GenerateAccessToken(ctx, user)
		if err != nil {
			return nil, err
		}

		s.writeAuditLog(ctx, audit_logs_models.AuditEntry{
			Message:     fmt.Sprintf("%s OAuth linked to existing account", provider),
			UserID:      &user.ID,
			WorkspaceID: nil,
		})

		return &users_dto.OAuthCallbackResponseDTO{
			UserID:    tokenResponse.UserID,
			Email:     user.Email,
			Token:     tokenResponse.Token,
			IsNewUser: false,
		}, nil
	}

	var githubOAuthID *string
	var googleOAuthID *string
	if provider == "github" {
		githubOAuthID = &oauthID
	} else {
		googleOAuthID = &oauthID
	}

	newUser, err := s.createAccountClaimingEmptyInstance(
		ctx,
		func(isTakingTheInstance bool) *users_models.User {
			return buildOAuthUser(email, name, githubOAuthID, googleOAuthID, isTakingTheInstance)
		},
	)
	if err != nil {
		return nil, err
	}

	tokenResponse, err := s.GenerateAccessToken(ctx, newUser)
	if err != nil {
		return nil, err
	}

	s.writeAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("User registered via %s OAuth: %s", provider, email),
		UserID:      &newUser.ID,
		WorkspaceID: nil,
	})

	s.writeInstanceClaimedAuditLog(ctx, newUser)

	return &users_dto.OAuthCallbackResponseDTO{
		UserID:    tokenResponse.UserID,
		Email:     newUser.Email,
		Token:     tokenResponse.Token,
		IsNewUser: true,
	}, nil
}

func (s *UserService) fetchGitHubPrimaryEmail(
	ctx context.Context,
	client *http.Client,
	userAPIURL string,
) (string, error) {
	emailsURL := "https://api.github.com/user/emails"
	if userAPIURL != "https://api.github.com/user" {
		baseURL := userAPIURL[:len(userAPIURL)-len("/user")]
		emailsURL = baseURL + "/user/emails"
	}

	emailsReq, err := http.NewRequestWithContext(ctx, "GET", emailsURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create user emails request: %w", err)
	}
	resp, err := client.Do(emailsReq)
	if err != nil {
		return "", fmt.Errorf("failed to get user emails: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("github account has no accessible email")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read emails response: %w", err)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.Unmarshal(body, &emails); err != nil {
		return "", fmt.Errorf("failed to parse emails: %w", err)
	}

	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}

	for _, email := range emails {
		if email.Verified {
			return email.Email, nil
		}
	}

	if len(emails) > 0 {
		return emails[0].Email, nil
	}

	return "", errors.New("github account has no accessible email")
}
