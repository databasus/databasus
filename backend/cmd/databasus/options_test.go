package main

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app"
)

func TestSelectCommand(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                string
		arguments           []string
		expectedKind        commandKind
		expectedConsoleKind app.ConsoleCommandKind
	}{
		{name: "serves without command flags", arguments: nil, expectedKind: commandServe},
		{
			name:         "probes the storage with --test-storage",
			arguments:    []string{"--test-storage"},
			expectedKind: commandProbeStorage,
		},
		{
			name:                "lists administrators with --list-admins",
			arguments:           []string{"--list-admins"},
			expectedKind:        commandConsole,
			expectedConsoleKind: app.ConsoleCommandListAdmins,
		},
		{
			name: "creates an account with --create-user and --email",
			arguments: []string{
				"--create-user",
				"--email=owner@example.com",
				"--password=secret",
				"--role=admin",
			},
			expectedKind:        commandConsole,
			expectedConsoleKind: app.ConsoleCommandCreateUser,
		},
		{
			name:                "resets a password with --new-password and --email",
			arguments:           []string{"--new-password=secret", "--email=owner@example.com"},
			expectedKind:        commandConsole,
			expectedConsoleKind: app.ConsoleCommandResetPassword,
		},
		{
			name:                "disables the second factor with --disable-2fa",
			arguments:           []string{"--disable-2fa"},
			expectedKind:        commandConsole,
			expectedConsoleKind: app.ConsoleCommandDisableTwoFactor,
		},
		{
			name:         "probes the storage first when several command flags are given",
			arguments:    []string{"--disable-2fa", "--list-admins", "--test-storage"},
			expectedKind: commandProbeStorage,
		},
		{
			name:                "lists administrators before creating an account when both flags are given",
			arguments:           []string{"--create-user", "--email=owner@example.com", "--list-admins"},
			expectedKind:        commandConsole,
			expectedConsoleKind: app.ConsoleCommandListAdmins,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			parsedOptions, err := parseOptions(testCase.arguments, io.Discard)
			require.NoError(t, err)

			command, err := selectCommand(parsedOptions)

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedKind, command.kind)
			assert.Equal(t, testCase.expectedConsoleKind, command.console.Kind)
		})
	}

	refusedCases := []struct {
		name          string
		arguments     []string
		expectedError error
	}{
		{
			name:          "refuses --create-user with --new-password before --new-password resets anything",
			arguments:     []string{"--create-user", "--email=owner@example.com", "--new-password=secret"},
			expectedError: ErrCreateUserTakesPassword,
		},
		{
			name:          "refuses --create-user without --email",
			arguments:     []string{"--create-user", "--password=secret", "--role=admin"},
			expectedError: ErrEmailMissing,
		},
		{
			name:          "refuses --new-password without --email",
			arguments:     []string{"--new-password=secret"},
			expectedError: ErrEmailMissing,
		},
	}

	for _, refusedCase := range refusedCases {
		t.Run(refusedCase.name, func(t *testing.T) {
			t.Parallel()

			parsedOptions, err := parseOptions(refusedCase.arguments, io.Discard)
			require.NoError(t, err)

			_, err = selectCommand(parsedOptions)

			require.ErrorIs(t, err, refusedCase.expectedError)
		})
	}
}
