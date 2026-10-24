package google_drive_storage

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

	"databasus-backend/internal/util/encryption"
	io_utils "databasus-backend/internal/util/io"
	"databasus-backend/internal/util/logger"
)

const (
	gdConnectTimeout      = 30 * time.Second
	gdResponseTimeout     = 30 * time.Second
	gdIdleConnTimeout     = 90 * time.Second
	gdTLSHandshakeTimeout = 30 * time.Second
	gdDeleteTimeout       = 30 * time.Second

	// Chunk size for Google Drive resumable uploads - 16MB provides good balance
	// between memory usage and upload efficiency. Google Drive requires chunks
	// to be multiples of 256KB for resumable uploads.
	gdChunkSize = 16 * 1024 * 1024
)

type GoogleDriveStorage struct {
	StorageID    uuid.UUID `json:"storageId"    gorm:"primaryKey;type:uuid;column:storage_id"`
	ClientID     string    `json:"clientId"     gorm:"not null;type:text;column:client_id"`
	ClientSecret string    `json:"clientSecret" gorm:"not null;type:text;column:client_secret"`
	TokenJSON    string    `json:"tokenJson"    gorm:"not null;type:text;column:token_json"`
}

func (s *GoogleDriveStorage) TableName() string {
	return "google_drive_storages"
}

func (s *GoogleDriveStorage) SaveFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileName string,
	file io.Reader,
) error {
	return s.withRetryOnAuth(ctx, logger, encryptor, func(driveService *drive.Service) error {
		folderID, err := s.ensureBackupsFolderExists(ctx, driveService)
		if err != nil {
			return fmt.Errorf("failed to create/find backups folder: %w", err)
		}

		_ = s.deleteByName(ctx, driveService, fileName, folderID)

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
			"name",
			fileName,
			"folder",
			"databasus_backups",
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

func (s *GoogleDriveStorage) GetFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileName string,
) (io.ReadCloser, error) {
	var driveFileID string
	var totalBytes int64

	err := s.withRetryOnAuth(
		ctx,
		logger,
		encryptor,
		func(driveService *drive.Service) error {
			folderID, err := s.findBackupsFolder(driveService)
			if err != nil {
				return fmt.Errorf("failed to find backups folder: %w", err)
			}

			driveFileID, err = s.lookupFileID(driveService, fileName, folderID)
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
			return s.downloadFileFromOffset(attemptCtx, logger, encryptor, driveFileID, offsetBytes)
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

func (s *GoogleDriveStorage) DeleteFile(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	fileName string,
) error {
	// Deletes run from cleanup paths whose caller context is often already cancelled, so the
	// operation carries its own deadline; the caller's ctx stays for log correlation only.
	deleteCtx, cancel := context.WithTimeout(context.Background(), gdDeleteTimeout)
	defer cancel()

	if err := s.withRetryOnAuth(deleteCtx, logger, encryptor, func(driveService *drive.Service) error {
		folderID, err := s.findBackupsFolder(driveService)
		if errors.Is(err, errBackupsFolderNotFound) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("failed to find backups folder: %w", err)
		}

		return s.deleteByName(deleteCtx, driveService, fileName, folderID)
	}); err != nil {
		logger.WarnContext(ctx, "failed to delete file from google drive", "file_name", fileName, "error", err)

		return err
	}

	logger.DebugContext(ctx, "deleted file from google drive", "file_name", fileName)

	return nil
}

func (s *GoogleDriveStorage) Validate(encryptor encryption.FieldEncryptor) error {
	switch {
	case s.ClientID == "":
		return errors.New("client ID is required")
	case s.ClientSecret == "":
		return errors.New("client secret is required")
	case s.TokenJSON == "":
		return errors.New("token JSON is required")
	}

	// Skip JSON validation if token is already encrypted
	if strings.HasPrefix(s.TokenJSON, "enc:") {
		return nil
	}

	// Validate that the token JSON contains a refresh token
	var token oauth2.Token
	if err := json.Unmarshal([]byte(s.TokenJSON), &token); err != nil {
		return fmt.Errorf("invalid token JSON format: %w", err)
	}

	if token.RefreshToken == "" {
		return errors.New("token JSON must contain a refresh token for automatic token refresh")
	}

	return nil
}

func (s *GoogleDriveStorage) TestConnection(encryptor encryption.FieldEncryptor) error {
	ctx := context.Background()
	connectionLogger := logger.GetLogger()

	return s.withRetryOnAuth(ctx, connectionLogger, encryptor, func(driveService *drive.Service) error {
		testFilename := "test-connection-" + uuid.New().String()
		testData := []byte("test")

		// Ensure the databasus_backups folder exists
		folderID, err := s.ensureBackupsFolderExists(ctx, driveService)
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
		resp, err := driveService.Files.Get(file.Id).Download()
		if err != nil {
			// Clean up test file before returning error
			_ = driveService.Files.Delete(file.Id).Context(ctx).Do()
			return fmt.Errorf("failed to read test file from Google Drive: %w", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				connectionLogger.WarnContext(ctx, "failed to close google drive response body", "error", err)
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

func (s *GoogleDriveStorage) HideSensitiveData() {
	s.ClientSecret = ""
	s.TokenJSON = ""
}

func (s *GoogleDriveStorage) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	var err error

	if s.ClientSecret != "" {
		s.ClientSecret, err = encryptor.Encrypt(s.ClientSecret)
		if err != nil {
			return fmt.Errorf("failed to encrypt Google Drive client secret: %w", err)
		}
	}

	if s.TokenJSON != "" {
		s.TokenJSON, err = encryptor.Encrypt(s.TokenJSON)
		if err != nil {
			return fmt.Errorf("failed to encrypt Google Drive token JSON: %w", err)
		}
	}

	return nil
}

func (s *GoogleDriveStorage) Update(incoming *GoogleDriveStorage) {
	s.ClientID = incoming.ClientID

	if incoming.ClientSecret != "" {
		s.ClientSecret = incoming.ClientSecret
	}

	if incoming.TokenJSON != "" {
		s.TokenJSON = incoming.TokenJSON
	}
}

// withRetryOnAuth runs fn and, after an authentication error, refreshes the token once and runs
// fn again. Refresh events go through the logger, whose layer redacts secrets; no part of a token is
// ever logged.
func (s *GoogleDriveStorage) withRetryOnAuth(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
	fn func(*drive.Service) error,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	driveService, err := s.getDriveService(encryptor)
	if err != nil {
		return err
	}

	err = fn(driveService)
	if err != nil && s.isAuthError(err) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		logger.InfoContext(ctx, "google drive auth error, refreshing token", "error", err)

		if refreshErr := s.refreshToken(ctx, encryptor); refreshErr != nil {
			logger.WarnContext(ctx, "google drive token refresh failed", "error", refreshErr)

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

		driveService, err = s.getDriveService(encryptor)
		if err != nil {
			return fmt.Errorf("failed to create service after token refresh: %w", err)
		}

		err = fn(driveService)
		if err != nil {
			logger.WarnContext(ctx, "google drive operation failed after token refresh", "error", err)
		}
	}

	return err
}

// isAuthError checks if the error is a 401 authentication error
func (s *GoogleDriveStorage) isAuthError(err error) bool {
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
func (s *GoogleDriveStorage) refreshToken(ctx context.Context, encryptor encryption.FieldEncryptor) error {
	if err := s.Validate(encryptor); err != nil {
		return err
	}

	clientSecret, err := encryptor.Decrypt(s.ClientSecret)
	if err != nil {
		return fmt.Errorf("failed to decrypt Google Drive client secret: %w", err)
	}

	tokenJSON, err := encryptor.Decrypt(s.TokenJSON)
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
		ClientID:     s.ClientID,
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

	// Kept as plaintext in memory; encryption happens on save.
	newTokenJSON, err := json.Marshal(newToken)
	if err != nil {
		return fmt.Errorf("failed to marshal refreshed token: %w", err)
	}

	s.TokenJSON = string(newTokenJSON)

	return nil
}

func (s *GoogleDriveStorage) getDriveService(
	encryptor encryption.FieldEncryptor,
) (*drive.Service, error) {
	if err := s.Validate(encryptor); err != nil {
		return nil, err
	}

	clientSecret, err := encryptor.Decrypt(s.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt Google Drive client secret: %w", err)
	}

	tokenJSON, err := encryptor.Decrypt(s.TokenJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt Google Drive token JSON: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal([]byte(tokenJSON), &token); err != nil {
		return nil, fmt.Errorf("invalid token JSON: %w", err)
	}

	ctx := context.Background()

	cfg := &oauth2.Config{
		ClientID:     s.ClientID,
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

	httpClient := s.buildHTTPClient(validatedTokenSource)

	driveService, err := drive.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("unable to create Drive client: %w", err)
	}

	return driveService, nil
}

func (s *GoogleDriveStorage) buildHTTPClient(tokenSource oauth2.TokenSource) *http.Client {
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

func (s *GoogleDriveStorage) lookupFileID(
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
		Do()
	if err != nil {
		return "", fmt.Errorf("file lookup failed: %w", err)
	}

	if len(results.Files) == 0 {
		return "", fmt.Errorf("file %q not found in Google Drive backups folder", name)
	}

	return results.Files[0].Id, nil
}

func (s *GoogleDriveStorage) deleteByName(
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
func (s *GoogleDriveStorage) ensureBackupsFolderExists(
	ctx context.Context,
	driveService *drive.Service,
) (string, error) {
	folderID, err := s.findBackupsFolder(driveService)
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

func (s *GoogleDriveStorage) findBackupsFolder(driveService *drive.Service) (string, error) {
	query := "name = 'databasus_backups' and mimeType = 'application/vnd.google-apps.folder' and trashed = false"

	results, err := driveService.Files.List().
		Q(query).
		Fields("files(id)").
		PageSize(1).
		Do()
	if err != nil {
		return "", fmt.Errorf("failed to search for backups folder: %w", err)
	}

	if len(results.Files) == 0 {
		return "", errBackupsFolderNotFound
	}

	return results.Files[0].Id, nil
}

func (s *GoogleDriveStorage) downloadFileFromOffset(
	attemptCtx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
	driveFileID string,
	offsetBytes int64,
) (io.ReadCloser, error) {
	var body io.ReadCloser

	err := s.withRetryOnAuth(
		attemptCtx,
		logger,
		encryptor,
		func(driveService *drive.Service) error {
			downloadCall := driveService.Files.Get(driveFileID).Context(attemptCtx)
			if offsetBytes > 0 {
				downloadCall.Header().Set("Range", fmt.Sprintf("bytes=%d-", offsetBytes))
			}

			response, err := downloadCall.Download() //nolint:bodyclose
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
