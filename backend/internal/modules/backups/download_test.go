package backups_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

// getPostgresArchiveSignature() starts every archive pg_dump writes in its custom format.
func getPostgresArchiveSignature() []byte { return []byte("PGDMP") }

//nolint:tparallel // The subtests share one source server.
func TestCreateDownloadToken(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("issues a token that expires in five minutes with the file name", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)

		issuedToken := issueDownloadToken(t, tm.owner, database.backup.Id)

		assert.NotEmpty(t, issuedToken.Token)
		assert.WithinDuration(t, tm.app.Clock.Now().Add(5*time.Minute), issuedToken.ExpiresAt, time.Second)
		assert.Equal(t, ".dump", filepath.Ext(issuedToken.FileName))
	})

	t.Run("returns 404 not_found for a backup that does not exist", func(t *testing.T) {
		tm := newTeam(t)

		response := requestDownloadToken(t, tm.owner, uuid.New())

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})

	t.Run("returns 401 unauthenticated without a session", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)

		response, err := tm.app.NewClient(t).CreateDownloadTokenWithResponse(t.Context(),
			apiclient.DownloadTokenCreate{BackupId: database.backup.Id})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})
}

//nolint:tparallel // The subtests share one source server.
func TestDownloadLogicalBackup(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	for _, storageEncryption := range []struct {
		name       string
		encryption apiclient.BackupEncryption
	}{
		{name: "unencrypted", encryption: apiclient.BackupEncryptionNONE},
		{name: "encrypted", encryption: apiclient.BackupEncryptionENCRYPTED},
	} {
		t.Run(
			"does not compress the backup stream when the client accepts gzip and storage is "+storageEncryption.name,
			func(t *testing.T) {
				team := newTeam(t)
				backup := completeDownloadBackup(t, downloadBackupSpec{
					team: team, server: server, encryption: storageEncryption.encryption,
				})
				issuedToken := issueDownloadToken(t, team.owner, backup.Id)

				response, err := team.app.NewClient(t).DownloadLogicalBackupWithResponse(
					t.Context(), backup.Id, &apiclient.DownloadLogicalBackupParams{Token: issuedToken.Token},
					func(_ context.Context, request *http.Request) error {
						request.Header.Set("Accept-Encoding", "gzip")
						return nil
					},
				)
				require.NoError(t, err)

				require.Equal(t, http.StatusOK, response.StatusCode())
				assert.False(t, response.HTTPResponse.Uncompressed)
				require.Empty(t, response.HTTPResponse.Header.Get("Content-Encoding"))
				assert.Equal(t, strconv.Itoa(len(response.Body)), response.HTTPResponse.Header.Get("Content-Length"))
				assert.Equal(t, int64(len(response.Body)), response.HTTPResponse.ContentLength)
				assert.True(t, bytes.HasPrefix(response.Body, getPostgresArchiveSignature()))
				assert.Contains(t, response.HTTPResponse.Header.Get("Content-Disposition"), issuedToken.FileName)
			},
		)

		t.Run("announces the exact file length when storage is "+storageEncryption.name, func(t *testing.T) {
			team := newTeam(t)
			backup := completeDownloadBackup(t, downloadBackupSpec{
				team: team, server: server, encryption: storageEncryption.encryption,
			})
			issuedToken := issueDownloadToken(t, team.owner, backup.Id)

			response, err := team.app.NewClient(t).DownloadLogicalBackupWithResponse(
				t.Context(), backup.Id, &apiclient.DownloadLogicalBackupParams{Token: issuedToken.Token},
				func(_ context.Context, request *http.Request) error {
					request.Header.Set("Accept-Encoding", "identity")
					return nil
				},
			)
			require.NoError(t, err)

			require.Equal(t, http.StatusOK, response.StatusCode())
			require.Greater(t, len(response.Body), 2*1024)
			require.NotZero(t, len(response.Body)%(1024*1024))
			assert.Equal(t, strconv.Itoa(len(response.Body)), response.HTTPResponse.Header.Get("Content-Length"))
			assert.Equal(t, int64(len(response.Body)), response.HTTPResponse.ContentLength)
			assert.True(t, bytes.HasPrefix(response.Body, getPostgresArchiveSignature()))
		})
	}

	t.Run("returns the file once and 401 unauthenticated to the second use of the token", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		issuedToken := issueDownloadToken(t, tm.owner, database.backup.Id)

		response := requestBackupDownload(t, tm, database.backup.Id, issuedToken.Token)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.True(t, bytes.HasPrefix(response.Body, getPostgresArchiveSignature()))
		require.NotNil(t, response.Headers200)
		require.NotNil(t, response.Headers200.ContentDisposition)
		assert.Contains(t, *response.Headers200.ContentDisposition, issuedToken.FileName)

		repeatedResponse := requestBackupDownload(t, tm, database.backup.Id, issuedToken.Token)

		requireProblem(t, repeatedResponse.StatusCode(), repeatedResponse.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("serves the file to exactly one of two requests racing with one token", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		issuedToken := issueDownloadToken(t, tm.owner, database.backup.Id)

		const requestCount = 2

		racingResponses := make([]*apiclient.DownloadLogicalBackupResponse, requestCount)
		startSignal := make(chan struct{})

		var requestsDone sync.WaitGroup

		for requestIndex := range requestCount {
			requestsDone.Go(func() {
				<-startSignal

				response, err := tm.app.NewClient(t).DownloadLogicalBackupWithResponse(t.Context(),
					database.backup.Id, &apiclient.DownloadLogicalBackupParams{Token: issuedToken.Token})
				if err != nil {
					t.Errorf("download the backup: %v", err)

					return
				}

				racingResponses[requestIndex] = response
			})
		}

		close(startSignal)
		requestsDone.Wait()

		servedCount := 0

		for _, response := range racingResponses {
			require.NotNil(t, response)

			if response.StatusCode() == http.StatusOK {
				servedCount++

				assert.True(t, bytes.HasPrefix(response.Body, getPostgresArchiveSignature()))

				continue
			}

			requireRefusedRacingDownload(t, response)
		}

		assert.Equal(t, 1, servedCount)

		thirdResponse := requestBackupDownload(t, tm, database.backup.Id, issuedToken.Token)
		requireProblem(t, thirdResponse.StatusCode(), thirdResponse.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("returns 401 unauthenticated without a file for a token past five minutes", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		issuedToken := issueDownloadToken(t, tm.owner, database.backup.Id)
		tm.app.Clock.Advance(5*time.Minute + time.Second)

		response := requestBackupDownload(t, tm, database.backup.Id, issuedToken.Token)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
		assert.False(t, bytes.Contains(response.Body, getPostgresArchiveSignature()))
	})

	t.Run("returns 401 unauthenticated for a token of another backup and keeps the token usable", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		otherBackup := completeRestorableBackup(t, tm.owner, database.databaseID)
		issuedToken := issueDownloadToken(t, tm.owner, database.backup.Id)

		response := requestBackupDownload(t, tm, otherBackup.Id, issuedToken.Token)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)

		downloadedFile := downloadBackupFile(t, tm, database.backup.Id, issuedToken.Token)
		assert.True(t, bytes.HasPrefix(downloadedFile, getPostgresArchiveSignature()))
	})

	t.Run("returns 401 unauthenticated for a token that was never issued", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)

		response := requestBackupDownload(t, tm, database.backup.Id, "never-issued-token")

		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("returns the dump decrypted when the backup is encrypted", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		saveLogicalBackupSettings(t, tm.owner, source.databaseID, source.storageID, newLogicalBackupSettings(),
			apiclient.BackupEncryptionENCRYPTED)
		encryptedBackup := completeRestorableBackup(t, tm.owner, source.databaseID)
		require.Equal(t, apiclient.BackupEncryptionENCRYPTED, encryptedBackup.Encryption)
		storedFile, err := os.ReadFile(tm.app.GetLocalStorageFilePath(getBackupFileName(t, tm, encryptedBackup.Id)))
		require.NoError(t, err)
		issuedToken := issueDownloadToken(t, tm.owner, encryptedBackup.Id)

		downloadedFile := downloadBackupFile(t, tm, encryptedBackup.Id, issuedToken.Token)

		assert.True(t, bytes.HasPrefix(downloadedFile, getPostgresArchiveSignature()))
		assert.False(t, bytes.HasPrefix(storedFile, getPostgresArchiveSignature()))
	})
}

//nolint:tparallel // The subtests share one source server.
func TestDownloadLogicalBackupWhileAnotherStreams(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("returns 409 already_running and issues no token while a download streams", func(t *testing.T) {
		tm := newTeam(t)
		largeBackup := completeLargeBackup(t, tm, server)
		streamingResponse := openBackupDownloadStream(t, tm, largeBackup.Id,
			issueDownloadToken(t, tm.owner, largeBackup.Id).Token)
		require.Equal(t, http.StatusOK, streamingResponse.StatusCode)
		tokenCountBefore := countDownloadTokens(t, tm, largeBackup.Id)

		response := requestDownloadToken(t, tm.owner, largeBackup.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodeAlreadyRunning)
		assert.Equal(t, tokenCountBefore, countDownloadTokens(t, tm, largeBackup.Id))
		require.NoError(t, streamingResponse.Body.Close())
	})

	t.Run("refuses a second token while the first download streams and serves it after", func(t *testing.T) {
		tm := newTeam(t)
		largeBackup := completeLargeBackup(t, tm, server)
		firstToken := issueDownloadToken(t, tm.owner, largeBackup.Id)
		secondToken := issueDownloadToken(t, tm.owner, largeBackup.Id)
		streamingResponse := openBackupDownloadStream(t, tm, largeBackup.Id, firstToken.Token)
		require.Equal(t, http.StatusOK, streamingResponse.StatusCode)

		refusedResponse := requestBackupDownload(t, tm, largeBackup.Id, secondToken.Token)

		requireProblem(t, refusedResponse.StatusCode(), refusedResponse.Body, http.StatusConflict,
			apiclient.ProblemCodeAlreadyRunning)

		firstFile, err := io.ReadAll(streamingResponse.Body)
		require.NoError(t, err)
		require.NoError(t, streamingResponse.Body.Close())

		secondFile := waitUntilBackupDownloads(t, tm, largeBackup.Id, secondToken.Token)
		assert.True(t, bytes.HasPrefix(firstFile, getPostgresArchiveSignature()))
		assert.Equal(t, firstFile, secondFile)
	})
}

// requireRefusedRacingDownload accepts both answers the loser of a token race may get: the user's
// stream is still held by the winner, or the winner already consumed the token.
func requireRefusedRacingDownload(t *testing.T, response *apiclient.DownloadLogicalBackupResponse) {
	t.Helper()

	switch response.StatusCode() {
	case http.StatusConflict:
		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodeAlreadyRunning)
	default:
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	}

	assert.False(t, bytes.Contains(response.Body, getPostgresArchiveSignature()))
}

// completeLargeBackup backs up a database of random data larger than the socket buffers of a local
// connection hold, so a download whose body nobody reads keeps streaming.
func completeLargeBackup(t *testing.T, tm team, server restoreServer) apiclient.LogicalBackup {
	t.Helper()

	source := newBackupSourceDatabase(t, tm, server)
	server.exec(t, source.sourceName, `CREATE TABLE filler AS
		SELECT (
			SELECT decode(string_agg(md5(random()::text || part::text || row_number::text), ''), 'hex')
			FROM generate_series(1, 65536) AS part
		) AS payload
		FROM generate_series(1, 32) AS row_number`)

	return completeRestorableBackup(t, tm.owner, source.databaseID)
}

type downloadBackupSpec struct {
	team       team
	server     restoreServer
	encryption apiclient.BackupEncryption
}

func completeDownloadBackup(t *testing.T, spec downloadBackupSpec) apiclient.LogicalBackup {
	t.Helper()

	source := newBackupSourceDatabase(t, spec.team, spec.server)
	spec.server.exec(t, source.sourceName, `INSERT INTO items
		SELECT item_number + 3, md5(item_number::text) FROM generate_series(1, 1000) AS item_number`)
	saveLogicalBackupSettings(t, spec.team.owner, source.databaseID, source.storageID, newLogicalBackupSettings(),
		spec.encryption)

	return completeRestorableBackup(t, spec.team.owner, source.databaseID)
}

func countDownloadTokens(t *testing.T, tm team, backupID uuid.UUID) int {
	t.Helper()

	return countRows(t, tm, "SELECT COUNT(*) FROM backups_download_tokens WHERE backup_id = ?", backupID)
}
