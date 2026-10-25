package users_services

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"

	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	users_dto "databasus-backend/internal/features/users/dto"
	users_errors "databasus-backend/internal/features/users/errors"
	users_interfaces "databasus-backend/internal/features/users/interfaces"
	users_models "databasus-backend/internal/features/users/models"
	users_repositories "databasus-backend/internal/features/users/repositories"
	"databasus-backend/internal/lib/mailer"
)

const (
	testEmailSubject = "Databasus test email"
	testEmailBody    = `<p>This message confirms that Databasus can deliver email through its mail server.</p>`
)

// The gate and the profile form must not disagree about what counts as an
// address, so the gate asks the validator the request binding already uses
// (UpdateUserInfoRequestDTO binds omitempty,email) rather than matching a
// pattern of its own.
var addressValidator = validator.New()

type SettingsService struct {
	userSettingsRepository *users_repositories.UsersSettingsRepository
	userRepository         *users_repositories.UserRepository
	auditLogWriter         users_interfaces.AuditLogWriter
	emailSender            users_interfaces.EmailSender
}

func (s *SettingsService) SetAuditLogWriter(writer users_interfaces.AuditLogWriter) {
	s.auditLogWriter = writer
}

func (s *SettingsService) SetEmailSender(sender users_interfaces.EmailSender) {
	s.emailSender = sender
}

func (s *SettingsService) GetSettings(ctx context.Context) (*users_models.UsersSettings, error) {
	return s.userSettingsRepository.GetSettings(ctx)
}

func (s *SettingsService) GetSettingsResponse(
	ctx context.Context,
) (*users_dto.SettingsResponseDTO, error) {
	settings, err := s.userSettingsRepository.GetSettings(ctx)
	if err != nil {
		return nil, err
	}

	return s.buildSettingsResponse(settings), nil
}

func (s *SettingsService) IsEmailConfigured() bool {
	return s.emailSender != nil && s.emailSender.IsConfigured()
}

func (s *SettingsService) SendTestEmail(ctx context.Context, admin *users_models.User) (string, error) {
	if !s.IsEmailConfigured() {
		return "", users_errors.ErrEmailNotConfigured
	}

	if !isDeliverableAddress(admin.Email) {
		return "", users_errors.ErrAdminEmailMissing
	}

	sendError := s.emailSender.SendEmail(ctx, admin.Email, testEmailSubject, testEmailBody)

	auditMessage := fmt.Sprintf("Test email sent to %s", admin.Email)
	if sendError != nil {
		auditMessage = fmt.Sprintf("Test email to %s failed", admin.Email)
	}

	s.writeAuditLog(ctx, audit_logs_models.AuditEntry{Message: auditMessage, UserID: &admin.ID})

	if sendError != nil {
		return "", sendError
	}

	return admin.Email, nil
}

func (s *SettingsService) UpdateSettings(
	ctx context.Context,
	request users_dto.UpdateSettingsRequestDTO,
	updatedBy *users_models.User,
) (*users_dto.SettingsResponseDTO, error) {
	if !updatedBy.CanUpdateSettings() {
		return nil, fmt.Errorf("insufficient permissions to update settings")
	}

	existingSettings, err := s.userSettingsRepository.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current settings: %w", err)
	}

	if request.IsTwoFactorAuthRequired && !existingSettings.IsTwoFactorAuthRequired {
		if err := s.checkCodesCanReachEveryAdmin(ctx); err != nil {
			return nil, err
		}
	}

	switches := []struct {
		name     string
		current  *bool
		proposed bool
	}{
		{
			"isAllowExternalRegistrations",
			&existingSettings.IsAllowExternalRegistrations,
			request.IsAllowExternalRegistrations,
		},
		{
			"isAllowMemberInvitations",
			&existingSettings.IsAllowMemberInvitations,
			request.IsAllowMemberInvitations,
		},
		{
			"isMemberAllowedToCreateWorkspaces",
			&existingSettings.IsMemberAllowedToCreateWorkspaces,
			request.IsMemberAllowedToCreateWorkspaces,
		},
		{
			"isTwoFactorAuthRequired",
			&existingSettings.IsTwoFactorAuthRequired,
			request.IsTwoFactorAuthRequired,
		},
	}

	auditLogMessages := []string{}

	for _, settingSwitch := range switches {
		if *settingSwitch.current == settingSwitch.proposed {
			continue
		}

		auditLogMessages = append(
			auditLogMessages,
			fmt.Sprintf(
				"%s: %t -> %t",
				settingSwitch.name,
				*settingSwitch.current,
				settingSwitch.proposed,
			),
		)

		*settingSwitch.current = settingSwitch.proposed
	}

	if err := s.userSettingsRepository.UpdateSettings(ctx, existingSettings); err != nil {
		return nil, fmt.Errorf("failed to update settings: %w", err)
	}

	for _, message := range auditLogMessages {
		s.writeAuditLog(ctx, audit_logs_models.AuditEntry{
			Message:     message,
			UserID:      &updatedBy.ID,
			WorkspaceID: nil,
		})
	}

	return s.buildSettingsResponse(existingSettings), nil
}

// DisableTwoFactorAuth is the way back into an instance whose mail server has
// died, so it takes no caller and asks no permission: whoever can run it already
// runs commands inside the instance. The entry it writes is what keeps the host
// from being a way to move a security switch unobserved.
func (s *SettingsService) DisableTwoFactorAuth(ctx context.Context) (bool, error) {
	settings, err := s.userSettingsRepository.GetSettings(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get current settings: %w", err)
	}

	if !settings.IsTwoFactorAuthRequired {
		return false, nil
	}

	settings.IsTwoFactorAuthRequired = false

	if err := s.userSettingsRepository.UpdateSettings(ctx, settings); err != nil {
		return false, fmt.Errorf("failed to update settings: %w", err)
	}

	s.writeAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     "isTwoFactorAuthRequired: true -> false (switched off from the host console)",
		UserID:      nil,
		WorkspaceID: nil,
	})

	return true, nil
}

// Turning the second factor on is what locks an instance out, so both ways that
// can happen are refused here: no mail server at all, and an administrator whose
// address no message can reach.
func (s *SettingsService) checkCodesCanReachEveryAdmin(ctx context.Context) error {
	if !s.IsEmailConfigured() {
		return fmt.Errorf(
			"cannot require two-factor authentication: the instance has no mail server configured",
		)
	}

	admins, err := s.userRepository.GetAdmins(ctx)
	if err != nil {
		return fmt.Errorf("failed to list administrators: %w", err)
	}

	for _, admin := range admins {
		if !admin.IsActiveUser() {
			continue
		}

		if addressValidator.Var(admin.Email, "required,email") != nil {
			return fmt.Errorf(
				"cannot require two-factor authentication: administrator %q has no valid email address",
				admin.Email,
			)
		}
	}

	return nil
}

// The validator keeps agreement with the profile form, and the transport's parser
// keeps an address it would refuse from surfacing as a delivery failure.
func isDeliverableAddress(address string) bool {
	if addressValidator.Var(address, "required,email") != nil {
		return false
	}

	_, err := mailer.ParseRecipient(address)

	return err == nil
}

func (s *SettingsService) buildSettingsResponse(
	settings *users_models.UsersSettings,
) *users_dto.SettingsResponseDTO {
	return &users_dto.SettingsResponseDTO{
		IsAllowExternalRegistrations:      settings.IsAllowExternalRegistrations,
		IsAllowMemberInvitations:          settings.IsAllowMemberInvitations,
		IsMemberAllowedToCreateWorkspaces: settings.IsMemberAllowedToCreateWorkspaces,
		IsTwoFactorAuthRequired:           settings.IsTwoFactorAuthRequired,
		IsEmailConfigured:                 s.IsEmailConfigured(),
	}
}

// The console commands reach this service without installing a writer, and a
// missing entry must not take the process down.
func (s *SettingsService) writeAuditLog(ctx context.Context, entry audit_logs_models.AuditEntry) {
	if s.auditLogWriter == nil {
		return
	}

	s.auditLogWriter.WriteAuditLog(ctx, entry)
}
