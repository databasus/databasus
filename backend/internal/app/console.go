package app

import (
	"context"
	"fmt"
	"log/slog"

	users_dto "databasus-backend/internal/features/users/dto"
	users_services "databasus-backend/internal/features/users/services"
)

type ConsoleCommandKind string

const (
	ConsoleCommandListAdmins       ConsoleCommandKind = "list-admins"
	ConsoleCommandCreateUser       ConsoleCommandKind = "create-user"
	ConsoleCommandResetPassword    ConsoleCommandKind = "reset-password"
	ConsoleCommandDisableTwoFactor ConsoleCommandKind = "disable-2fa"
)

type ConsoleCommand struct {
	Kind        ConsoleCommandKind
	Email       string
	NewPassword string
	// Password, Role and Name describe the account ConsoleCommandCreateUser creates.
	Password string
	Role     string
	Name     string
}

// RunConsoleCommand performs one console command and returns the text to print. It starts no
// routes and no background loops.
func (a *App) RunConsoleCommand(ctx context.Context, command ConsoleCommand) (string, error) {
	switch command.Kind {
	case ConsoleCommandListAdmins:
		return a.listAdmins(ctx)
	case ConsoleCommandCreateUser:
		return a.createUser(ctx, command)
	case ConsoleCommandResetPassword:
		return a.resetPassword(ctx, command)
	case ConsoleCommandDisableTwoFactor:
		return a.disableTwoFactor(ctx)
	default:
		return "", fmt.Errorf("%w: %q", ErrUnknownConsoleCommand, command.Kind)
	}
}

// The listing and the created account are printed rather than logged: the owner runs these to
// read the addresses, and the logger masks them as the PII they otherwise are.
func (a *App) listAdmins(ctx context.Context) (string, error) {
	listing, err := users_services.GetManagementService().RenderAdminListing(ctx)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to list administrators", slog.Any("error", err))

		return "", fmt.Errorf("list administrators: %w", err)
	}

	return listing, nil
}

func (a *App) createUser(ctx context.Context, command ConsoleCommand) (string, error) {
	outcome, err := users_services.GetUserService().CreateUserFromConsole(
		ctx,
		&users_dto.CreateConsoleAccountRequest{
			Email:    command.Email,
			Password: command.Password,
			Role:     command.Role,
			Name:     command.Name,
		},
	)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create the account", slog.Any("error", err))

		return "", fmt.Errorf("create the account: %w", err)
	}

	return users_services.RenderConsoleAccountOutcome(outcome), nil
}

func (a *App) resetPassword(ctx context.Context, command ConsoleCommand) (string, error) {
	a.logger.InfoContext(ctx, "resetting password")

	err := users_services.GetUserService().ChangeUserPasswordByEmail(ctx, command.Email, command.NewPassword)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to reset password", slog.Any("error", err))

		return "", fmt.Errorf("reset the password: %w", err)
	}

	a.logger.InfoContext(ctx, "password reset successfully")

	return "", nil
}

func (a *App) disableTwoFactor(ctx context.Context) (string, error) {
	isChanged, err := users_services.GetSettingsService().DisableTwoFactorAuth(ctx)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to switch the second factor off", slog.Any("error", err))

		return "", fmt.Errorf("switch the second factor off: %w", err)
	}

	if isChanged {
		return "Two-factor authentication is now off. The next password sign-in needs no code.", nil
	}

	return "Two-factor authentication was already off. Nothing changed.", nil
}
