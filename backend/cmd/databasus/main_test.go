package main

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app"
	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/testdb"
	"databasus-backend/internal/lib/testwait"
)

const targetRootPassword = "target-root-password-123"

//nolint:paralleltest // Each scenario sets the process environment the server loads its settings from.
func TestServeUntilStopped(t *testing.T) {
	t.Run("publishes an accepted recovery and serves the recovered instance paused without a restart",
		func(t *testing.T) {
			source := newRecoverySource(t)
			target := startRebuildingServer(t, testdb.Open(t))
			targetRoot := target.signUp(t)
			recoveryID := target.prepareRecovery(t, targetRoot, source)

			applyResponse := applyTargetRecovery(t, targetRoot, recoveryID)
			restoredRoot := target.waitForSignIn(t, source.rootEmail)

			require.Equal(t, http.StatusAccepted, applyResponse.StatusCode(), string(applyResponse.Body))
			assert.Equal(t, http.StatusUnauthorized, getStatusCode(t, targetRoot))
			status := getRecoveryStatus(t, restoredRoot)
			assert.True(t, status.IsRecoveryPaused)
			require.NotNil(t, status.Recovery)
			assert.Equal(t, recoveryID, status.Recovery.Id)
			assert.Equal(t, apiclient.InstanceRecoveryStatusPAUSED, status.Recovery.Status)
			assert.Equal(t, source.masterKey, target.readMasterKey(t))
			target.requireServing(t)

			target.restart(t)
			restartedRoot := target.waitForSignIn(t, source.rootEmail)
			assert.True(t, getRecoveryStatus(t, restartedRoot).IsRecoveryPaused)
		})

	t.Run("ends the accepted recovery as QUIESCENCE_FAILED and keeps serving while a console command runs",
		func(t *testing.T) {
			source := newRecoverySource(t)
			target := startRebuildingServer(t, testdb.Open(t))
			targetRoot := target.signUp(t)
			targetKey := target.readMasterKey(t)
			recoveryID := target.prepareRecovery(t, targetRoot, source)
			consoleLock, err := app.OpenConsoleLifecycle(target.environment)
			require.NoError(t, err)
			t.Cleanup(func() { _ = consoleLock.Close() })

			applyResponse := applyTargetRecovery(t, targetRoot, recoveryID)
			failedRecovery := waitForRecoveryStatus(t, targetRoot, recoveryID, apiclient.InstanceRecoveryStatusFAILED)

			require.Equal(t, http.StatusAccepted, applyResponse.StatusCode(), string(applyResponse.Body))
			require.NotNil(t, failedRecovery.Failure)
			assert.Equal(t, apiclient.InstanceRecoveryFailureQUIESCENCEFAILED, *failedRecovery.Failure)
			assert.False(t, getRecoveryStatus(t, targetRoot).IsRecoveryPaused)
			assert.Equal(t, targetKey, target.readMasterKey(t))
			target.requireServing(t)
		})
}

// recoverySource is a Databasus backup file made through the API of another instance, and that
// instance's master key, kept apart from it as the owner keeps them.
type recoverySource struct {
	archive   []byte
	masterKey []byte
	rootEmail string
}

func newRecoverySource(t *testing.T) recoverySource {
	t.Helper()

	sourceApp := apptest.New(t)
	rootAdmin := sourceApp.CreateAdmin(t)
	workspace := apptest.CreateWorkspace(t, rootAdmin, "Recovered workspace")
	storageID := sourceApp.CreateLocalStorage(t, rootAdmin, workspace.Id)

	settings, err := rootAdmin.Client.GetInstanceBackupSettingsWithResponse(t.Context())
	require.NoError(t, err)
	require.NotNil(t, settings.JSON200)
	selection := *settings.JSON200
	selection.StorageIds = []uuid.UUID{storageID}
	updated, err := rootAdmin.Client.UpdateInstanceBackupSettingsWithResponse(t.Context(), selection)
	require.NoError(t, err)
	require.NotNil(t, updated.JSON200, string(updated.Body))

	sourceApp.RunJobPass(t, "instancebackups_backup_production")
	sourceApp.RunJobPass(t, "instancebackups_copy_delivery")

	backups, err := rootAdmin.Client.ListInstanceBackupsWithResponse(t.Context(), nil)
	require.NoError(t, err)
	require.NotNil(t, backups.JSON200)
	require.Len(t, backups.JSON200.Items, 1)
	storedBackup := backups.JSON200.Items[0]
	require.Len(t, storedBackup.Copies, 1)

	token, err := rootAdmin.Client.CreateInstanceBackupDownloadTokenWithResponse(t.Context(), storedBackup.Id,
		apiclient.InstanceBackupDownloadTokenCreate{CopyId: storedBackup.Copies[0].Id})
	require.NoError(t, err)
	require.NotNil(t, token.JSON201, string(token.Body))
	download, err := sourceApp.NewClient(t).DownloadInstanceBackupWithResponse(t.Context(),
		&apiclient.DownloadInstanceBackupParams{Token: token.JSON201.Token})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, download.StatusCode())

	return recoverySource{archive: download.Body, masterKey: sourceApp.ReadMasterKeyFile(t), rootEmail: rootAdmin.Email}
}

// rebuildingServer runs serveUntilStopped in the test process on a free port, as main does.
type rebuildingServer struct {
	environment *app.Environment
	address     string
	exitCodes   chan int
	stop        func()
}

func startRebuildingServer(t *testing.T, databaseDsn string) *rebuildingServer {
	t.Helper()

	t.Setenv("ENV_MODE", "production")
	t.Setenv("DATABASE_DSN", databaseDsn)
	t.Setenv("IS_DISABLE_ANONYMOUS_TELEMETRY", "true")

	rootDirectory := t.TempDir()

	environment, err := app.LoadEnvironment(t.Context(), config.LoadSpec{
		IsDatabaseRequired:       true,
		RootDirectory:            rootDirectory,
		PublishedDatabaseDsnPath: filepath.Join(rootDirectory, "never-published"),
	})
	require.NoError(t, err)

	server := &rebuildingServer{environment: environment, address: getFreeAddress(t), exitCodes: nil, stop: nil}
	server.start(t)

	return server
}

func (s *rebuildingServer) start(t *testing.T) {
	t.Helper()

	lifecycleLock, err := app.OpenServerLifecycle(t.Context(), s.environment)
	require.NoError(t, err)

	serveCtx, stopServing := context.WithCancel(context.WithoutCancel(t.Context()))
	exitCodes := make(chan int, 1)

	go func() {
		exitCodes <- serveUntilStopped(serveCtx, s.environment, lifecycleLock, s.address)
		_ = lifecycleLock.Close()
	}()

	var stopOnce sync.Once
	s.exitCodes = exitCodes
	s.stop = func() {
		stopOnce.Do(func() {
			stopServing()
			assert.Equal(t, 0, <-exitCodes)
		})
	}
	t.Cleanup(s.stop)

	requireVersionAnswers(t, s.address)
}

func (s *rebuildingServer) restart(t *testing.T) {
	t.Helper()

	s.stop()
	s.start(t)
}

func (s *rebuildingServer) requireServing(t *testing.T) {
	t.Helper()

	assert.Empty(t, s.exitCodes, "serveUntilStopped returned after the rebuild")
	requireVersionAnswers(t, s.address)
}

func (s *rebuildingServer) newClient(t *testing.T, options ...apiclient.ClientOption) *apiclient.ClientWithResponses {
	t.Helper()

	return apiclient.New(t, "http://"+s.address, options...)
}

func (s *rebuildingServer) signUp(t *testing.T) *apiclient.ClientWithResponses {
	t.Helper()

	response, err := s.newClient(t).SignUpWithResponse(t.Context(), apiclient.SignUpRequest{
		Email:                    openapi_types.Email(apptest.NewTestEmail("target-root")),
		Password:                 new(targetRootPassword),
		Name:                     "Target root",
		CloudflareTurnstileToken: nil,
	})
	require.NoError(t, err)
	require.NotNil(t, response.JSON201, string(response.Body))

	return s.newClient(t, apiclient.WithSessionToken(response.JSON201.Token))
}

// waitForSignIn waits until the rebuilt server accepts the account, which exists only in the
// recovered database.
func (s *rebuildingServer) waitForSignIn(t *testing.T, email string) *apiclient.ClientWithResponses {
	t.Helper()

	var sessionToken string

	testwait.Until(t, "the recovered root admin to sign in", func(ctx context.Context) error {
		response, err := s.newClient(t).SignInWithResponse(ctx, apiclient.SignInRequest{
			Email:                    email,
			Password:                 new(apptest.TestPassword),
			CloudflareTurnstileToken: nil,
		})
		if err != nil {
			return err
		}

		if response.JSON200 == nil {
			return fmt.Errorf("sign-in answered %d", response.StatusCode())
		}

		completed, err := response.JSON200.AsSignInCompleted()
		if err != nil {
			return err
		}

		sessionToken = completed.Token

		return nil
	})

	return s.newClient(t, apiclient.WithSessionToken(sessionToken))
}

func (s *rebuildingServer) prepareRecovery(
	t *testing.T,
	root *apiclient.ClientWithResponses,
	source recoverySource,
) uuid.UUID {
	t.Helper()

	contentType, body := newRecoveryUploadBody(t, source)

	response, err := root.CreateInstanceRecoveryWithBodyWithResponse(t.Context(), contentType, body)
	require.NoError(t, err)
	require.NotNil(t, response.JSON202, string(response.Body))

	waitForRecoveryStatus(t, root, response.JSON202.Id, apiclient.InstanceRecoveryStatusREADY)

	return response.JSON202.Id
}

func (s *rebuildingServer) readMasterKey(t *testing.T) []byte {
	t.Helper()

	masterKey, err := os.ReadFile(s.environment.Config.SecretKeyPath)
	require.NoError(t, err)

	return masterKey
}

func newRecoveryUploadBody(t *testing.T, source recoverySource) (string, *bytes.Buffer) {
	t.Helper()

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("isInsecureTransferAcknowledged", "true"))
	require.NoError(t, writer.WriteField("masterKey", string(source.masterKey)))
	archiveWriter, err := writer.CreateFormFile("archive", "databasus.dbbackup")
	require.NoError(t, err)
	_, err = archiveWriter.Write(source.archive)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	return writer.FormDataContentType(), &body
}

func applyTargetRecovery(
	t *testing.T,
	root *apiclient.ClientWithResponses,
	recoveryID uuid.UUID,
) *apiclient.ApplyInstanceRecoveryResponse {
	t.Helper()

	response, err := root.ApplyInstanceRecoveryWithResponse(t.Context(), recoveryID, apiclient.InstanceRecoveryApply{
		IsReplacementConfirmed: true,
		Proof:                  apiclient.IdentityProof{Password: new(targetRootPassword), Code: nil},
	})
	require.NoError(t, err)

	return response
}

func waitForRecoveryStatus(
	t *testing.T,
	root *apiclient.ClientWithResponses,
	recoveryID uuid.UUID,
	expectedStatus apiclient.InstanceRecoveryStatus,
) apiclient.InstanceRecovery {
	t.Helper()

	var observedRecovery apiclient.InstanceRecovery

	testwait.Until(t, "the recovery to become "+string(expectedStatus), func(ctx context.Context) error {
		response, err := root.GetInstanceRecoveryWithResponse(ctx, recoveryID)
		if err != nil {
			return err
		}

		if response.JSON200 == nil {
			return fmt.Errorf("the recovery answered %d", response.StatusCode())
		}

		if response.JSON200.Status != expectedStatus {
			return fmt.Errorf("the recovery is %s", response.JSON200.Status)
		}

		observedRecovery = *response.JSON200

		return nil
	})

	return observedRecovery
}

func getRecoveryStatus(t *testing.T, root *apiclient.ClientWithResponses) apiclient.InstanceBackupStatus {
	t.Helper()

	response, err := root.GetInstanceBackupStatusWithResponse(t.Context())
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))

	return *response.JSON200
}

func getStatusCode(t *testing.T, root *apiclient.ClientWithResponses) int {
	t.Helper()

	response, err := root.GetInstanceBackupStatusWithResponse(t.Context())
	require.NoError(t, err)

	return response.StatusCode()
}

func getFreeAddress(t *testing.T) string {
	t.Helper()

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	return address
}
