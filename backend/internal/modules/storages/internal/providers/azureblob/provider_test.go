package azureblob

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/streaming"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	test_utils "databasus-backend/internal/lib/httpclienttest"
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
