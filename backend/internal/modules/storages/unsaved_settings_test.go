package storages_test

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/sftp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func TestTestStorageSettings(t *testing.T) {
	t.Parallel()

	t.Run("connects to the entered FTP server and creates no storage when settings are tested before saving",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			ftpServer := containers.StartFtp(t)

			testResult := testUnsavedSettings(t, tm.owner, apiclient.StorageSettingsTest{
				WorkspaceId: tm.workspace.Id,
				StorageId:   nil,
				Settings: newFtpSettings(t, apiclient.FtpStorageSettings{
					Host:     ftpServer.Host,
					Port:     ftpServer.Port,
					Username: containers.FtpUsername,
					Password: new(containers.FtpPassword),
				}),
			})

			assert.True(t, testResult.IsConnected, "the test must sign in to the FTP server: %v", testResult.Error)
			assert.Empty(t, listStorages(t, tm.owner, tm.workspace.Id))
		})

	t.Run("reports not connected when the entered FTP password is wrong", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		ftpServer := containers.StartFtp(t)

		testResult := testUnsavedSettings(t, tm.owner, apiclient.StorageSettingsTest{
			WorkspaceId: tm.workspace.Id,
			StorageId:   nil,
			Settings: newFtpSettings(t, apiclient.FtpStorageSettings{
				Host:     ftpServer.Host,
				Port:     ftpServer.Port,
				Username: containers.FtpUsername,
				Password: new("wrong-password"),
			}),
		})

		assert.False(t, testResult.IsConnected)
		assert.NotNil(t, testResult.Error)
	})

	t.Run("signs in with the stored SFTP password and creates the new path while the saved path stays",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			sftpServer := containers.StartSftp(t)
			savedPath := "/" + containers.SftpUploadDir + "/saved"
			editedPath := "/" + containers.SftpUploadDir + "/edited-" + uuid.NewString()[:8]
			sftpStorage := createStorage(t, tm.owner, apiclient.StorageCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Archive",
				Settings: newSftpSettings(t, apiclient.SftpStorageSettings{
					AuthMethods:         []apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPASSWORD},
					Host:                sftpServer.Host,
					Port:                sftpServer.Port,
					Username:            containers.SftpUsername,
					Password:            new(containers.SftpPassword),
					Path:                new(savedPath),
					IsSkipHostKeyVerify: true,
				}),
			})

			testResult := testUnsavedSettings(t, tm.owner, apiclient.StorageSettingsTest{
				WorkspaceId: tm.workspace.Id,
				StorageId:   &sftpStorage.Id,
				Settings: newSftpSettings(t, apiclient.SftpStorageSettings{
					AuthMethods:         []apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPASSWORD},
					Host:                sftpServer.Host,
					Port:                sftpServer.Port,
					Username:            containers.SftpUsername,
					Path:                new(editedPath),
					IsSkipHostKeyVerify: true,
				}),
			})

			require.True(t, testResult.IsConnected, "the test must sign in with the stored password: %v",
				testResult.Error)
			assert.True(t, isSftpDirectoryPresent(t, sftpServer, editedPath))
			savedSftp, err := getStorage(t, tm.owner, sftpStorage.Id).Settings.AsSftpStorageSettings()
			require.NoError(t, err)
			assert.Equal(t, new(savedPath), savedSftp.Path)
		})

	t.Run("returns 404 not_found and connects nowhere when the named NAS storage belongs to another workspace",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			listener := startSilentListener(t)
			stranger := tm.stranger(t)
			strangerWorkspace := apptest.CreateWorkspace(t, stranger, "Other team")
			strangerStorage := createStorage(t, stranger, newNasCreation(t, strangerWorkspace.Id, listener))

			response, err := tm.owner.Client.TestStorageSettingsWithResponse(t.Context(), apiclient.StorageSettingsTest{
				WorkspaceId: tm.workspace.Id,
				StorageId:   &strangerStorage.Id,
				Settings:    newNasSettingsWithoutPassword(t, listener),
			})

			require.NoError(t, err)
			requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
			assert.False(t, listener.isConnectionPending(t))
		})

	t.Run("returns 400 invalid_request and connects nowhere when FTP settings name a saved SFTP storage",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			listener := startSilentListener(t)
			sftpStorage := createStorage(t, tm.owner, apiclient.StorageCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Archive",
				Settings: newSftpSettings(t, apiclient.SftpStorageSettings{
					AuthMethods: []apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPASSWORD},
					Host:        listener.host,
					Port:        listener.port,
					Username:    "backup",
					Password:    new("stored-password"),
				}),
			})

			response, err := tm.owner.Client.TestStorageSettingsWithResponse(t.Context(), apiclient.StorageSettingsTest{
				WorkspaceId: tm.workspace.Id,
				StorageId:   &sftpStorage.Id,
				Settings: newFtpSettings(t, apiclient.FtpStorageSettings{
					Host:     listener.host,
					Port:     listener.port,
					Username: "backup",
				}),
			})

			require.NoError(t, err)
			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest)
			assert.False(t, listener.isConnectionPending(t))
		})

	t.Run("returns 400 invalid_request and connects nowhere when S3 settings change the prefix of the named storage",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			listener := startSilentListener(t)
			endpointURL := fmt.Sprintf("http://%s:%d", listener.host, listener.port)
			s3Storage := createStorage(t, tm.owner, apiclient.StorageCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Offsite",
				Settings: newS3Settings(t, apiclient.S3StorageSettings{
					S3Bucket:    "backups",
					S3Endpoint:  new(endpointURL),
					S3AccessKey: new("stored-access"),
					S3SecretKey: new("stored-secret"),
					S3Prefix:    new("databasus/"),
				}),
			})

			response, err := tm.owner.Client.TestStorageSettingsWithResponse(t.Context(), apiclient.StorageSettingsTest{
				WorkspaceId: tm.workspace.Id,
				StorageId:   &s3Storage.Id,
				Settings: newS3Settings(t, apiclient.S3StorageSettings{
					S3Bucket:   "backups",
					S3Endpoint: new(endpointURL),
					S3Prefix:   new("elsewhere/"),
				}),
			})

			require.NoError(t, err)
			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest)
			assert.False(t, listener.isConnectionPending(t))
		})

	t.Run("returns 403 permission_denied and connects nowhere when a viewer tests settings with a saved NAS storage",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
			storedListener := startSilentListener(t)
			viewerListener := startSilentListener(t)
			nasStorage := createStorage(t, tm.owner, newNasCreation(t, tm.workspace.Id, storedListener))

			response, err := viewer.Client.TestStorageSettingsWithResponse(t.Context(), apiclient.StorageSettingsTest{
				WorkspaceId: tm.workspace.Id,
				StorageId:   &nasStorage.Id,
				Settings:    newNasSettingsWithoutPassword(t, viewerListener),
			})

			require.NoError(t, err)
			requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
				apiclient.ProblemCodePermissionDenied)
			assert.False(t, viewerListener.isConnectionPending(t))
			assert.False(t, storedListener.isConnectionPending(t))
		})

	t.Run("reports a local storage as connected when a workspace member tests unsaved settings", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)

		testResult := testUnsavedSettings(t, member, apiclient.StorageSettingsTest{
			WorkspaceId: tm.workspace.Id,
			StorageId:   nil,
			Settings:    newLocalSettings(t),
		})

		assert.True(t, testResult.IsConnected)
		assert.Empty(t, listStorages(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 400 invalid_request and connects nowhere when the unsaved settings are invalid", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		listener := startSilentListener(t)

		response, err := tm.owner.Client.TestStorageSettingsWithResponse(t.Context(), apiclient.StorageSettingsTest{
			WorkspaceId: tm.workspace.Id,
			StorageId:   nil,
			Settings:    newNasSettingsWithoutPassword(t, listener),
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		assert.False(t, listener.isConnectionPending(t))
	})

	t.Run("returns 401 unauthenticated without a session", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)

		response, err := tm.app.NewClient(t).TestStorageSettingsWithResponse(t.Context(), apiclient.StorageSettingsTest{
			WorkspaceId: tm.workspace.Id,
			StorageId:   nil,
			Settings:    newLocalSettings(t),
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})
}

func testUnsavedSettings(
	t *testing.T,
	session apptest.Session,
	settingsTest apiclient.StorageSettingsTest,
) apiclient.StorageTestResult {
	t.Helper()

	response, err := session.Client.TestStorageSettingsWithResponse(t.Context(), settingsTest)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func newNasCreation(t *testing.T, workspaceID uuid.UUID, listener silentListener) apiclient.StorageCreate {
	t.Helper()

	return apiclient.StorageCreate{
		WorkspaceId: workspaceID,
		Name:        "Share",
		Settings: newNasSettings(t, apiclient.NasStorageSettings{
			Host:     listener.host,
			Port:     listener.port,
			Share:    "backups",
			Username: "backup",
			Password: new("stored-nas-password"),
		}),
	}
}

func newNasSettingsWithoutPassword(t *testing.T, listener silentListener) apiclient.StorageSettings {
	t.Helper()

	return newNasSettings(t, apiclient.NasStorageSettings{
		Host:     listener.host,
		Port:     listener.port,
		Share:    "backups",
		Username: "backup",
	})
}

const acceptWindow = 100 * time.Millisecond

// silentListener is a TCP port no server answers on. A connection the backend opens to it waits in
// the accept queue, so the test can tell whether anything reached it.
type silentListener struct {
	listener *net.TCPListener
	host     string
	port     int
}

func startSilentListener(t *testing.T) silentListener {
	t.Helper()

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	address, isTCPAddress := listener.Addr().(*net.TCPAddr)
	require.True(t, isTCPAddress)

	return silentListener{listener: listener, host: "127.0.0.1", port: address.Port}
}

// isConnectionPending watches the accept queue for acceptWindow after the request has answered: any
// connection the request opened is queued by then, and Accept with an expired deadline would not
// look at the queue at all.
func (l silentListener) isConnectionPending(t *testing.T) bool {
	t.Helper()

	require.NoError(t, l.listener.SetDeadline(time.Now().Add(acceptWindow)))

	connection, err := l.listener.Accept()
	if err == nil {
		_ = connection.Close()

		return true
	}

	var netErr net.Error
	require.True(t, errors.As(err, &netErr) && netErr.Timeout(), "accept failed: %v", err)

	return false
}

func isSftpDirectoryPresent(t *testing.T, server containers.Endpoint, path string) bool {
	t.Helper()

	sshClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", server.Host, server.Port), &ssh.ClientConfig{
		User: containers.SftpUsername,
		Auth: []ssh.AuthMethod{ssh.Password(containers.SftpPassword)},
		// The test container generates its host key at start, so there is nothing to pin.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	})
	require.NoError(t, err)
	defer func() { _ = sshClient.Close() }()

	sftpClient, err := sftp.NewClient(sshClient)
	require.NoError(t, err)
	defer func() { _ = sftpClient.Close() }()

	fileInfo, err := sftpClient.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}

	require.NoError(t, err)

	return fileInfo.IsDir()
}
