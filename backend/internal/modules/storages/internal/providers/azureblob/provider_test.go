package azureblob

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/streaming"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	test_utils "databasus-backend/internal/lib/httpclienttest"
	io_utils "databasus-backend/internal/lib/iox"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/modules/storages/internal/providers/providertest"
)

func startAzurite(t *testing.T) (*blobContainer, *azblob.Client, string) {
	t.Helper()

	endpoint := containers.StartAzurite(t)
	address := fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
	containerName := "uncommitted-blocks"

	connectionString := fmt.Sprintf(
		"DefaultEndpointsProtocol=http;AccountName=%s;AccountKey=%s;BlobEndpoint=http://%s/%s",
		containers.AzuriteAccountName, containers.AzuriteAccountKey, address, containers.AzuriteAccountName,
	)

	client, err := azblob.NewClientFromConnectionString(connectionString, nil)
	require.NoError(t, err)

	_, err = client.CreateContainer(t.Context(), containerName, nil)
	require.NoError(t, err)

	testedContainer := &blobContainer{
		provider: New(
			Deps{Repositories: Repositories{settings: nil}, FieldEncryptor: providertest.NewFieldEncryptor(t)},
		),
		settings: Settings{
			AuthMethod:       AuthMethodConnectionString,
			ConnectionString: connectionString,
			ContainerName:    containerName,
		},
	}

	return testedContainer, client, containerName
}

func stageBlock(t *testing.T, client *azblob.Client, containerName, blobName string) {
	t.Helper()

	blockClient := client.ServiceClient().NewContainerClient(containerName).NewBlockBlobClient(blobName)

	_, err := blockClient.StageBlock(
		t.Context(),
		base64.StdEncoding.EncodeToString([]byte("000000")),
		streaming.NopCloser(bytes.NewReader(bytes.Repeat([]byte("x"), 1024))),
		nil,
	)
	require.NoError(t, err)
}

func getBlockList(t *testing.T, client *azblob.Client, containerName, blobName string) blockblob.GetBlockListResponse {
	t.Helper()

	blockClient := client.ServiceClient().NewContainerClient(containerName).NewBlockBlobClient(blobName)

	blocks, err := blockClient.GetBlockList(t.Context(), blockblob.BlockListTypeAll, nil)
	require.NoError(t, err)

	return blocks
}

func TestDiscardUncommittedBlocks(t *testing.T) {
	// Azurite's DeleteBlob succeeds on a blob holding only uncommitted blocks and drops
	// them, while Azure returns 404 and keeps charging for them for a week. An
	// end-to-end assertion therefore cannot separate a fixed build from an unfixed one,
	// so the discarding step is exercised on its own.
	t.Run("turns the blocks of an interrupted upload into a blob the delete removes", func(t *testing.T) {
		testedContainer, client, containerName := startAzurite(t)
		blobName := "interrupted-" + uuid.New().String()

		stageBlock(t, client, containerName, blobName)
		require.Len(t, getBlockList(t, client, containerName, blobName).UncommittedBlocks, 1)

		require.NoError(t, testedContainer.discardUncommittedBlocks(t.Context(), client, blobName))

		assert.Empty(t, getBlockList(t, client, containerName, blobName).UncommittedBlocks,
			"the staged blocks must not survive as an invisible charge")

		properties, err := client.ServiceClient().NewContainerClient(containerName).
			NewBlobClient(blobName).GetProperties(t.Context(), nil)

		require.NoError(t, err, "committing an empty list leaves a zero-length blob that DeleteFile removes")
		assert.EqualValues(t, 0, *properties.ContentLength)
	})

	t.Run("leaves the content of an existing blob alone", func(t *testing.T) {
		testedContainer, client, containerName := startAzurite(t)
		blobName := "committed-" + uuid.New().String()

		_, err := client.UploadBuffer(t.Context(), containerName, blobName, []byte("payload"), nil)
		require.NoError(t, err)

		// A single Put Blob leaves no committed block list, so an existing blob with
		// stale staged blocks is exactly the case that would be truncated by a guard
		// that looked at the committed list instead of at existence.
		stageBlock(t, client, containerName, blobName)

		blocks := getBlockList(t, client, containerName, blobName)
		require.Empty(t, blocks.CommittedBlocks, "the case only discriminates while Put Blob leaves no committed list")
		require.Len(t, blocks.UncommittedBlocks, 1)

		require.NoError(t, testedContainer.discardUncommittedBlocks(t.Context(), client, blobName))

		properties, err := client.ServiceClient().NewContainerClient(containerName).
			NewBlobClient(blobName).GetProperties(t.Context(), nil)

		require.NoError(t, err)
		assert.EqualValues(t, len("payload"), *properties.ContentLength,
			"an existing blob must not be emptied by cleanup")
	})

	t.Run("succeeds when nothing exists", func(t *testing.T) {
		testedContainer, client, _ := startAzurite(t)

		assert.NoError(t, testedContainer.discardUncommittedBlocks(t.Context(), client, "absent-"+uuid.New().String()))
	})
}

func TestSaveFile(t *testing.T) {
	t.Run("saves the file when a block is throttled for longer than the client library retries", func(t *testing.T) {
		refusalCount := azureSDKDefaultMaxRetries + 3
		refusedContainer, client, containerName, proxy := startRefusedAzurite(t, providertest.RefusingProxySpec{
			TargetURL:    "",
			IsRefusable:  isStageBlock,
			RefusalCount: refusalCount,
			Status:       http.StatusServiceUnavailable,
			ContentType:  "application/xml",
			Body:         azureErrorBody("ServerBusy"),
		})

		persistedBytes := bytes.Repeat([]byte("backup"), 200*1024)
		fileName := uuid.NewString()

		require.NoError(t, refusedContainer.saveFile(
			t.Context(), providertest.NewDiscardLogger(), fileName, bytes.NewReader(persistedBytes),
		))

		assert.Equal(t, refusalCount+1, proxy.GetRefusableCount())
		assert.Equal(t, persistedBytes, downloadBlob(t, client, containerName, fileName))
	})

	t.Run("saves the file when its block list commit is refused for a while", func(t *testing.T) {
		refusedContainer, client, containerName, proxy := startRefusedAzurite(
			t,
			uncheckedAzureRefusal(isCommitBlockList, 3),
		)

		persistedBytes := bytes.Repeat([]byte("backup"), 1024)
		fileName := uuid.NewString()

		require.NoError(t, refusedContainer.saveFile(
			t.Context(), providertest.NewDiscardLogger(), fileName, bytes.NewReader(persistedBytes),
		))

		assert.Equal(t, 4, proxy.GetRefusableCount())
		assert.Equal(t, persistedBytes, downloadBlob(t, client, containerName, fileName))
	})

	t.Run("saves an empty file when its upload is refused for a while", func(t *testing.T) {
		fileName := uuid.NewString()
		refusedContainer, client, containerName, proxy := startRefusedAzurite(
			t, uncheckedAzureRefusal(isBlobWrite(fileName), 3),
		)

		require.NoError(t, refusedContainer.saveFile(
			t.Context(), providertest.NewDiscardLogger(), fileName, bytes.NewReader(nil),
		))

		assert.Equal(t, 4, proxy.GetRefusableCount())
		assert.Empty(t, downloadBlob(t, client, containerName, fileName))
	})

	t.Run("returns an error after one request when a block upload is denied", func(t *testing.T) {
		refusedContainer, client, containerName, proxy := startRefusedAzurite(t, providertest.RefusingProxySpec{
			TargetURL:    "",
			IsRefusable:  isStageBlock,
			RefusalCount: 100,
			Status:       http.StatusForbidden,
			ContentType:  "application/xml",
			Body:         azureErrorBody("AuthorizationFailure"),
		})

		fileName := uuid.NewString()
		err := refusedContainer.saveFile(
			t.Context(), providertest.NewDiscardLogger(), fileName, bytes.NewReader([]byte("backup")),
		)

		responseError, isResponseError := errors.AsType[*azcore.ResponseError](err)
		require.True(t, isResponseError)
		assert.Equal(t, http.StatusForbidden, responseError.StatusCode)
		assert.Equal(t, 1, proxy.GetRefusableCount())

		_, err = client.ServiceClient().NewContainerClient(containerName).NewBlobClient(fileName).
			GetProperties(t.Context(), nil)
		assert.True(t, isAzureNotFound(err))
	})
}

// azureSDKDefaultMaxRetries is how often the Azure SDK repeats a refused request on its own before
// the upload's repeats take over.
const azureSDKDefaultMaxRetries = 3

// startRefusedAzurite reaches a fresh Azurite container through a proxy that answers the requests
// refusal matches, with retry delays shrunk so a test does not wait out the production backoff.
func startRefusedAzurite(
	t *testing.T,
	refusal providertest.RefusingProxySpec,
) (*blobContainer, *azblob.Client, string, *providertest.RefusingProxy) {
	t.Helper()

	_, client, containerName := startAzurite(t)

	azuriteURL, err := url.Parse(client.URL())
	require.NoError(t, err)

	refusal.TargetURL = azuriteURL.Scheme + "://" + azuriteURL.Host
	proxy := providertest.StartRefusingProxy(t, refusal)

	connectionString := fmt.Sprintf(
		"DefaultEndpointsProtocol=http;AccountName=%s;AccountKey=%s;BlobEndpoint=%s/%s",
		containers.AzuriteAccountName, containers.AzuriteAccountKey, proxy.URL, containers.AzuriteAccountName,
	)

	fieldEncryptor := providertest.NewFieldEncryptor(t)
	provider := New(Deps{Repositories: Repositories{settings: nil}, FieldEncryptor: fieldEncryptor})
	provider.uploadRetry = io_utils.TransferRetryPolicySpec{MaxAttemptsWithoutProgress: 0, BaseDelay: time.Millisecond}

	refusedContainer := &blobContainer{
		provider: provider,
		settings: Settings{
			AuthMethod:       AuthMethodConnectionString,
			ConnectionString: providertest.Encrypt(t, fieldEncryptor, connectionString),
			ContainerName:    containerName,
		},
	}

	return refusedContainer, client, containerName, proxy
}

// uncheckedAzureRefusal answers 507, a server error the Azure SDK does not retry itself, so every
// refusal reaches the upload's own repeats at once and the test does not wait out the SDK's pauses.
func uncheckedAzureRefusal(isRefusable func(r *http.Request) bool, refusalCount int) providertest.RefusingProxySpec {
	return providertest.RefusingProxySpec{
		TargetURL:    "",
		IsRefusable:  isRefusable,
		RefusalCount: refusalCount,
		Status:       http.StatusInsufficientStorage,
		ContentType:  "application/xml",
		Body:         azureErrorBody("InsufficientStorage"),
	}
}

func isCommitBlockList(r *http.Request) bool {
	return r.Method == http.MethodPut && r.URL.Query().Get("comp") == "blocklist"
}

func isBlobWrite(blobName string) func(r *http.Request) bool {
	return func(r *http.Request) bool {
		return r.Method == http.MethodPut && !r.URL.Query().Has("comp") && strings.HasSuffix(r.URL.Path, "/"+blobName)
	}
}

func isStageBlock(r *http.Request) bool {
	return r.Method == http.MethodPut && r.URL.Query().Get("comp") == "block"
}

func azureErrorBody(code string) string {
	return fmt.Sprintf(
		`<?xml version="1.0" encoding="utf-8"?><Error><Code>%s</Code><Message>%s</Message></Error>`,
		code,
		code,
	)
}

func downloadBlob(t *testing.T, client *azblob.Client, containerName, blobName string) []byte {
	t.Helper()

	response, err := client.DownloadStream(t.Context(), containerName, blobName, nil)
	require.NoError(t, err)

	defer func() {
		_ = response.Body.Close()
	}()

	content, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	return content
}

func TestTransport(t *testing.T) {
	t.Parallel()

	t.Run("takes the proxy from the environment", func(t *testing.T) {
		t.Parallel()

		transport := getAzureTransport(t)

		require.NotNil(t, transport.Proxy)
		assert.Equal(t,
			reflect.ValueOf(http.ProxyFromEnvironment).Pointer(),
			reflect.ValueOf(transport.Proxy).Pointer(),
		)
	})

	t.Run("tunnels through the proxy and rejects an untrusted certificate", func(t *testing.T) {
		t.Parallel()

		exchange := test_utils.SendThroughTunnelProxy(t, getAzureTransport(t), "SharedKey account:signature")

		test_utils.AssertProxyReceivedOnlyTunnelRequest(t, exchange)
		test_utils.AssertCertificateRejected(t, exchange)
	})
}

func getAzureTransport(t *testing.T) *http.Transport {
	t.Helper()

	testedContainer := &blobContainer{
		provider: New(
			Deps{Repositories: Repositories{settings: nil}, FieldEncryptor: providertest.NewFieldEncryptor(t)},
		),
		settings: Settings{},
	}
	options := testedContainer.buildClientOptions()

	client, isClient := options.Transport.(*http.Client)
	require.True(t, isClient, "Azure client options must carry an *http.Client")

	transport, isTransport := client.Transport.(*http.Transport)
	require.True(t, isTransport, "Azure HTTP client must use an *http.Transport")

	return transport
}
