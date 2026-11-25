package instancebackups_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

var mailedCodePattern = regexp.MustCompile(`>(\d{6})<`)

func TestCreateMasterKeyExport(t *testing.T) {
	t.Parallel()

	t.Run("downloads exactly the installed key once without a code when two-factor sign-in is off", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		installedKey := owner.App.ReadMasterKeyFile(t)

		issuedExport := requireMasterKeyExport(t, owner.RootAdmin, nil)
		firstDownload := downloadMasterKey(t, owner.App, issuedExport.Token)
		secondDownload := downloadMasterKey(t, owner.App, issuedExport.Token)

		require.Equal(t, http.StatusOK, firstDownload.StatusCode(), string(firstDownload.Body))
		assert.Equal(t, installedKey, firstDownload.Body)
		assert.Equal(t, installedKey, owner.App.ReadMasterKeyFile(t))
		assert.Equal(t, "no-store", firstDownload.HTTPResponse.Header.Get("Cache-Control"))
		assert.Equal(t, `attachment; filename=databasus-master-key.txt`,
			firstDownload.HTTPResponse.Header.Get("Content-Disposition"))
		requireProblemCode(t, secondDownload.StatusCode(), secondDownload.Body,
			http.StatusUnauthorized, apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("ignores a code when two-factor sign-in is off", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		unusedCode := "000000"

		issuedExport := requireMasterKeyExport(t, owner.RootAdmin, &unusedCode)
		download := downloadMasterKey(t, owner.App, issuedExport.Token)

		require.Equal(t, http.StatusOK, download.StatusCode(), string(download.Body))
		assert.Equal(t, owner.App.ReadMasterKeyFile(t), download.Body)
	})

	t.Run("returns 401 for a download token after its 60 seconds passed", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		issuedExport := requireMasterKeyExport(t, owner.RootAdmin, nil)

		owner.App.Clock.Advance(61 * time.Second)
		download := downloadMasterKey(t, owner.App, issuedExport.Token)

		requireProblemCode(t, download.StatusCode(), download.Body,
			http.StatusUnauthorized, apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("keeps the key out of other responses, audit events and logs", func(t *testing.T) {
		t.Parallel()
		logs := &capturedLogs{mutex: sync.Mutex{}, buffer: bytes.Buffer{}}
		owner := newInstanceOwner(t, apptest.WithLogSink(slog.NewTextHandler(logs, nil)))
		masterKey := strings.TrimSpace(string(owner.App.ReadMasterKeyFile(t)))

		issuedExport := requireMasterKeyExport(t, owner.RootAdmin, nil)
		download := downloadMasterKey(t, owner.App, issuedExport.Token)
		require.Equal(t, http.StatusOK, download.StatusCode())
		statusResponse, err := owner.RootAdmin.Client.GetInstanceBackupStatusWithResponse(t.Context())
		require.NoError(t, err)
		settingsResponse, err := owner.RootAdmin.Client.GetInstanceBackupSettingsWithResponse(t.Context())
		require.NoError(t, err)
		auditEvents := apptest.ListAllAuditEvents(t, owner.RootAdmin, apiclient.ListAuditEventsParams{
			WorkspaceId: nil, AuthorUserId: &owner.RootAdmin.UserID, Limit: nil, After: nil,
		})

		assert.NotContains(t, string(statusResponse.Body), masterKey)
		assert.NotContains(t, string(settingsResponse.Body), masterKey)
		require.Contains(t, logs.String(), "Root admin downloaded the master key")
		assert.NotContains(t, logs.String(), masterKey)
		assert.NotContains(t, logs.String(), issuedExport.Token)
		auditMessages := make([]string, 0, len(auditEvents))
		for _, auditEvent := range auditEvents {
			auditMessages = append(auditMessages, auditEvent.Message)
		}
		assert.Contains(t, auditMessages, "Root admin downloaded the master key")
		assert.NotContains(t, strings.Join(auditMessages, "\n"), masterKey)
	})

	t.Run("returns 403 when an instance admin who is not root admin exports", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		response := createMasterKeyExport(t, owner.OtherAdmin, nil)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusForbidden, apiclient.ProblemCodePermissionDenied)
	})

	t.Run("exports over HTTP without any acknowledgment of the risk", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		response, err := owner.RootAdmin.Client.CreateMasterKeyExportWithResponse(
			t.Context(), apiclient.MasterKeyExportCreate{Code: nil},
		)

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
	})

	t.Run("accepts an emailed code once and refuses it again when two-factor sign-in is on", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)
		code := requireMasterKeyCode(t, owner)

		firstExport := createMasterKeyExport(t, owner.RootAdmin, &code)
		secondExport := createMasterKeyExport(t, owner.RootAdmin, &code)

		require.Equal(t, http.StatusCreated, firstExport.StatusCode(), string(firstExport.Body))
		requireProblemCode(t, secondExport.StatusCode(), secondExport.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
	})

	t.Run("refuses an export without a code when two-factor sign-in is on", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)

		response := createMasterKeyExport(t, owner.RootAdmin, nil)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
	})

	t.Run("refuses a wrong code when two-factor sign-in is on", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)
		requireMasterKeyCode(t, owner)
		wrongCode := "000000"

		response := createMasterKeyExport(t, owner.RootAdmin, &wrongCode)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
	})

	t.Run("refuses the account password sent as a code when two-factor sign-in is on", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)
		password := apptest.TestPassword

		response := createMasterKeyExport(t, owner.RootAdmin, &password)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
	})

	t.Run("refuses a code after its five minutes passed when two-factor sign-in is on", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)
		code := requireMasterKeyCode(t, owner)

		owner.App.Clock.Advance(5*time.Minute + time.Second)
		response := createMasterKeyExport(t, owner.RootAdmin, &code)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
	})

	t.Run("refuses a code issued for a recovery when two-factor sign-in is on", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)
		prepared := owner.App.PrepareRecovery(t)
		recoveryCode := requireRecoveryCode(t, owner, prepared.ID)

		response := createMasterKeyExport(t, owner.RootAdmin, &recoveryCode)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
	})

	t.Run("refuses a code after root admin changed the email address", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)
		code := requireMasterKeyCode(t, owner)

		renamed, err := owner.RootAdmin.Client.UpdateUserWithResponse(t.Context(), owner.RootAdmin.UserID,
			apiclient.UserUpdate{Name: "Owner", Email: apptest.NewTestEmail("renamed")})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, renamed.StatusCode(), string(renamed.Body))
		response := createMasterKeyExport(t, owner.RootAdmin, &code)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
	})
	t.Run("exports with root admin's app code and sends no mail", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)
		connectedApp := owner.App.ConnectAuthenticatorApp(t, owner.RootAdmin)
		code := owner.App.GenerateAuthenticatorCode(t, connectedApp.Secret)

		issuedExport := requireMasterKeyExport(t, owner.RootAdmin, &code)
		replayedResponse := createMasterKeyExport(t, owner.RootAdmin, &code)

		download := downloadMasterKey(t, owner.App, issuedExport.Token)
		require.Equal(t, http.StatusOK, download.StatusCode(), string(download.Body))
		assert.Equal(t, owner.App.ReadMasterKeyFile(t), download.Body)
		assert.Empty(t, owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email))
		requireProblemCode(t, replayedResponse.StatusCode(), replayedResponse.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
	})

	t.Run("exports with a recovery code on an instance without a mail server and spends the code",
		func(t *testing.T) {
			t.Parallel()
			owner := newInstanceOwner(t, apptest.WithoutMailServer())
			connectedApp := owner.App.ConnectAuthenticatorApp(t, owner.RootAdmin)
			setTwoFactorRequirement(t, owner.RootAdmin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)

			issuedExport := requireMasterKeyExport(t, owner.RootAdmin, &connectedApp.RecoveryCodes[0])
			reusedResponse := createMasterKeyExport(t, owner.RootAdmin, &connectedApp.RecoveryCodes[0])

			download := downloadMasterKey(t, owner.App, issuedExport.Token)
			require.Equal(t, http.StatusOK, download.StatusCode(), string(download.Body))
			assert.Equal(t, owner.App.ReadMasterKeyFile(t), download.Body)
			requireProblemCode(t, reusedResponse.StatusCode(), reusedResponse.Body,
				http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
		})

	t.Run("refuses a wrong app code when root admin has an app", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)
		connectedApp := owner.App.ConnectAuthenticatorApp(t, owner.RootAdmin)
		wrongCode := getWrongCode(owner.App.GenerateAuthenticatorCode(t, connectedApp.Secret))

		response := createMasterKeyExport(t, owner.RootAdmin, &wrongCode)

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
	})
}

func TestRequestMasterKeyConfirmation(t *testing.T) {
	t.Parallel()

	t.Run("answers the app method without expiry and sends no mail when root admin has an app",
		func(t *testing.T) {
			t.Parallel()
			owner := newInstanceOwner(t, apptest.WithoutMailServer())
			owner.App.ConnectAuthenticatorApp(t, owner.RootAdmin)
			setTwoFactorRequirement(t, owner.RootAdmin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)

			response, err := owner.RootAdmin.Client.RequestMasterKeyConfirmationWithResponse(t.Context())

			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
			assert.Equal(t, apiclient.IdentityConfirmationMethodAUTHENTICATORAPP, response.JSON201.Method)
			assert.Nil(t, response.JSON201.ExpiresAt)
		})

	t.Run("returns 409 and exports without a code when nothing is required although root admin has an app",
		func(t *testing.T) {
			t.Parallel()
			owner := newInstanceOwner(t)
			owner.App.ConnectAuthenticatorApp(t, owner.RootAdmin)

			response, err := owner.RootAdmin.Client.RequestMasterKeyConfirmationWithResponse(t.Context())

			require.NoError(t, err)
			requireProblemCode(t, response.StatusCode(), response.Body,
				http.StatusConflict, apiclient.ProblemCodeIdentityProofUnavailable)
			requireMasterKeyExport(t, owner.RootAdmin, nil)
		})

	t.Run("returns 409 under the app method without a mail server after root admin's app was removed",
		func(t *testing.T) {
			t.Parallel()
			owner := newInstanceOwner(t, apptest.WithoutMailServer())
			connectedApp := owner.App.ConnectAuthenticatorApp(t, owner.RootAdmin)
			removalCode := owner.App.GenerateAuthenticatorCode(t, connectedApp.Secret)
			removal, err := owner.RootAdmin.Client.DisconnectAuthenticatorAppWithResponse(t.Context(),
				owner.RootAdmin.UserID, apiclient.SecondFactorProof{Code: &removalCode})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, removal.StatusCode(), string(removal.Body))
			setTwoFactorRequirement(t, owner.RootAdmin, apiclient.TwoFactorRequirementAUTHENTICATORAPP)

			response, err := owner.RootAdmin.Client.RequestMasterKeyConfirmationWithResponse(t.Context())

			require.NoError(t, err)
			requireProblemCode(t, response.StatusCode(), response.Body,
				http.StatusConflict, apiclient.ProblemCodeIdentityProofUnavailable)
		})

	t.Run("answers the email method with the expiry of the emailed code when root admin has no app",
		func(t *testing.T) {
			t.Parallel()
			owner := newTwoFactorInstanceOwner(t)

			response, err := owner.RootAdmin.Client.RequestMasterKeyConfirmationWithResponse(t.Context())

			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
			assert.Equal(t, apiclient.IdentityConfirmationMethodEMAIL, response.JSON201.Method)
			assert.NotNil(t, response.JSON201.ExpiresAt)
			assert.Len(t, owner.App.Mail.GetMessagesTo(owner.RootAdmin.Email), 1)
		})

	t.Run("returns 409 without a mail server", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t, apptest.WithoutMailServer())
		rootAdmin := app.CreateAdmin(t)
		requireTwoFactorAfterMailServerLoss(t, app)

		response, err := rootAdmin.Client.RequestMasterKeyConfirmationWithResponse(t.Context())

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusConflict, apiclient.ProblemCodeIdentityProofUnavailable)
	})

	t.Run("refuses a second code within a minute", func(t *testing.T) {
		t.Parallel()
		owner := newTwoFactorInstanceOwner(t)
		requireMasterKeyCode(t, owner)

		response, err := owner.RootAdmin.Client.RequestMasterKeyConfirmationWithResponse(t.Context())

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusTooManyRequests, apiclient.ProblemCodeRateLimited)
	})
}

func TestMasterKeyAcknowledgment(t *testing.T) {
	t.Parallel()

	t.Run("marks the current key saved for every session, while a download alone does not", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		issuedExport := requireMasterKeyExport(t, owner.RootAdmin, nil)
		download := downloadMasterKey(t, owner.App, issuedExport.Token)
		require.Equal(t, http.StatusOK, download.StatusCode())

		isAcknowledgedAfterDownload := getStatus(t, owner.RootAdmin).IsMasterKeyAcknowledged
		acknowledgment, err := owner.RootAdmin.Client.AcknowledgeMasterKeyWithResponse(
			t.Context(), apiclient.MasterKeyAcknowledgment{IsSaved: true},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, acknowledgment.StatusCode(), string(acknowledgment.Body))
		otherSession := signInAgain(t, owner.App, owner.RootAdmin.Email)

		assert.False(t, isAcknowledgedAfterDownload)
		assert.True(t, getStatus(t, otherSession).IsMasterKeyAcknowledged)
	})

	t.Run("returns 403 when an instance admin who is not root admin acknowledges", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		response, err := owner.OtherAdmin.Client.AcknowledgeMasterKeyWithResponse(
			t.Context(), apiclient.MasterKeyAcknowledgment{IsSaved: true},
		)

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusForbidden, apiclient.ProblemCodePermissionDenied)
		assert.False(t, getStatus(t, owner.RootAdmin).IsMasterKeyAcknowledged)
	})

	t.Run("asks again when the installed key differs from the saved one", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		acknowledgment, err := owner.RootAdmin.Client.AcknowledgeMasterKeyWithResponse(
			t.Context(), apiclient.MasterKeyAcknowledgment{IsSaved: true},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, acknowledgment.StatusCode(), string(acknowledgment.Body))

		// A recovery installs another key; the API cannot do that, so the stored digest is replaced.
		owner.App.ExecuteSQL(t, "UPDATE instancebackups_configs SET acknowledged_master_key_sha256 = ?",
			strings.Repeat("cd", 32))

		assert.False(t, getStatus(t, owner.RootAdmin).IsMasterKeyAcknowledged)
	})
}

func TestMasterKeyExportCodeRequirement(t *testing.T) {
	t.Parallel()

	t.Run("reports a code required only after the instance requires two-factor sign-in", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		isRequiredBefore := getStatus(t, owner.RootAdmin).IsConfirmationCodeRequired

		requireTwoFactorSignIn(t, owner.RootAdmin)

		assert.False(t, isRequiredBefore)
		assert.True(t, getStatus(t, owner.RootAdmin).IsConfirmationCodeRequired)
	})
}

func newTwoFactorInstanceOwner(t *testing.T) instanceOwner {
	t.Helper()

	owner := newInstanceOwner(t)
	requireTwoFactorSignIn(t, owner.RootAdmin)

	return owner
}

// requireTwoFactorAfterMailServerLoss turns two-factor sign-in on directly, as on an instance that
// lost its mail server after root admin turned it on; the API refuses to turn it on without one.
func requireTwoFactorAfterMailServerLoss(t *testing.T, app *apptest.App) {
	t.Helper()

	app.ExecuteSQL(t, "UPDATE users_settings SET two_factor_requirement = 'EMAIL'")
}

func requireTwoFactorSignIn(t *testing.T, rootAdmin apptest.Session) {
	t.Helper()

	setTwoFactorRequirement(t, rootAdmin, apiclient.TwoFactorRequirementEMAIL)
}

func setTwoFactorRequirement(
	t *testing.T,
	rootAdmin apptest.Session,
	requirement apiclient.TwoFactorRequirement,
) {
	t.Helper()

	currentSettings, err := rootAdmin.Client.GetInstanceSettingsWithResponse(t.Context())
	require.NoError(t, err)
	require.NotNil(t, currentSettings.JSON200, string(currentSettings.Body))

	response, err := rootAdmin.Client.UpdateInstanceSettingsWithResponse(t.Context(), apiclient.InstanceSettingsUpdate{
		IsAllowExternalRegistrations:      currentSettings.JSON200.IsAllowExternalRegistrations,
		IsAllowMemberInvitations:          currentSettings.JSON200.IsAllowMemberInvitations,
		IsMemberAllowedToCreateWorkspaces: currentSettings.JSON200.IsMemberAllowedToCreateWorkspaces,
		TwoFactorRequirement:              requirement,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
}

func createMasterKeyExport(
	t *testing.T,
	session apptest.Session,
	code *string,
) *apiclient.CreateMasterKeyExportResponse {
	t.Helper()

	response, err := session.Client.CreateMasterKeyExportWithResponse(t.Context(), apiclient.MasterKeyExportCreate{
		Code: code,
	})
	require.NoError(t, err)

	return response
}

func requireMasterKeyExport(t *testing.T, session apptest.Session, code *string) apiclient.MasterKeyExport {
	t.Helper()

	response := createMasterKeyExport(t, session, code)
	require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))

	return *response.JSON201
}

func downloadMasterKey(t *testing.T, app *apptest.App, token string) *apiclient.DownloadMasterKeyResponse {
	t.Helper()

	response, err := app.NewClient(t).DownloadMasterKeyWithResponse(
		t.Context(), &apiclient.DownloadMasterKeyParams{Token: token},
	)
	require.NoError(t, err)

	return response
}

func requireMasterKeyCode(t *testing.T, owner instanceOwner) string {
	t.Helper()

	return requireMailedCode(t, owner.App, owner.RootAdmin)
}

func requireMailedCode(t *testing.T, app *apptest.App, session apptest.Session) string {
	t.Helper()

	response, err := session.Client.RequestMasterKeyConfirmationWithResponse(t.Context())
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))

	return readMailedCode(t, app, session.Email)
}

// readMailedCode reads the code of the newest message sent to the address.
func readMailedCode(t *testing.T, app *apptest.App, email string) string {
	t.Helper()

	messages := app.Mail.GetMessagesTo(email)
	require.NotEmpty(t, messages)
	match := mailedCodePattern.FindStringSubmatch(messages[len(messages)-1].HTMLBody)
	require.Len(t, match, 2)

	return match[1]
}

func signInAgain(t *testing.T, app *apptest.App, email string) apptest.Session {
	t.Helper()

	response, err := app.NewClient(t).SignInWithResponse(t.Context(), apiclient.SignInRequest{
		Email:    email,
		Password: new(apptest.TestPassword),
	})
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))

	completed, err := response.JSON200.AsSignInCompleted()
	require.NoError(t, err)

	return apptest.Session{
		Client: app.NewSessionClient(t, completed.Token),
		UserID: completed.UserId,
		Email:  email,
		Token:  completed.Token,
	}
}

func getStatus(t *testing.T, session apptest.Session) apiclient.InstanceBackupStatus {
	t.Helper()

	response, err := session.Client.GetInstanceBackupStatusWithResponse(t.Context())
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))

	return *response.JSON200
}

// capturedLogs collects the application's log records, which several goroutines write.
type capturedLogs struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (c *capturedLogs) Write(record []byte) (int, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.buffer.Write(record)
}

func (c *capturedLogs) String() string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.buffer.String()
}
