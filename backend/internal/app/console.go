package app

import (
	"context"
	"fmt"
	"log/slog"

	"databasus-backend/internal/modules/users"
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
	listing, err := a.usersModule.RenderAdminListing(ctx)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to list administrators", slog.Any("error", err))

		return "", fmt.Errorf("list administrators: %w", err)
	}

	return listing, nil
}

func (a *App) createUser(ctx context.Context, command ConsoleCommand) (string, error) {
	outcome, err := a.usersModule.CreateUserFromConsole(ctx, users.ConsoleUserRequest{
		Email:    command.Email,
		Password: command.Password,
		Role:     command.Role,
		Name:     command.Name,
	})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create the account", slog.Any("error", err))

		return "", fmt.Errorf("create the account: %w", err)
	}

	return renderConsoleUserOutcome(outcome), nil
}

func (a *App) resetPassword(ctx context.Context, command ConsoleCommand) (string, error) {
	a.logger.InfoContext(ctx, "resetting password")

	err := a.usersModule.ChangePasswordFromConsole(ctx, users.ConsolePasswordChange{
		Email:    command.Email,
		Password: command.NewPassword,
	})
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to reset password", slog.Any("error", err))

		return "", fmt.Errorf("reset the password: %w", err)
	}

	a.logger.InfoContext(ctx, "password reset successfully")

	return "", nil
}

func (a *App) disableTwoFactor(ctx context.Context) (string, error) {
	isChanged, err := a.usersModule.DisableTwoFactorAuth(ctx)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to switch the second factor off", slog.Any("error", err))

		return "", fmt.Errorf("switch the second factor off: %w", err)
	}

	if isChanged {
		return "Two-factor authentication is now off. The next password sign-in needs no code.", nil
	}

	return "Two-factor authentication was already off. Nothing changed.", nil
}

// renderConsoleUserOutcome is the line the operator reads. A rerun says the password was not
// changed, because that is how a deployment learns a rerun did not rotate its secret.
func renderConsoleUserOutcome(outcome users.ConsoleUserOutcome) string {
	if !outcome.IsCreated {
		return fmt.Sprintf(
			"Account %s already exists with the %s role. Nothing changed, and its password was not changed.",
			outcome.User.Email,
			outcome.User.Role,
		)
	}

	line := fmt.Sprintf("Created account %s with the %s role.", outcome.User.Email, outcome.User.Role)
	if outcome.IsBootstrapAdmin {
		line += " It is the administrator of this instance."
	}

	return line
}
