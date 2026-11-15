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

	t.Run("downloads exactly the installed key once after root admin confirms the password", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		issuedExport := requireMasterKeyExport(t, owner.RootAdmin, passwordProof(apptest.TestPassword))
		firstDownload, err := owner.App.NewClient(t).DownloadMasterKeyWithResponse(
			t.Context(), &apiclient.DownloadMasterKeyParams{Token: issuedExport.Token},
		)
		require.NoError(t, err)
		secondDownload, err := owner.App.NewClient(t).DownloadMasterKeyWithResponse(
			t.Context(), &apiclient.DownloadMasterKeyParams{Token: issuedExport.Token},
		)
		require.NoError(t, err)

		require.Equal(t, http.StatusOK, firstDownload.StatusCode(), string(firstDownload.Body))
		assert.Equal(t, owner.App.ReadMasterKeyFile(t), firstDownload.Body)
		assert.Equal(t, "no-store", firstDownload.HTTPResponse.Header.Get("Cache-Control"))
		assert.Contains(t, firstDownload.HTTPResponse.Header.Get("Content-Disposition"), "attachment")
		requireProblemCode(t, secondDownload.StatusCode(), secondDownload.Body,
			http.StatusUnauthorized, apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("returns 401 for a download token after its 60 seconds passed", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		issuedExport := requireMasterKeyExport(t, owner.RootAdmin, passwordProof(apptest.TestPassword))

		owner.App.Clock.Advance(61 * time.Second)
		download, err := owner.App.NewClient(t).DownloadMasterKeyWithResponse(
			t.Context(), &apiclient.DownloadMasterKeyParams{Token: issuedExport.Token},
		)

		require.NoError(t, err)
		requireProblemCode(t, download.StatusCode(), download.Body,
			http.StatusUnauthorized, apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("keeps the key out of other responses, audit events and logs", func(t *testing.T) {
		t.Parallel()
		logs := &capturedLogs{mutex: sync.Mutex{}, buffer: bytes.Buffer{}}
		owner := newInstanceOwner(t, apptest.WithLogSink(slog.NewTextHandler(logs, nil)))
		masterKey := strings.TrimSpace(string(owner.App.ReadMasterKeyFile(t)))

		issuedExport := requireMasterKeyExport(t, owner.RootAdmin, passwordProof(apptest.TestPassword))
		download, err := owner.App.NewClient(t).DownloadMasterKeyWithResponse(
			t.Context(), &apiclient.DownloadMasterKeyParams{Token: issuedExport.Token},
		)
		require.NoError(t, err)
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

	t.Run("refuses a wrong password without issuing a token", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		response := createMasterKeyExport(t, owner.RootAdmin, passwordProof("wrong-password-123"))

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
	})

	t.Run("returns 403 when an instance admin who is not root admin exports", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		response := createMasterKeyExport(t, owner.OtherAdmin, passwordProof(apptest.TestPassword))

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusForbidden, apiclient.ProblemCodePermissionDenied)
	})

	t.Run("refuses an export over HTTP until the risk is acknowledged", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		response, err := owner.RootAdmin.Client.CreateMasterKeyExportWithResponse(
			t.Context(),
			apiclient.MasterKeyExportCreate{Proof: passwordProof(apptest.TestPassword)},
		)

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest)
	})

	t.Run("needs no acknowledgment when the public address uses HTTPS", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t, apptest.WithPublicURL("https://backup.example.com"))
		rootAdmin := app.CreateAdmin(t)

		response, err := rootAdmin.Client.CreateMasterKeyExportWithResponse(
			t.Context(),
			apiclient.MasterKeyExportCreate{Proof: passwordProof(apptest.TestPassword)},
		)

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
	})

	t.Run("accepts an emailed code once and refuses it again", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		code := requireMasterKeyCode(t, owner)

		firstExport := createMasterKeyExport(t, owner.RootAdmin, codeProof(code))
		secondExport := createMasterKeyExport(t, owner.RootAdmin, codeProof(code))

		require.Equal(t, http.StatusCreated, firstExport.StatusCode(), string(firstExport.Body))
		requireProblemCode(t, secondExport.StatusCode(), secondExport.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
	})

	t.Run("refuses a code after its five minutes passed", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		code := requireMasterKeyCode(t, owner)

		owner.App.Clock.Advance(5*time.Minute + time.Second)
		response := createMasterKeyExport(t, owner.RootAdmin, codeProof(code))

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
	})

	t.Run("refuses a code from another session of root admin", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		code := requireMasterKeyCode(t, owner)
		owner.App.Clock.Advance(time.Second)
		otherSession := signInAgain(t, owner.App, owner.RootAdmin.Email)

		response := createMasterKeyExport(t, otherSession, codeProof(code))

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
	})

	t.Run("refuses a code after root admin changed the email address", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		code := requireMasterKeyCode(t, owner)

		renamed, err := owner.RootAdmin.Client.UpdateUserWithResponse(t.Context(), owner.RootAdmin.UserID,
			apiclient.UserUpdate{Name: "Owner", Email: apptest.NewTestEmail("renamed")})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, renamed.StatusCode(), string(renamed.Body))
		response := createMasterKeyExport(t, owner.RootAdmin, codeProof(code))

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
	})
}

func TestRequestMasterKeyConfirmation(t *testing.T) {
	t.Parallel()

	t.Run("returns 409 without a mail server", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t, apptest.WithoutMailServer())
		rootAdmin := app.CreateAdmin(t)

		response, err := rootAdmin.Client.RequestMasterKeyConfirmationWithResponse(t.Context())

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusConflict, apiclient.ProblemCodeIdentityProofUnavailable)
	})

	t.Run("refuses a second code within a minute", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		requireMasterKeyCode(t, owner)

		response, err := owner.RootAdmin.Client.RequestMasterKeyConfirmationWithResponse(t.Context())

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusTooManyRequests, apiclient.ProblemCodeRateLimited)
	})

	t.Run("lets an account without a password confirm through an emailed code", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t, apptest.WithGitHub())
		email := apptest.NewTestEmail("owner")
		signIn, err := app.NewClient(t).SignInWithGitHubWithResponse(t.Context(), apiclient.OAuthSignInRequest{
			Code:        app.GitHub.Authorize(apptest.NewGitHubAccount(email)),
			RedirectUri: "http://localhost/auth/callback",
		})
		require.NoError(t, err)
		require.NotNil(t, signIn.JSON200, string(signIn.Body))
		rootAdmin := apptest.Session{
			Client: app.NewSessionClient(t, signIn.JSON200.Token),
			UserID: signIn.JSON200.UserId,
			Email:  email,
			Token:  signIn.JSON200.Token,
		}

		passwordExport := createMasterKeyExport(t, rootAdmin, passwordProof(apptest.TestPassword))
		code := requireMailedCode(t, app, rootAdmin)
		codeExport := createMasterKeyExport(t, rootAdmin, codeProof(code))

		requireProblemCode(t, passwordExport.StatusCode(), passwordExport.Body,
			http.StatusBadRequest, apiclient.ProblemCodeIdentityProofInvalid)
		require.Equal(t, http.StatusCreated, codeExport.StatusCode(), string(codeExport.Body))
	})
}

func TestAcknowledgeMasterKey(t *testing.T) {
	t.Parallel()

	t.Run("marks the current key saved for every session, while a download alone does not", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		issuedExport := requireMasterKeyExport(t, owner.RootAdmin, passwordProof(apptest.TestPassword))
		download, err := owner.App.NewClient(t).DownloadMasterKeyWithResponse(
			t.Context(), &apiclient.DownloadMasterKeyParams{Token: issuedExport.Token},
		)
		require.NoError(t, err)
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
	})
}

func passwordProof(password string) apiclient.IdentityProof {
	return apiclient.IdentityProof{Password: &password, Code: nil}
}

func codeProof(code string) apiclient.IdentityProof {
	return apiclient.IdentityProof{Password: nil, Code: &code}
}

func createMasterKeyExport(
	t *testing.T,
	session apptest.Session,
	proof apiclient.IdentityProof,
) *apiclient.CreateMasterKeyExportResponse {
	t.Helper()

	response, err := session.Client.CreateMasterKeyExportWithResponse(t.Context(), apiclient.MasterKeyExportCreate{
		Proof:                          proof,
		IsInsecureTransferAcknowledged: new(true),
	})
	require.NoError(t, err)

	return response
}

func requireMasterKeyExport(
	t *testing.T,
	session apptest.Session,
	proof apiclient.IdentityProof,
) apiclient.MasterKeyExport {
	t.Helper()

	response := createMasterKeyExport(t, session, proof)
	require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))

	return *response.JSON201
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
