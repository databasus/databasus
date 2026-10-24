package main

import (
	"flag"
	"io"

	"databasus-backend/internal/app"
)

const storageProbeFlagName = "test-storage"

type options struct {
	shouldTestStorage      bool
	shouldListAdmins       bool
	shouldDisableTwoFactor bool
	shouldCreateUser       bool
	newPassword            string
	email                  string
	newUserPassword        string
	newUserRole            string
	newUserName            string
}

type commandKind string

const (
	commandServe        commandKind = "serve"
	commandProbeStorage commandKind = "probe-storage"
	commandConsole      commandKind = "console"
)

type selectedCommand struct {
	kind    commandKind
	console app.ConsoleCommand
}

func parseOptions(arguments []string, errorOutput io.Writer) (options, error) {
	flags := flag.NewFlagSet("databasus", flag.ContinueOnError)
	flags.SetOutput(errorOutput)

	var parsedOptions options

	flags.BoolVar(
		&parsedOptions.shouldTestStorage,
		storageProbeFlagName,
		false,
		"Save and delete a local storage probe",
	)
	flags.BoolVar(&parsedOptions.shouldListAdmins, "list-admins", false,
		"Print every administrator account of this instance")
	// The name carries a digit, unlike every other flag here, because it is the term an owner
	// locked out by a dead mail server searches for.
	flags.BoolVar(&parsedOptions.shouldDisableTwoFactor, "disable-2fa", false,
		"Stop requiring an emailed code at sign-in")
	flags.BoolVar(&parsedOptions.shouldCreateUser, "create-user", false,
		"Create an account with the given email, password and role")
	flags.StringVar(&parsedOptions.newPassword, "new-password", "", "Set a new password for the user")
	flags.StringVar(&parsedOptions.email, "email", "", "Email of the user to create or to reset password")
	flags.StringVar(&parsedOptions.newUserPassword, "password", "", "Password of the account --create-user creates")
	flags.StringVar(
		&parsedOptions.newUserRole,
		"role",
		"",
		"Role of the account --create-user creates: admin or member",
	)
	flags.StringVar(&parsedOptions.newUserName, "name", "", "Display name of the account --create-user creates")

	err := flags.Parse(arguments)

	return parsedOptions, err
}

// selectCommand picks the first command flag in a fixed order and refuses a command that cannot run
// before any setting is read or the database is touched. --create-user is checked before
// --new-password, so a --new-password passed by mistake is refused instead of resetting some
// account's password.
func selectCommand(parsedOptions options) (selectedCommand, error) {
	switch {
	case parsedOptions.shouldTestStorage:
		return selectedCommand{kind: commandProbeStorage, console: app.ConsoleCommand{}}, nil
	case parsedOptions.shouldListAdmins:
		return newConsoleCommand(app.ConsoleCommand{Kind: app.ConsoleCommandListAdmins}), nil
	case parsedOptions.shouldCreateUser:
		if parsedOptions.newPassword != "" {
			return selectedCommand{}, ErrCreateUserTakesPassword
		}

		if parsedOptions.email == "" {
			return selectedCommand{}, ErrEmailMissing
		}

		return newConsoleCommand(app.ConsoleCommand{
			Kind:     app.ConsoleCommandCreateUser,
			Email:    parsedOptions.email,
			Password: parsedOptions.newUserPassword,
			Role:     parsedOptions.newUserRole,
			Name:     parsedOptions.newUserName,
		}), nil
	case parsedOptions.newPassword != "":
		if parsedOptions.email == "" {
			return selectedCommand{}, ErrEmailMissing
		}

		return newConsoleCommand(app.ConsoleCommand{
			Kind:        app.ConsoleCommandResetPassword,
			Email:       parsedOptions.email,
			NewPassword: parsedOptions.newPassword,
		}), nil
	case parsedOptions.shouldDisableTwoFactor:
		return newConsoleCommand(app.ConsoleCommand{Kind: app.ConsoleCommandDisableTwoFactor}), nil
	default:
		return selectedCommand{kind: commandServe, console: app.ConsoleCommand{}}, nil
	}
}

func newConsoleCommand(command app.ConsoleCommand) selectedCommand {
	return selectedCommand{kind: commandConsole, console: command}
}
