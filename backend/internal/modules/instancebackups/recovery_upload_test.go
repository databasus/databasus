package instancebackups_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

const recoveryPreparationJobName = "instancebackups_recovery_preparation"

// uploadedKey has the surrounding whitespace an operator may paste, which the upload keeps.
const uploadedKey = " uploaded source key that is long enough to be a Databasus key \n"

func TestCreateInstanceRecovery(t *testing.T) {
	t.Parallel()

	t.Run("accepts an upload without a declared length and reports its preparation failure", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		response := uploadRecovery(t, owner.RootAdmin, newRecoveryUpload([]byte("not a Databasus backup")))
		require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))
		owner.App.RunJobPass(t, recoveryPreparationJobName)
		failedRecovery := getRecovery(t, owner.RootAdmin, response.JSON202.Id)

		assert.Equal(t, apiclient.InstanceRecoveryStatusVALIDATING, response.JSON202.Status)
		assert.Equal(t, apiclient.InstanceRecoveryOriginUPLOAD, response.JSON202.Origin)
		assert.Len(t, response.JSON202.ArchiveSha256, 64)
		assert.Equal(t, apiclient.InstanceRecoveryStatusFAILED, failedRecovery.Status)
		require.NotNil(t, failedRecovery.Failure)
		assert.Equal(t, apiclient.InstanceRecoveryFailureARCHIVEINVALID, *failedRecovery.Failure)
		assert.NotContains(t, string(response.Body), "uploaded source key")
	})

	t.Run("returns 403 permission_denied to another administrator and stages nothing", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		refusedResponse := uploadRecovery(t, owner.OtherAdmin, newRecoveryUpload([]byte("archive")))
		rootResponse := uploadRecovery(t, owner.RootAdmin, newRecoveryUpload([]byte("archive")))

		requireProblemCode(t, refusedResponse.StatusCode(), refusedResponse.Body,
			http.StatusForbidden, apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, http.StatusAccepted, rootResponse.StatusCode(), string(rootResponse.Body))
	})

	t.Run("returns 401 unauthenticated without a session", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		upload := newRecoveryUpload([]byte("archive"))

		response, err := owner.App.NewClient(t).CreateInstanceRecoveryWithBodyWithResponse(
			t.Context(), upload.contentType, upload.body,
		)

		require.NoError(t, err)
		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusUnauthorized, apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("returns 400 invalid_request for an upload without a master key", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		response := uploadRecovery(t, owner.RootAdmin, newRecoveryUploadWith([]recoveryUploadField{
			{name: "isInsecureTransferAcknowledged", content: []byte("true")},
			{name: "archive", fileName: "databasus.dbbackup", content: []byte("archive")},
		}))

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest)
		assert.Contains(t, string(response.Body), `"field":"masterKey"`)
	})

	t.Run("returns 400 invalid_request over HTTP without the acknowledgment", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		response := uploadRecovery(t, owner.RootAdmin, newRecoveryUploadWith([]recoveryUploadField{
			{name: "masterKey", content: []byte(uploadedKey)},
			{name: "archive", fileName: "databasus.dbbackup", content: []byte("archive")},
		}))

		requireProblemCode(t, response.StatusCode(), response.Body,
			http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest)
		assert.Contains(t, string(response.Body), `"field":"isInsecureTransferAcknowledged"`)
	})

	t.Run("accepts an upload without the acknowledgment over an HTTPS public address", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t, apptest.WithPublicURL("https://databasus.example.com"))

		response := uploadRecovery(t, owner.RootAdmin, newRecoveryUploadWith([]recoveryUploadField{
			{name: "masterKey", content: []byte(uploadedKey)},
			{name: "archive", fileName: "databasus.dbbackup", content: []byte("archive")},
		}))

		assert.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))
	})

	t.Run("returns 413 archive_too_large for an archive beyond the configured limit", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t, apptest.WithRecoveryMaxArchiveBytes(16))

		refusedResponse := uploadRecovery(t, owner.RootAdmin, newRecoveryUpload(bytes.Repeat([]byte("a"), 17)))
		acceptedResponse := uploadRecovery(t, owner.RootAdmin, newRecoveryUpload(bytes.Repeat([]byte("a"), 16)))

		requireProblemCode(t, refusedResponse.StatusCode(), refusedResponse.Body,
			http.StatusRequestEntityTooLarge, apiclient.ProblemCodeArchiveTooLarge)
		assert.Equal(t, http.StatusAccepted, acceptedResponse.StatusCode(), string(acceptedResponse.Body))
	})

	t.Run("returns 400 invalid_request and stages nothing when the body ends before its declared length",
		func(t *testing.T) {
			t.Parallel()
			owner := newInstanceOwner(t)
			upload := newRecoveryUpload(bytes.Repeat([]byte("a"), 4096))
			sentBody, err := io.ReadAll(upload.body)
			require.NoError(t, err)

			status := sendTruncatedUpload(t, owner, upload.contentType, sentBody)
			nextResponse := uploadRecovery(t, owner.RootAdmin, newRecoveryUpload([]byte("archive")))

			assert.Equal(t, http.StatusBadRequest, status)
			assert.Equal(t, http.StatusAccepted, nextResponse.StatusCode(), string(nextResponse.Body))
		})

	t.Run("returns 409 already_running while another recovery is pending", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		firstResponse := uploadRecovery(t, owner.RootAdmin, newRecoveryUpload([]byte("archive")))
		require.Equal(t, http.StatusAccepted, firstResponse.StatusCode(), string(firstResponse.Body))

		secondResponse := uploadRecovery(t, owner.RootAdmin, newRecoveryUpload([]byte("archive")))

		requireProblemCode(t, secondResponse.StatusCode(), secondResponse.Body,
			http.StatusConflict, apiclient.ProblemCodeAlreadyRunning)
	})
}

func TestCancelInstanceRecovery(t *testing.T) {
	t.Parallel()

	t.Run("cancels a pending recovery and lets root admin upload again", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		uploadResponse := uploadRecovery(t, owner.RootAdmin, newRecoveryUpload([]byte("archive")))
		require.Equal(t, http.StatusAccepted, uploadResponse.StatusCode(), string(uploadResponse.Body))

		cancelResponse, err := owner.RootAdmin.Client.CancelInstanceRecoveryWithResponse(
			t.Context(), uploadResponse.JSON202.Id,
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, cancelResponse.StatusCode(), string(cancelResponse.Body))
		nextResponse := uploadRecovery(t, owner.RootAdmin, newRecoveryUpload([]byte("archive")))

		assert.Equal(t, apiclient.InstanceRecoveryStatusCANCELLED, cancelResponse.JSON200.Status)
		assert.Nil(t, cancelResponse.JSON200.Failure)
		assert.Equal(t, apiclient.InstanceRecoveryStatusCANCELLED,
			getRecovery(t, owner.RootAdmin, uploadResponse.JSON202.Id).Status)
		assert.Equal(t, http.StatusAccepted, nextResponse.StatusCode(), string(nextResponse.Body))
	})

	t.Run("returns 403 permission_denied to another administrator and keeps the recovery", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		uploadResponse := uploadRecovery(t, owner.RootAdmin, newRecoveryUpload([]byte("archive")))
		require.Equal(t, http.StatusAccepted, uploadResponse.StatusCode(), string(uploadResponse.Body))

		cancelResponse, err := owner.OtherAdmin.Client.CancelInstanceRecoveryWithResponse(
			t.Context(), uploadResponse.JSON202.Id,
		)

		require.NoError(t, err)
		requireProblemCode(t, cancelResponse.StatusCode(), cancelResponse.Body,
			http.StatusForbidden, apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, apiclient.InstanceRecoveryStatusVALIDATING,
			getRecovery(t, owner.RootAdmin, uploadResponse.JSON202.Id).Status)
	})

	t.Run("returns 404 not_found for a recovery the journal does not hold", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)

		cancelResponse, err := owner.RootAdmin.Client.CancelInstanceRecoveryWithResponse(t.Context(), uuid.New())

		require.NoError(t, err)
		requireProblemCode(t, cancelResponse.StatusCode(), cancelResponse.Body,
			http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})

	t.Run("returns 409 recovery_not_ready for a recovery that already failed", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		uploadResponse := uploadRecovery(t, owner.RootAdmin, newRecoveryUpload([]byte("archive")))
		require.Equal(t, http.StatusAccepted, uploadResponse.StatusCode(), string(uploadResponse.Body))
		owner.App.RunJobPass(t, recoveryPreparationJobName)

		cancelResponse, err := owner.RootAdmin.Client.CancelInstanceRecoveryWithResponse(
			t.Context(), uploadResponse.JSON202.Id,
		)

		require.NoError(t, err)
		requireProblemCode(t, cancelResponse.StatusCode(), cancelResponse.Body,
			http.StatusConflict, apiclient.ProblemCodeRecoveryNotReady)
	})
}

type recoveryUploadField struct {
	name     string
	fileName string
	content  []byte
}

// recoveryUpload is a multipart body sent without a declared length, as a browser streaming a
// large file may.
type recoveryUpload struct {
	contentType string
	body        io.Reader
}

// newRecoveryUpload is what the import form sends over HTTP: the acknowledgment, the key, then the
// archive.
func newRecoveryUpload(archiveContent []byte) recoveryUpload {
	return newRecoveryUploadWith([]recoveryUploadField{
		{name: "isInsecureTransferAcknowledged", content: []byte("true")},
		{name: "masterKey", content: []byte(uploadedKey)},
		{name: "archive", fileName: "databasus.dbbackup", content: archiveContent},
	})
}

func newRecoveryUploadWith(fields []recoveryUploadField) recoveryUpload {
	var body bytes.Buffer

	writer := multipart.NewWriter(&body)
	for _, field := range fields {
		var fieldWriter io.Writer
		if field.fileName == "" {
			fieldWriter, _ = writer.CreateFormField(field.name)
		} else {
			fieldWriter, _ = writer.CreateFormFile(field.name, field.fileName)
		}

		_, _ = fieldWriter.Write(field.content)
	}

	_ = writer.Close()

	return recoveryUpload{
		contentType: writer.FormDataContentType(),
		body:        struct{ io.Reader }{&body},
	}
}

func uploadRecovery(
	t *testing.T,
	session apptest.Session,
	upload recoveryUpload,
) *apiclient.CreateInstanceRecoveryResponse {
	t.Helper()

	response, err := session.Client.CreateInstanceRecoveryWithBodyWithResponse(
		t.Context(), upload.contentType, upload.body,
	)
	require.NoError(t, err)

	return response
}

// sendTruncatedUpload declares a longer body than it sends and stops writing, as a client that
// disconnects in the middle of an upload does. The contract client cannot send such a request.
func sendTruncatedUpload(t *testing.T, owner instanceOwner, contentType string, body []byte) int {
	t.Helper()

	address, err := url.Parse(owner.App.URL)
	require.NoError(t, err)
	connection, err := (&net.Dialer{}).DialContext(t.Context(), "tcp", address.Host)
	require.NoError(t, err)
	defer func() { _ = connection.Close() }()

	sentBody := body[:len(body)/2]
	request := fmt.Sprintf("POST /api/v1/instance-recoveries HTTP/1.1\r\nHost: %s\r\n"+
		"Authorization: Bearer %s\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n",
		address.Host, owner.RootAdmin.Token, contentType, len(body))
	_, err = connection.Write(append([]byte(request), sentBody...))
	require.NoError(t, err)
	tcpConnection, isTCP := connection.(*net.TCPConn)
	require.True(t, isTCP)
	require.NoError(t, tcpConnection.CloseWrite())

	response, err := http.ReadResponse(bufio.NewReader(connection), nil)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	return response.StatusCode
}

func getRecovery(t *testing.T, session apptest.Session, recoveryID uuid.UUID) apiclient.InstanceRecovery {
	t.Helper()

	response, err := session.Client.GetInstanceRecoveryWithResponse(t.Context(), recoveryID)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))

	return *response.JSON200
}
