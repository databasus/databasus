package azureblob

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/google/uuid"

	io_utils "databasus-backend/internal/lib/iox"
)

// blobContainer is one Azure Blob storage while a call runs: the settings it was called with and the
// provider that keeps the connections.
type blobContainer struct {
	provider *Provider
	settings Settings
}

const (
	azureConnectTimeout        = 30 * time.Second
	azureResponseTimeout       = 30 * time.Second
	azureIdleConnTimeout       = 90 * time.Second
	azureTLSHandshakeTimeout   = 30 * time.Second
	azureDeleteTimeout         = 30 * time.Second
	azureConnectionTestTimeout = 10 * time.Second

	// Go keeps two idle connections per host by default, which drops connections as soon as more
	// than two writes to one account run at once.
	azureMaxIdleConnsPerHost = 16

	// Chunk size for block blob uploads - 16MB provides good balance between
	// memory usage and upload efficiency. This creates backpressure to pg_dump
	// by only reading one chunk at a time and waiting for Azure to confirm receipt.
	azureChunkSize = 16 * 1024 * 1024
)

type readSeekCloser struct {
	*bytes.Reader
}

func (r *readSeekCloser) Close() error {
	return nil
}

func (c *blobContainer) saveFile(ctx context.Context, logger *slog.Logger, fileName string, file io.Reader) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("upload cancelled before start: %w", ctx.Err())
	default:
	}

	client, err := c.getClient()
	if err != nil {
		logger.ErrorContext(ctx, "failed to build the azure blob client",
			slog.Any("container", c.settings.ContainerName), slog.Any("error", err))

		return err
	}

	startedAt := time.Now().UTC()

	logger.DebugContext(ctx, "saving file to azure blob storage",
		slog.Any("file_name", fileName), slog.Any("container", c.settings.ContainerName))

	blobName := c.buildBlobName(fileName)
	blockBlobClient := client.ServiceClient().
		NewContainerClient(c.settings.ContainerName).
		NewBlockBlobClient(blobName)

	var blockIDs []string
	blockNumber := 0
	buf := make([]byte, azureChunkSize)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("upload cancelled: %w", ctx.Err())
		default:
		}

		n, readErr := io.ReadFull(file, buf)

		if n == 0 && readErr == io.EOF {
			break
		}

		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return fmt.Errorf("read error: %w", readErr)
		}

		blockID := base64.StdEncoding.EncodeToString(fmt.Appendf(nil, "%06d", blockNumber))

		_, err := blockBlobClient.StageBlock(
			ctx,
			blockID,
			&readSeekCloser{bytes.NewReader(buf[:n])},
			nil,
		)
		if err != nil {
			select {
			case <-ctx.Done():
				return fmt.Errorf("upload cancelled: %w", ctx.Err())
			default:
				return fmt.Errorf("failed to stage block %d: %w", blockNumber, err)
			}
		}

		blockIDs = append(blockIDs, blockID)
		blockNumber++

		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
	}

	if len(blockIDs) == 0 {
		_, err = client.UploadStream(
			ctx,
			c.settings.ContainerName,
			blobName,
			bytes.NewReader([]byte{}),
			nil,
		)
		if err != nil {
			logger.ErrorContext(ctx, "failed to save empty file to azure blob storage",
				slog.Any("file_name", fileName), slog.Any("error", err))

			return fmt.Errorf("failed to upload empty blob: %w", err)
		}

		logger.DebugContext(ctx, "saved empty file to azure blob storage", slog.Any("file_name", fileName))

		return nil
	}

	_, err = blockBlobClient.CommitBlockList(ctx, blockIDs, &blockblob.CommitBlockListOptions{})
	if err != nil {
		logger.ErrorContext(ctx, "failed to save file to azure blob storage",
			slog.Any("file_name", fileName), slog.Any("error", err))

		return fmt.Errorf("failed to commit block list: %w", err)
	}

	logger.DebugContext(ctx, "saved file to azure blob storage",
		slog.String("file_name", fileName),
		slog.Int("block_count", len(blockIDs)),
		slog.Duration("duration", time.Since(startedAt)))

	return nil
}

func (c *blobContainer) openFile(ctx context.Context, logger *slog.Logger, fileName string) (io.ReadCloser, error) {
	client, err := c.getClient()
	if err != nil {
		return nil, err
	}

	blobName := c.buildBlobName(fileName)

	properties, err := client.ServiceClient().
		NewContainerClient(c.settings.ContainerName).
		NewBlobClient(blobName).
		GetProperties(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to stat blob in Azure: %w", err)
	}

	return io_utils.NewResumingReader(io_utils.ResumingReaderSpec{
		StreamCtx:  ctx,
		Logger:     logger,
		FileName:   fileName,
		TotalBytes: getBlobSizeOrUnknown(properties.ContentLength),
		OpenAtOffset: func(attemptCtx context.Context, offsetBytes int64) (io.ReadCloser, error) {
			return c.downloadBlobFromOffset(attemptCtx, client, blobName, offsetBytes)
		},
		IsRetryableError: isRetryableAzureError,
	}), nil
}

func isRetryableAzureError(err error) bool {
	if responseError, isResponseError := errors.AsType[*azcore.ResponseError](err); isResponseError {
		return responseError.StatusCode == http.StatusRequestTimeout ||
			responseError.StatusCode == http.StatusTooManyRequests ||
			responseError.StatusCode >= http.StatusInternalServerError
	}

	return true
}

func getBlobSizeOrUnknown(contentLength *int64) int64 {
	if contentLength == nil {
		return io_utils.UnknownTotalBytes
	}

	return *contentLength
}

func (c *blobContainer) deleteFile(ctx context.Context, logger *slog.Logger, fileName string) error {
	client, err := c.getClient()
	if err != nil {
		return err
	}

	blobName := c.buildBlobName(fileName)

	// Deletes run from cleanup paths whose caller context is often already cancelled, so the
	// operation outlives a cancellation under its own deadline and keeps the caller's values.
	deleteCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), azureDeleteTimeout)
	defer cancel()

	if err := c.discardUncommittedBlocks(deleteCtx, client, blobName); err != nil {
		return err
	}

	_, err = client.DeleteBlob(
		deleteCtx,
		c.settings.ContainerName,
		blobName,
		nil,
	)
	if err != nil {
		if isAzureNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to delete blob from Azure: %w", err)
	}

	logger.DebugContext(ctx, "deleted file from azure blob storage", slog.Any("file_name", fileName))

	return nil
}

func (c *blobContainer) testConnection(ctx context.Context) error {
	client, err := c.getClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), azureConnectionTestTimeout)
	defer cancel()

	containerClient := client.ServiceClient().NewContainerClient(c.settings.ContainerName)
	_, err = containerClient.GetProperties(ctx, nil)
	if err != nil {
		if isAzureNotFound(err) {
			return fmt.Errorf("container '%s' does not exist", c.settings.ContainerName)
		}

		if errors.Is(err, context.DeadlineExceeded) {
			return errors.New("failed to connect to Azure Blob Storage. Please check params")
		}
		return fmt.Errorf("failed to connect to Azure Blob Storage: %w", err)
	}

	testBlobName := c.buildBlobName(uuid.New().String() + "-test")
	testData := []byte("test connection")

	_, err = client.UploadStream(
		ctx,
		c.settings.ContainerName,
		testBlobName,
		bytes.NewReader(testData),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to upload test blob to Azure: %w", err)
	}

	_, err = client.DeleteBlob(
		ctx,
		c.settings.ContainerName,
		testBlobName,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to delete test blob from Azure: %w", err)
	}

	return nil
}

// An interrupted upload leaves staged blocks that no blob points at, and Azure
// keeps billing for them for a week. There is no API that drops them, so cleanup
// commits an empty list, which turns them into a zero-length blob the delete then
// removes. Existence is the guard rather than the committed block list, because a
// blob written with a single Put Blob has no committed blocks and committing an
// empty list over it would truncate real content.
//
// This assumes Azure answers Get Block List for a blob that was never committed.
// Nothing here can detect it if Azure ever answers 404 instead.
func (c *blobContainer) discardUncommittedBlocks(
	ctx context.Context,
	client *azblob.Client,
	blobName string,
) error {
	containerClient := client.ServiceClient().NewContainerClient(c.settings.ContainerName)

	_, err := containerClient.NewBlobClient(blobName).GetProperties(ctx, nil)
	if err == nil {
		return nil
	}

	if !isAzureNotFound(err) {
		return fmt.Errorf("failed to inspect azure blob: %w", err)
	}

	blockBlobClient := containerClient.NewBlockBlobClient(blobName)

	blocks, err := blockBlobClient.GetBlockList(ctx, blockblob.BlockListTypeAll, nil)
	if err != nil {
		if isAzureNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to list azure blocks: %w", err)
	}

	if len(blocks.UncommittedBlocks) == 0 {
		return nil
	}

	if _, err := blockBlobClient.CommitBlockList(ctx, nil, nil); err != nil {
		return fmt.Errorf("failed to discard uncommitted azure blocks: %w", err)
	}

	return nil
}

func isAzureNotFound(err error) bool {
	var responseError *azcore.ResponseError

	return errors.As(err, &responseError) && responseError.StatusCode == http.StatusNotFound
}

func (c *blobContainer) buildBlobName(fileName string) string {
	if c.settings.Prefix == "" {
		return fileName
	}

	prefix := c.settings.Prefix
	prefix = strings.TrimPrefix(prefix, "/")

	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	return prefix + fileName
}

func (c *blobContainer) getClient() (*azblob.Client, error) {
	var client *azblob.Client
	var err error

	clientOptions := c.buildClientOptions()

	switch c.settings.AuthMethod {
	case AuthMethodConnectionString:
		connectionString, decryptErr := c.provider.fieldEncryptor.Decrypt(c.settings.ConnectionString)
		if decryptErr != nil {
			return nil, fmt.Errorf("failed to decrypt Azure connection string: %w", decryptErr)
		}

		client, err = azblob.NewClientFromConnectionString(connectionString, clientOptions)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create Azure Blob client from connection string: %w",
				err,
			)
		}
	case AuthMethodAccountKey:
		accountKey, decryptErr := c.provider.fieldEncryptor.Decrypt(c.settings.AccountKey)
		if decryptErr != nil {
			return nil, fmt.Errorf("failed to decrypt Azure account key: %w", decryptErr)
		}

		accountURL := c.buildAccountURL()
		credential, credErr := azblob.NewSharedKeyCredential(c.settings.AccountName, accountKey)
		if credErr != nil {
			return nil, fmt.Errorf("failed to create Azure shared key credential: %w", credErr)
		}

		client, err = azblob.NewClientWithSharedKeyCredential(accountURL, credential, clientOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure Blob client with shared key: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported auth method: %s", c.settings.AuthMethod)
	}

	return client, nil
}

func (c *blobContainer) buildClientOptions() *azblob.ClientOptions {
	return &azblob.ClientOptions{
		Transport: c.provider.httpClient,
		Retry: policy.RetryOptions{
			MaxRetries: 0,
		},
	}
}

func (c *blobContainer) buildAccountURL() string {
	if c.settings.Endpoint != "" {
		endpoint := c.settings.Endpoint
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			endpoint = "https://" + endpoint
		}
		return endpoint
	}

	return fmt.Sprintf("https://%s.blob.core.windows.net/", c.settings.AccountName)
}

func (c *blobContainer) downloadBlobFromOffset(
	attemptCtx context.Context,
	client *azblob.Client,
	blobName string,
	offsetBytes int64,
) (io.ReadCloser, error) {
	response, err := client.DownloadStream(
		attemptCtx,
		c.settings.ContainerName,
		blobName,
		&azblob.DownloadStreamOptions{Range: blob.HTTPRange{Offset: offsetBytes}},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to download blob from Azure: %w", err)
	}

	if offsetBytes > 0 {
		contentRange := ""
		if response.ContentRange != nil {
			contentRange = *response.ContentRange
		}

		if err := io_utils.VerifyContentRangeStart(contentRange, offsetBytes); err != nil {
			_ = response.Body.Close()

			return nil, err
		}
	}

	return response.Body, nil
}
