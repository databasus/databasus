package googledrive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	io_utils "databasus-backend/internal/lib/iox"
	"databasus-backend/internal/modules/storages/internal/providers"
)

// driveClient is one Google Drive storage while a call runs. Its settings are a copy, so a token it
// refreshes stays in memory for the rest of the call and reaches no other call.
type driveClient struct {
	provider *Provider
	settings Settings
}

const (
	gdConnectTimeout        = 30 * time.Second
	gdResponseTimeout       = 30 * time.Second
	gdIdleConnTimeout       = 90 * time.Second
	gdTLSHandshakeTimeout   = 30 * time.Second
	gdDeleteTimeout         = 30 * time.Second
	gdConnectionTestTimeout = 30 * time.Second

	// Chunk size for Google Drive resumable uploads - 16MB provides good balance
	// between memory usage and upload efficiency. Google Drive requires chunks
	// to be multiples of 256KB for resumable uploads.
	gdChunkSize = 16 * 1024 * 1024
)

func (c *driveClient) saveFile(ctx context.Context, logger *slog.Logger, fileName string, file io.Reader) error {
	return c.withRetryOnAuth(ctx, logger, func(driveService *drive.Service) error {
		folderID, err := c.ensureBackupsFolderExists(ctx, driveService)
		if err != nil {
			return fmt.Errorf("failed to create/find backups folder: %w", err)
		}

		_ = c.deleteByName(ctx, driveService, fileName, folderID)

		fileMeta := &drive.File{
			Name:    fileName,
			Parents: []string{folderID},
		}

		backpressureReader := &backpressureReader{
			reader:    file,
			ctx:       ctx,
			chunkSize: gdChunkSize,
			buf:       make([]byte, gdChunkSize),
		}

		_, err = driveService.Files.Create(fileMeta).
			Media(backpressureReader, googleapi.ChunkSize(gdChunkSize)).
			Context(ctx).
			Do()
		if err != nil {
			select {
			case <-ctx.Done():
				return fmt.Errorf("upload cancelled: %w", ctx.Err())
			default:
			}
			return fmt.Errorf("failed to upload file to Google Drive: %w", err)
		}

		logger.InfoContext(
			ctx,
			"file uploaded to Google Drive",
			slog.Any("name", fileName),
			slog.Any("folder", "databasus_backups"),
		)
		return nil
	})
}

type backpressureReader struct {
	reader     io.Reader
	ctx        context.Context
	chunkSize  int
	buf        []byte
	bufStart   int
	bufEnd     int
	totalBytes int64
	chunkCount int
}

func (r *backpressureReader) Read(p []byte) (n int, err error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
	}

	if r.bufStart >= r.bufEnd {
		r.chunkCount++

		bytesRead, readErr := io.ReadFull(r.reader, r.buf)
		if bytesRead > 0 {
			r.bufStart = 0
			r.bufEnd = bytesRead
		}

		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return 0, readErr
		}

		if bytesRead == 0 && readErr == io.EOF {
			return 0, io.EOF
		}
	}

	n = copy(p, r.buf[r.bufStart:r.bufEnd])
	r.bufStart += n
	r.totalBytes += int64(n)

	if r.bufStart >= r.bufEnd {
		select {
		case <-r.ctx.Done():
			return n, r.ctx.Err()
		default:
		}
	}

	return n, nil
}

func (c *driveClient) openFile(ctx context.Context, logger *slog.Logger, fileName string) (io.ReadCloser, error) {
	var driveFileID string
	var totalBytes int64

	err := c.withRetryOnAuth(ctx, logger,
		func(driveService *drive.Service) error {
			folderID, err := c.findBackupsFolder(ctx, driveService)
			if err != nil {
				return fmt.Errorf("failed to find backups folder: %w", err)
			}

			driveFileID, err = c.lookupFileID(ctx, driveService, fileName, folderID)
			if err != nil {
				return err
			}

			metadata, err := driveService.Files.Get(driveFileID).Fields("size").Context(ctx).Do()
			if err != nil {
				return fmt.Errorf("failed to stat file in Google Drive: %w", err)
			}

			totalBytes = metadata.Size

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	// Drive omits the size for files it did not store verbatim, and a zero would end the stream
	// before its first byte.
	if totalBytes <= 0 {
		totalBytes = io_utils.UnknownTotalBytes
	}

	return io_utils.NewResumingReader(io_utils.ResumingReaderSpec{
		StreamCtx:  ctx,
		Logger:     logger,
		FileName:   fileName,
		TotalBytes: totalBytes,
		OpenAtOffset: func(attemptCtx context.Context, offsetBytes int64) (io.ReadCloser, error) {
			return c.downloadFileFromOffset(attemptCtx, logger, driveFileID, offsetBytes)
		},
		IsRetryableError: isRetryableGoogleDriveError,
	}), nil
}

func isRetryableGoogleDriveError(err error) bool {
	var apiError *googleapi.Error
	if errors.As(err, &apiError) {
		return apiError.Code == http.StatusRequestTimeout ||
			apiError.Code == http.StatusTooManyRequests ||
			apiError.Code >= http.StatusInternalServerError
	}

	return true
}

func (c *driveClient) deleteFile(ctx context.Context, logger *slog.Logger, fileName string) error {
	// Deletes run from cleanup paths whose caller context is often already cancelled, so the
	// operation outlives a cancellation under its own deadline and keeps the caller's values.
	deleteCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), gdDeleteTimeout)
	defer cancel()

	if err := c.withRetryOnAuth(deleteCtx, logger, func(driveService *drive.Service) error {
		folderID, err := c.findBackupsFolder(ctx, driveService)
		if errors.Is(err, errBackupsFolderNotFound) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("failed to find backups folder: %w", err)
		}

		return c.deleteByName(deleteCtx, driveService, fileName, folderID)
	}); err != nil {
		logger.WarnContext(
			ctx,
			"failed to delete file from google drive",
			slog.Any("file_name", fileName),
			slog.Any("error", err),
		)

		return err
	}

	logger.DebugContext(ctx, "deleted file from google drive", slog.Any("file_name", fileName))

	return nil
}

// validateSettings skips the token check once the token is encrypted: only a token the request sent
// arrives in clear text, and a stored one passed the check when it was saved.
func validateSettings(settings Settings) error {
	switch {
	case settings.ClientID == "":
		return providers.NewInvalidSettingError("clientId", "is required")
	case settings.ClientSecret == "":
		return providers.NewInvalidSettingError("clientSecret", "is required")
	case settings.TokenJSON == "":
		return providers.NewInvalidSettingError("tokenJson", "is required")
	}

	if strings.HasPrefix(settings.TokenJSON, "enc:") {
		return nil
	}

	var token oauth2.Token
	if err := json.Unmarshal([]byte(settings.TokenJSON), &token); err != nil {
		return providers.NewInvalidSettingError("tokenJson", "is not a JSON OAuth token")
	}

	if token.RefreshToken == "" {
		return providers.NewInvalidSettingError("tokenJson", "must contain a refresh token for automatic token refresh")
	}

	return nil
}

func (c *driveClient) testConnection(ctx context.Context, connectionLogger *slog.Logger) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), gdConnectionTestTimeout)
	defer cancel()

	return c.withRetryOnAuth(ctx, connectionLogger, func(driveService *drive.Service) error {
		testFilename := "test-connection-" + uuid.New().String()
		testData := []byte("test")

		// Ensure the databasus_backups folder exists
		folderID, err := c.ensureBackupsFolderExists(ctx, driveService)
		if err != nil {
			return fmt.Errorf("failed to create/find backups folder: %w", err)
		}

		// Test write operation
		fileMeta := &drive.File{
			Name:    testFilename,
			Parents: []string{folderID},
		}
		file, err := driveService.Files.Create(fileMeta).
			Media(strings.NewReader(string(testData))).
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("failed to write test file to Google Drive: %w", err)
		}

		// Test read operation
		resp, err := driveService.Files.Get(file.Id).Context(ctx).Download()
		if err != nil {
			// Clean up test file before returning error
			_ = driveService.Files.Delete(file.Id).Context(ctx).Do()
			return fmt.Errorf("failed to read test file from Google Drive: %w", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				connectionLogger.WarnContext(ctx, "failed to close google drive response body", slog.Any("error", err))
			}
		}()

		readData, err := io.ReadAll(resp.Body)
		if err != nil {
			// Clean up test file before returning error
			_ = driveService.Files.Delete(file.Id).Context(ctx).Do()
			return fmt.Errorf("failed to read test file data: %w", err)
		}

		// Clean up test file
		if err := driveService.Files.Delete(file.Id).Context(ctx).Do(); err != nil {
			return fmt.Errorf("failed to clean up test file: %w", err)
		}

		// Verify data matches
		if string(readData) != string(testData) {
			return fmt.Errorf(
				"test file data mismatch: expected %q, got %q",
				string(testData),
				string(readData),
			)
		}

		return nil
	})
}

// withRetryOnAuth runs fn and, after an authentication error, refreshes the token once and runs
// fn again. Refresh events go through the logger, whose layer redacts secrets; no part of a token is
// ever logged.
func (c *driveClient) withRetryOnAuth(
	ctx context.Context,
	logger *slog.Logger,
	fn func(*drive.Service) error,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	driveService, err := c.getDriveService(ctx)
	if err != nil {
		return err
	}

	err = fn(driveService)
	if err != nil && c.isAuthError(err) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		logger.InfoContext(ctx, "google drive auth error, refreshing token", slog.Any("error", err))

		if refreshErr := c.refreshToken(ctx); refreshErr != nil {
			logger.WarnContext(ctx, "google drive token refresh failed", slog.Any("error", refreshErr))

			if strings.Contains(refreshErr.Error(), "invalid_grant") ||
				strings.Contains(refreshErr.Error(), "refresh token") {
				return fmt.Errorf(
					"google drive refresh token has expired. Please re-authenticate and update your token configuration. Original error: %w. Refresh error: %w",
					err,
					refreshErr,
				)
			}

			return fmt.Errorf("failed to refresh token after auth error: %w", refreshErr)
		}

		logger.InfoContext(ctx, "google drive token refreshed, retrying operation")

		driveService, err = c.getDriveService(ctx)
		if err != nil {
			return fmt.Errorf("failed to create service after token refresh: %w", err)
		}

		err = fn(driveService)
		if err != nil {
			logger.WarnContext(ctx, "google drive operation failed after token refresh", slog.Any("error", err))
		}
	}

	return err
}

// isAuthError checks if the error is a 401 authentication error
func (c *driveClient) isAuthError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "Invalid Credentials") ||
		strings.Contains(errStr, "authError") ||
		strings.Contains(errStr, "invalid authentication credentials")
}

// refreshToken takes ctx so the token request ends with the operation that needed it, and so a
// test can route it through oauth2.HTTPClient.
func (c *driveClient) refreshToken(ctx context.Context) error {
	if err := validateSettings(c.settings); err != nil {
		return err
	}

	clientSecret, err := c.provider.fieldEncryptor.Decrypt(c.settings.ClientSecret)
	if err != nil {
		return fmt.Errorf("failed to decrypt Google Drive client secret: %w", err)
	}

	tokenJSON, err := c.provider.fieldEncryptor.Decrypt(c.settings.TokenJSON)
	if err != nil {
		return fmt.Errorf("failed to decrypt Google Drive token JSON: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal([]byte(tokenJSON), &token); err != nil {
		return fmt.Errorf("invalid token JSON: %w", err)
	}

	if token.RefreshToken == "" {
		return fmt.Errorf("no refresh token available in stored token")
	}

	cfg := &oauth2.Config{
		ClientID:     c.settings.ClientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{"https://www.googleapis.com/auth/drive.file"},
	}

	// The token source refreshes only an expired token, so expiring it guarantees the refresh.
	token.Expiry = time.Now().UTC().Add(-time.Hour)

	newToken, err := cfg.TokenSource(ctx, &token).Token()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	if newToken.AccessToken == token.AccessToken {
		return fmt.Errorf(
			"token refresh did not return a new access token - this indicates the refresh token may be invalid",
		)
	}

	// Google returns a refresh token only on the first authorization, so the stored one is kept.
	if newToken.RefreshToken == "" {
		newToken.RefreshToken = token.RefreshToken
	}

	// Kept in memory for this call only: no operation writes a refreshed token to the database.
	newTokenJSON, err := json.Marshal(newToken)
	if err != nil {
		return fmt.Errorf("failed to marshal refreshed token: %w", err)
	}

	c.settings.TokenJSON = string(newTokenJSON)

	return nil
}

func (c *driveClient) getDriveService(ctx context.Context) (*drive.Service, error) {
	if err := validateSettings(c.settings); err != nil {
		return nil, err
	}

	clientSecret, err := c.provider.fieldEncryptor.Decrypt(c.settings.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt Google Drive client secret: %w", err)
	}

	tokenJSON, err := c.provider.fieldEncryptor.Decrypt(c.settings.TokenJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt Google Drive token JSON: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal([]byte(tokenJSON), &token); err != nil {
		return nil, fmt.Errorf("invalid token JSON: %w", err)
	}

	cfg := &oauth2.Config{
		ClientID:     c.settings.ClientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{"https://www.googleapis.com/auth/drive.file"},
	}

	tokenSource := cfg.TokenSource(ctx, &token)

	currentToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get current token: %w", err)
	}

	validatedTokenSource := oauth2.StaticTokenSource(currentToken)

	httpClient := c.buildHTTPClient(validatedTokenSource)

	driveService, err := drive.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("unable to create Drive client: %w", err)
	}

	return driveService, nil
}

func (c *driveClient) buildHTTPClient(tokenSource oauth2.TokenSource) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: gdConnectTimeout,
		}).DialContext,
		TLSHandshakeTimeout:   gdTLSHandshakeTimeout,
		ResponseHeaderTimeout: gdResponseTimeout,
		IdleConnTimeout:       gdIdleConnTimeout,
	}

	return &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
			Base:   transport,
		},
	}
}

func (c *driveClient) lookupFileID(
	ctx context.Context,
	driveService *drive.Service,
	name string,
	folderID string,
) (string, error) {
	query := fmt.Sprintf(
		"name = '%s' and trashed = false and '%s' in parents",
		escapeForQuery(name),
		folderID,
	)

	results, err := driveService.Files.List().
		Q(query).
		Fields("files(id)").
		PageSize(1).
		Context(ctx).
		Do()
	if err != nil {
		return "", fmt.Errorf("file lookup failed: %w", err)
	}

	if len(results.Files) == 0 {
		return "", fmt.Errorf("file %q not found in Google Drive backups folder", name)
	}

	return results.Files[0].Id, nil
}

func (c *driveClient) deleteByName(
	ctx context.Context,
	driveService *drive.Service,
	name string,
	folderID string,
) error {
	query := fmt.Sprintf(
		"name = '%s' and trashed = false and '%s' in parents",
		escapeForQuery(name),
		folderID,
	)

	err := driveService.
		Files.
		List().
		Q(query).
		Fields("files(id)").
		Pages(ctx, func(p *drive.FileList) error {
			for _, file := range p.Files {
				if err := driveService.Files.Delete(file.Id).Context(ctx).Do(); err != nil {
					return err
				}
			}

			return nil
		})
	if err != nil {
		return fmt.Errorf("failed to delete %q: %w", name, err)
	}

	return nil
}

func escapeForQuery(s string) string {
	return strings.ReplaceAll(s, `'`, `\'`)
}

// ensureBackupsFolderExists creates the databasus_backups folder if it doesn't exist
func (c *driveClient) ensureBackupsFolderExists(
	ctx context.Context,
	driveService *drive.Service,
) (string, error) {
	folderID, err := c.findBackupsFolder(ctx, driveService)
	if err == nil {
		return folderID, nil
	}

	// Folder doesn't exist, create it
	folderMeta := &drive.File{
		Name:     "databasus_backups",
		MimeType: "application/vnd.google-apps.folder",
	}

	folder, err := driveService.Files.Create(folderMeta).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to create databasus_backups folder: %w", err)
	}

	return folder.Id, nil
}

func (c *driveClient) findBackupsFolder(ctx context.Context, driveService *drive.Service) (string, error) {
	query := "name = 'databasus_backups' and mimeType = 'application/vnd.google-apps.folder' and trashed = false"

	results, err := driveService.Files.List().
		Q(query).
		Fields("files(id)").
		PageSize(1).
		Context(ctx).
		Do()
	if err != nil {
		return "", fmt.Errorf("failed to search for backups folder: %w", err)
	}

	if len(results.Files) == 0 {
		return "", errBackupsFolderNotFound
	}

	return results.Files[0].Id, nil
}

func (c *driveClient) downloadFileFromOffset(
	attemptCtx context.Context,
	logger *slog.Logger,
	driveFileID string,
	offsetBytes int64,
) (io.ReadCloser, error) {
	var body io.ReadCloser

	err := c.withRetryOnAuth(attemptCtx, logger,
		func(driveService *drive.Service) error {
			downloadCall := driveService.Files.Get(driveFileID).Context(attemptCtx)
			if offsetBytes > 0 {
				downloadCall.Header().Set("Range", fmt.Sprintf("bytes=%d-", offsetBytes))
			}

			// The body is the stream handed to the caller, who closes it.
			response, err := downloadCall.Download()
			if err != nil {
				return fmt.Errorf("failed to download file from Google Drive: %w", err)
			}

			if offsetBytes > 0 {
				if err := io_utils.VerifyContentRangeStart(
					response.Header.Get("Content-Range"),
					offsetBytes,
				); err != nil {
					_ = response.Body.Close()

					return err
				}
			}

			body = response.Body

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return body, nil
}
