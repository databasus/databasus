package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	old_config "databasus-backend/internal/config"
	users_dto "databasus-backend/internal/features/users/dto"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
)

const consolePassword = "console-password-123"

func TestRunConsoleCommand(t *testing.T) {
	t.Run("creates an administrator that signs in and records it in the audit log", func(t *testing.T) {
		testApp := newTestApp(t)
		email := newConsoleEmail()

		output, err := testApp.RunConsoleCommand(t.Context(), ConsoleCommand{
			Kind: ConsoleCommandCreateUser, Email: email, Password: consolePassword, Role: "admin",
		})

		require.NoError(t, err)
		assert.Contains(t, output, email)
		require.NoError(t, signIn(t, email, consolePassword))
		assert.True(
			t,
			hasAuditRecordContaining(
				t,
				"Account created from the host console: "+email+" (role "+string(users.RoleAdmin)+")",
			),
		)
		assert.Nil(t, testApp.backgroundGroup)
	})

	t.Run("resets the password so only the new one signs in and records the change", func(t *testing.T) {
		testApp := newTestApp(t)
		email := createConsoleAdministrator(t, testApp)

		_, err := testApp.RunConsoleCommand(t.Context(), ConsoleCommand{
			Kind: ConsoleCommandResetPassword, Email: email, NewPassword: "rotated-password-456",
		})

		require.NoError(t, err)
		require.NoError(t, signIn(t, email, "rotated-password-456"))
		require.Error(t, signIn(t, email, consolePassword))
		assert.Equal(t, int64(1), countPasswordChangeRecords(t, email))
		assert.Nil(t, testApp.backgroundGroup)
	})

	t.Run("returns an error and changes no account when the address is unknown", func(t *testing.T) {
		testApp := newTestApp(t)
		email := createConsoleAdministrator(t, testApp)

		_, err := testApp.RunConsoleCommand(t.Context(), ConsoleCommand{
			Kind: ConsoleCommandResetPassword, Email: newConsoleEmail(), NewPassword: "rotated-password-456",
		})

		require.Error(t, err)
		require.NoError(t, signIn(t, email, consolePassword))
	})

	t.Run("switches the second factor off and records it in the audit log", func(t *testing.T) {
		testApp := newTestApp(t)
		users_testing.EnableTwoFactorAuth(t.Context())

		output, err := testApp.RunConsoleCommand(t.Context(), ConsoleCommand{Kind: ConsoleCommandDisableTwoFactor})

		require.NoError(t, err)
		assert.Contains(t, output, "Two-factor authentication is now off")

		settings, err := users_services.GetSettingsService().GetSettings(t.Context())
		require.NoError(t, err)
		assert.False(t, settings.IsTwoFactorAuthRequired)
		assert.True(t, hasAuditRecordContaining(t, "isTwoFactorAuthRequired: true -> false"))
		assert.Nil(t, testApp.backgroundGroup)
	})

	t.Run("lists the administrators without masking their addresses", func(t *testing.T) {
		testApp := newTestApp(t)
		email := createConsoleAdministrator(t, testApp)

		output, err := testApp.RunConsoleCommand(t.Context(), ConsoleCommand{Kind: ConsoleCommandListAdmins})

		require.NoError(t, err)
		assert.Contains(t, output, email)
		assert.Nil(t, testApp.backgroundGroup)
	})
}

func TestProbeStorage(t *testing.T) {
	t.Run("saves and deletes the probe without a connection string", func(t *testing.T) {
		t.Setenv("ENV_MODE", "production")
		t.Setenv("DATABASE_DSN", "")

		environment, err := LoadEnvironment(t.Context(), newIsolatedLoadSpec(t, false))
		require.NoError(t, err)
		require.NoError(t, os.MkdirAll(environment.Config.TempFolder, 0o755))
		require.NoError(t, os.MkdirAll(environment.Config.DataFolder, 0o755))

		require.NoError(t, ProbeStorage(t.Context(), environment))

		leftProbes, err := filepath.Glob(filepath.Join(environment.Config.DataFolder, ".databasus-storage-test-*"))
		require.NoError(t, err)
		assert.Empty(t, leftProbes)
	})

	t.Run("names both paths when the data directory cannot be written", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root ignores directory permissions")
		}

		t.Setenv("ENV_MODE", "production")
		t.Setenv("DATABASE_DSN", "")

		environment, err := LoadEnvironment(t.Context(), newIsolatedLoadSpec(t, false))
		require.NoError(t, err)
		require.NoError(t, os.MkdirAll(environment.Config.TempFolder, 0o755))
		require.NoError(t, os.MkdirAll(environment.Config.DataFolder, 0o500))

		err = ProbeStorage(t.Context(), environment)

		var probeError *StorageProbeError
		require.ErrorAs(t, err, &probeError)
		assert.Contains(t, err.Error(), environment.Config.DataFolder)
		assert.Contains(t, err.Error(), environment.Config.TempFolder)
		assert.Contains(t, err.Error(), "save a file through local storage")
		assert.Contains(t, err.Error(), storagePermissionsDocumentationURL)

		leftTemporaryProbes, err := filepath.Glob(
			filepath.Join(environment.Config.TempFolder, ".databasus-storage-test-*"),
		)
		require.NoError(t, err)
		assert.Empty(t, leftTemporaryProbes)
	})

	t.Run("names the delete operation when the probe cannot be deleted", func(t *testing.T) {
		t.Setenv("ENV_MODE", "production")
		t.Setenv("DATABASE_DSN", "")

		environment, err := LoadEnvironment(t.Context(), newIsolatedLoadSpec(t, false))
		require.NoError(t, err)
		old_config.Install(environment.Config)

		err = probeStorage(t.Context(), environment, &undeletableStorage{})

		var probeError *StorageProbeError
		require.ErrorAs(t, err, &probeError)
		assert.Contains(t, err.Error(), "delete a file through local storage")
		assert.Contains(t, err.Error(), storagePermissionsDocumentationURL)
	})
}

// undeletableStorage stands in for a volume that refuses deletes, which a test cannot cause on a
// real directory without also refusing the save before it.
type undeletableStorage struct{}

func (*undeletableStorage) SaveFile(
	context.Context, encryption.FieldEncryptor, *slog.Logger, string, io.Reader,
) error {
	return nil
}

func (*undeletableStorage) DeleteFile(context.Context, encryption.FieldEncryptor, *slog.Logger, string) error {
	return errors.New("operation not permitted")
}

func createConsoleAdministrator(t *testing.T, testApp *App) string {
	t.Helper()

	email := newConsoleEmail()

	_, err := testApp.RunConsoleCommand(t.Context(), ConsoleCommand{
		Kind: ConsoleCommandCreateUser, Email: email, Password: consolePassword, Role: "admin",
	})
	require.NoError(t, err)

	return email
}

func newConsoleEmail() string {
	return "console-" + strings.ReplaceAll(uuid.NewString(), "-", "") + "@databasus.local"
}

func signIn(t *testing.T, email, password string) error {
	t.Helper()

	_, err := users_services.GetUserService().SignIn(
		t.Context(),
		&users_dto.SignInRequestDTO{Email: email, Password: password},
	)

	return err
}

func hasAuditRecordContaining(t *testing.T, messagePart string) bool {
	t.Helper()

	var isPresent bool
	require.NoError(t, storage.GetDb().WithContext(t.Context()).Raw(
		"SELECT EXISTS (SELECT 1 FROM lib_audit_events WHERE strpos(message, ?) > 0)", messagePart,
	).Scan(&isPresent).Error)

	return isPresent
}

func countPasswordChangeRecords(t *testing.T, email string) int64 {
	t.Helper()

	var recordCount int64
	require.NoError(t, storage.GetDb().WithContext(t.Context()).Raw(
		`SELECT COUNT(*) FROM lib_audit_events
		 WHERE message = 'Password changed' AND author_email = ?`,
		email,
	).Scan(&recordCount).Error)

	return recordCount
}
