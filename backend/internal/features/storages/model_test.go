package storages

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	storage_files "databasus-backend/internal/features/storages/files"
	azure_blob_storage "databasus-backend/internal/features/storages/models/azure_blob"
	ftp_storage "databasus-backend/internal/features/storages/models/ftp"
	local_storage "databasus-backend/internal/features/storages/models/local"
	nas_storage "databasus-backend/internal/features/storages/models/nas"
	rclone_storage "databasus-backend/internal/features/storages/models/rclone"
	s3_storage "databasus-backend/internal/features/storages/models/s3"
	sftp_storage "databasus-backend/internal/features/storages/models/sftp"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	db "databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/containers"
)

type S3Container struct {
	endpoint   string
	accessKey  string
	secretKey  string
	bucketName string
	region     string
}

func Test_Storage_BasicOperations(t *testing.T) {
	minio := containers.StartMinioContainer(t)
	azurite := containers.StartAzuriteContainer(t)
	samba := containers.StartSambaContainer(t)
	ftp := containers.StartFtpContainer(t)
	sftp := containers.StartSftpContainer(t)

	testFilePath, err := setupTestFile()
	require.NoError(t, err, "Failed to setup test file")
	defer os.Remove(testFilePath)

	s3WithStorageClass := buildTestStorage(t, StorageTypeS3, minio)
	s3WithStorageClass.S3Storage.S3StorageClass = s3_storage.S3StorageClassStandard

	azureWithConnectionString := buildTestStorage(t, StorageTypeAzureBlob, azurite)
	azureWithConnectionString.AzureBlobStorage = &azure_blob_storage.AzureBlobStorage{
		AuthMethod:       azure_blob_storage.AuthMethodConnectionString,
		ConnectionString: getAzuriteConnectionString(addressOf(azurite.Endpoint)),
		ContainerName:    azureWithConnectionString.AzureBlobStorage.ContainerName,
	}

	testCases := []struct {
		name    string
		storage StorageFileSaver
	}{
		{
			name:    "LocalStorage",
			storage: createTestStorageOfType(t, StorageTypeLocal, containers.ContainerHandle{}),
		},
		{
			name:    "S3Storage",
			storage: createTestStorageOfType(t, StorageTypeS3, minio),
		},
		{
			name:    "S3Storage_WithStorageClass",
			storage: saveTestStorage(t, s3WithStorageClass),
		},
		{
			name:    "NASStorage",
			storage: createTestStorageOfType(t, StorageTypeNAS, samba),
		},
		{
			name:    "AzureBlobStorage_AccountKey",
			storage: createTestStorageOfType(t, StorageTypeAzureBlob, azurite),
		},
		{
			name:    "AzureBlobStorage_ConnectionString",
			storage: saveTestStorage(t, azureWithConnectionString),
		},
		{
			name:    "FTPStorage",
			storage: createTestStorageOfType(t, StorageTypeFTP, ftp),
		},
		{
			name:    "SFTPStorage",
			storage: createTestStorageOfType(t, StorageTypeSFTP, sftp),
		},
		{
			name:    "RcloneStorage",
			storage: createTestStorageOfType(t, StorageTypeRclone, minio),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encryptor := encryption.GetFieldEncryptor()

			t.Run("Test_TestConnection_ConnectionSucceeds", func(t *testing.T) {
				err := tc.storage.TestConnection(encryptor)
				assert.NoError(t, err, "TestConnection should succeed")
			})

			t.Run("Test_TestValidation_ValidationSucceeds", func(t *testing.T) {
				err := tc.storage.Validate(encryptor)
				assert.NoError(t, err, "Validate should succeed")
			})

			t.Run("Test_TestSaveAndGetFile_ReturnsCorrectContent", func(t *testing.T) {
				fileData, err := os.ReadFile(testFilePath)
				require.NoError(t, err, "Should be able to read test file")

				fileID := uuid.New()

				err = tc.storage.SaveFile(
					t.Context(),
					encryptor,
					logger.GetLogger(),
					fileID.String(),
					bytes.NewReader(fileData),
				)
				require.NoError(t, err, "SaveFile should succeed")

				file, err := tc.storage.GetFile(t.Context(), encryptor, logger.GetLogger(), fileID.String())
				assert.NoError(t, err, "GetFile should succeed")
				defer file.Close()

				content, err := io.ReadAll(file)
				assert.NoError(t, err, "Should be able to read file")
				assert.Equal(t, fileData, content, "File content should match the original")
			})

			// Reads run through a ResumingReader that stops at the size the backend declared, so a
			// backend reporting the wrong length truncates the download. One inline byte of test
			// data is too small to catch that.
			t.Run("Test_SaveAndGetLargeFile_ReturnsCompleteContent", func(t *testing.T) {
				largeFileData := generateFileContent(256 * 1024)
				fileID := uuid.New()

				err := tc.storage.SaveFile(
					t.Context(),
					encryptor,
					logger.GetLogger(),
					fileID.String(),
					bytes.NewReader(largeFileData),
				)
				require.NoError(t, err, "SaveFile should succeed")

				t.Cleanup(func() {
					_ = tc.storage.DeleteFile(t.Context(), encryptor, logger.GetLogger(), fileID.String())
				})

				file, err := tc.storage.GetFile(
					t.Context(),
					encryptor,
					logger.GetLogger(),
					fileID.String(),
				)
				require.NoError(t, err, "GetFile should succeed")
				defer file.Close()

				content, err := io.ReadAll(file)
				require.NoError(t, err, "Should be able to read file")
				assert.Equal(t, largeFileData, content, "File content should match the original")
			})

			t.Run("Test_TestDeleteFile_RemovesFileFromDisk", func(t *testing.T) {
				fileData, err := os.ReadFile(testFilePath)
				require.NoError(t, err, "Should be able to read test file")

				fileID := uuid.New()
				err = tc.storage.SaveFile(
					t.Context(),
					encryptor,
					logger.GetLogger(),
					fileID.String(),
					bytes.NewReader(fileData),
				)
				require.NoError(t, err, "SaveFile should succeed")

				err = tc.storage.DeleteFile(t.Context(), encryptor, logger.GetLogger(), fileID.String())
				assert.NoError(t, err, "DeleteFile should succeed")

				file, err := tc.storage.GetFile(t.Context(), encryptor, logger.GetLogger(), fileID.String())
				assert.Error(t, err, "GetFile should fail for non-existent file")
				if file != nil {
					file.Close()
				}
			})

			t.Run("Test_TestDeleteNonExistentFile_DoesNotError", func(t *testing.T) {
				nonExistentID := uuid.New()
				err := tc.storage.DeleteFile(t.Context(), encryptor, logger.GetLogger(), nonExistentID.String())
				assert.NoError(t, err, "DeleteFile should not error for non-existent file")
			})
		})
	}
}

// The storage answers every packet 50 ms late, so each round trip a write pays shows up in its
// duration and in nothing else the runner's load can inflate as much. The check targets the two
// mistakes that keep WAL archiving from keeping up with a distant storage: a handshake per file and
// a round trip per block. The connection count catches the first exactly, while S3, Azure and
// Rclone reconnect for only one TCP round trip, too little for any timing to tell apart.
const (
	roundTripEgressDelay   = 50 * time.Millisecond
	reusedFileWriteCount   = 10
	segmentSizedFileBytes  = 16 * 1024 * 1024
	metadataSizedFileBytes = 1024
)

// A round trip per 32 KiB block, the buffer io.Copy streams with, costs a segment 512 round trips,
// about 25s. A streamed segment takes 0.5-1s on a developer machine, so a quarter of the slow path
// leaves the loaded CI runner several times the room it needs and still fails the slow path.
const segmentWriteLimit = 128 * roundTripEgressDelay

type remoteStorageTarget struct {
	name           string
	storageType    StorageType
	startContainer func(t *testing.T) containers.ContainerHandle
}

func getRemoteStorageTargets() []remoteStorageTarget {
	return []remoteStorageTarget{
		{"S3", StorageTypeS3, containers.StartMinioContainer},
		{"AzureBlob", StorageTypeAzureBlob, containers.StartAzuriteContainer},
		{"SFTP", StorageTypeSFTP, containers.StartSftpContainer},
		{"FTP", StorageTypeFTP, containers.StartFtpContainer},
		{"NAS", StorageTypeNAS, containers.StartSambaContainer},
		{"Rclone", StorageTypeRclone, containers.StartMinioContainer},
	}
}

func Test_WriteFile_WhenStorageAnswers50msLate_PaysNoHandshakePerFileOrRoundTripPerBlock(t *testing.T) {
	for _, target := range getRemoteStorageTargets() {
		t.Run(target.name, func(t *testing.T) {
			t.Parallel()

			container := target.startContainer(t)
			containers.AddEgressDelay(t, container, roundTripEgressDelay)
			t.Cleanup(func() { containers.ClearEgressShaping(t, container) })

			// FTP data connections still go straight to the container's passive ports, so only
			// control connections, one per session, cross the forwarder.
			forwarder := test_utils.StartCountingForwarder(t, addressOf(container.Endpoint))
			forwardedContainer := container
			forwardedContainer.Endpoint = containers.Endpoint{Host: forwarder.Host, Port: forwarder.Port}

			storage := createTestStorageOfType(t, target.storageType, forwardedContainer)
			writtenFiles := trackWrittenFiles(t, storage.ID)
			metadataContent := generateRandomContent(t, metadataSizedFileBytes)
			segmentContent := generateRandomContent(t, segmentSizedFileBytes)

			// The first write opens the session, and its handshake is not what the test measures.
			require.NoError(t, writtenFiles.writeUniqueFile(t.Context(), metadataContent))
			connectionsAfterFirstWrite := forwarder.GetAcceptedCount()

			for range reusedFileWriteCount {
				require.NoError(t, writtenFiles.writeUniqueFile(t.Context(), metadataContent))
			}

			assert.Equal(
				t,
				connectionsAfterFirstWrite,
				forwarder.GetAcceptedCount(),
				"%d files written one after another must reuse the connection the first write opened",
				reusedFileWriteCount,
			)

			startedAt := time.Now()
			require.NoError(t, writtenFiles.writeUniqueFile(t.Context(), segmentContent))
			segmentWriteDuration := time.Since(startedAt)

			assert.Less(
				t,
				segmentWriteDuration,
				segmentWriteLimit,
				"a %d MiB segment took %s, as long as paying a round trip for every four 32 KiB blocks",
				segmentSizedFileBytes/1024/1024,
				segmentWriteDuration,
			)

			t.Logf("segment written in %s", segmentWriteDuration)
		})
	}
}

// Every file a test writes, the failed attempts included, is handed to the deletion worker at the
// end of the test, the way a backup that never publishes its files hands them back.
type writtenFileTracker struct {
	storageID uuid.UUID

	mutex      sync.Mutex
	references []storage_files.StoredFileReference
}

func trackWrittenFiles(t *testing.T, storageID uuid.UUID) *writtenFileTracker {
	t.Helper()

	tracker := &writtenFileTracker{storageID: storageID}
	t.Cleanup(tracker.requestDeletionAndDrainFunc(t))

	return tracker
}

func (w *writtenFileTracker) write(ctx context.Context, fileName string, content []byte) error {
	reference := storage_files.StoredFileReference{StorageID: w.storageID, FileName: fileName}

	w.mutex.Lock()
	w.references = append(w.references, reference)
	w.mutex.Unlock()

	_, err := GetStorageFileStore().WriteFile(ctx, reference, bytes.NewReader(content))

	return err
}

func (w *writtenFileTracker) writeUniqueFile(ctx context.Context, content []byte) error {
	return w.write(ctx, "test-file-"+uuid.New().String(), content)
}

func (w *writtenFileTracker) requestDeletionAndDrainFunc(t *testing.T) func() {
	return func() {
		w.mutex.Lock()
		references := slices.Clone(w.references)
		w.references = nil
		w.mutex.Unlock()

		requestDeletionAndDrain(t, references)
	}
}

func requestDeletionAndDrain(t *testing.T, references []storage_files.StoredFileReference) {
	t.Helper()

	if len(references) == 0 {
		return
	}

	ctx := context.WithoutCancel(t.Context())

	err := db.GetDb().Transaction(func(tx *gorm.DB) error {
		return GetStorageFileStore().RequestFileDeletions(ctx, tx, references)
	})
	if err != nil {
		t.Errorf("failed to request deletion of written files: %v", err)
		return
	}

	if err := DrainStorageFileDeletions(ctx, references...); err != nil {
		t.Errorf("failed to drain deletion of written files: %v", err)
	}
}

func generateRandomContent(t *testing.T, size int) []byte {
	t.Helper()

	content := make([]byte, size)
	_, err := rand.Read(content)
	require.NoError(t, err)

	return content
}

func Test_WriteFile_WhenCredentialsChangeToWrongOnes_NextWriteFailsUntilCorrected(t *testing.T) {
	for _, target := range getRemoteStorageTargets() {
		t.Run(target.name, func(t *testing.T) {
			t.Parallel()

			storage := createTestStorageOfType(t, target.storageType, target.startContainer(t))
			writtenFiles := trackWrittenFiles(t, storage.ID)
			content := generateRandomContent(t, metadataSizedFileBytes)

			require.NoError(t, writtenFiles.writeUniqueFile(t.Context(), content))

			setTestStorageSecret(t, storage.ID, getWrongTestSecret(t, target.storageType))
			require.Error(
				t,
				writtenFiles.writeUniqueFile(t.Context(), content),
				"a write after the credentials changed must use the new, wrong ones",
			)

			setTestStorageSecret(t, storage.ID, getCorrectTestSecret(t, target.storageType))
			require.NoError(
				t,
				writtenFiles.writeUniqueFile(t.Context(), content),
				"a write after the credentials were corrected must use the corrected ones",
			)
		})
	}
}

func Test_WriteFile_AfterStorageIsDeleted_Fails(t *testing.T) {
	for _, target := range getRemoteStorageTargets() {
		t.Run(target.name, func(t *testing.T) {
			t.Parallel()

			storage := createTestStorageOfType(t, target.storageType, target.startContainer(t))
			writtenFiles := trackWrittenFiles(t, storage.ID)
			content := generateRandomContent(t, metadataSizedFileBytes)

			require.NoError(t, writtenFiles.writeUniqueFile(t.Context(), content))

			// The written file is removed while its storage still exists, since the deletion worker
			// can no longer reach a deleted storage.
			writtenFiles.requestDeletionAndDrainFunc(t)()
			RemoveTestStorage(t.Context(), storage.ID)

			// The write is not tracked: no file can reach a deleted storage, and a deletion request
			// for it would name a storage that no longer exists.
			_, err := GetStorageFileStore().WriteFile(
				t.Context(),
				storage_files.StoredFileReference{
					StorageID: storage.ID,
					FileName:  "after-deletion-" + uuid.New().String(),
				},
				bytes.NewReader(content),
			)
			assert.Error(t, err)
		})
	}
}

// The deadline plus the three seconds the spec allows a write to outlive it.
const (
	silentStorageWriteDeadline = 5 * time.Second
	silentStorageReturnLimit   = 8 * time.Second
)

func Test_WriteFile_WhenStorageGoesSilentMidConnection_FailsByDeadlineAndRedials(t *testing.T) {
	for _, target := range getRemoteStorageTargets() {
		if target.storageType == StorageTypeRclone {
			continue
		}

		t.Run(target.name, func(t *testing.T) {
			t.Parallel()

			container := target.startContainer(t)
			storage := createTestStorageOfType(t, target.storageType, container)
			writtenFiles := trackWrittenFiles(t, storage.ID)
			content := generateRandomContent(t, segmentSizedFileBytes)

			require.NoError(t, writtenFiles.writeUniqueFile(t.Context(), content))

			containers.SetEgressLoss(t, container)
			t.Cleanup(func() { containers.ClearEgressShaping(t, container) })

			writeCtx, cancel := context.WithTimeout(t.Context(), silentStorageWriteDeadline)
			defer cancel()

			writeResult := make(chan error, 1)
			go func() {
				writeResult <- writtenFiles.writeUniqueFile(writeCtx, content)
			}()

			select {
			case err := <-writeResult:
				require.Error(t, err, "a write to a silent storage must fail")
			case <-time.After(silentStorageReturnLimit):
				t.Fatalf("a write to a silent storage did not return within %s", silentStorageReturnLimit)
			}

			containers.ClearEgressShaping(t, container)

			require.NoError(
				t,
				writtenFiles.writeUniqueFile(t.Context(), content),
				"the next write must not reuse the connection that went silent",
			)
		})
	}
}

var rcloneSecretLine = regexp.MustCompile(`(?m)^secret_access_key = .*$`)

func setTestStorageSecret(t *testing.T, storageID uuid.UUID, secret string) {
	t.Helper()

	storage, err := getStorageRepository().FindByID(t.Context(), storageID)
	require.NoError(t, err)

	switch storage.Type {
	case StorageTypeS3:
		storage.S3Storage.S3SecretKey = secret
	case StorageTypeAzureBlob:
		storage.AzureBlobStorage.AccountKey = secret
	case StorageTypeSFTP:
		storage.SFTPStorage.Password = secret
	case StorageTypeFTP:
		storage.FTPStorage.Password = secret
	case StorageTypeNAS:
		storage.NASStorage.Password = secret
	case StorageTypeRclone:
		configContent, err := encryption.GetFieldEncryptor().Decrypt(storage.RcloneStorage.ConfigContent)
		require.NoError(t, err)

		storage.RcloneStorage.ConfigContent = rcloneSecretLine.ReplaceAllLiteralString(
			configContent, "secret_access_key = "+secret,
		)
	default:
		t.Fatalf("no test secret for storage type %s", storage.Type)
	}

	require.NoError(t, storage.EncryptSensitiveData(encryption.GetFieldEncryptor()))

	_, err = getStorageRepository().Save(storage)
	require.NoError(t, err)
}

func getCorrectTestSecret(t *testing.T, storageType StorageType) string {
	t.Helper()

	switch storageType {
	case StorageTypeS3, StorageTypeRclone:
		return containers.MinioRootPassword
	case StorageTypeAzureBlob:
		return containers.AzuriteAccountKey
	case StorageTypeSFTP:
		return containers.SftpPassword
	case StorageTypeFTP:
		return containers.FtpPassword
	case StorageTypeNAS:
		return containers.SambaPassword
	default:
		t.Fatalf("no test secret for storage type %s", storageType)
		return ""
	}
}

// An Azure account key must stay valid base64, or the client fails on decoding it and the test
// would never reach the server.
func getWrongTestSecret(t *testing.T, storageType StorageType) string {
	t.Helper()

	if storageType == StorageTypeAzureBlob {
		return base64.StdEncoding.EncodeToString(generateRandomContent(t, 64))
	}

	return "wrong-secret-" + uuid.New().String()
}

func addressOf(endpoint containers.Endpoint) string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

func generateFileContent(size int) []byte {
	generated := make([]byte, size)
	for i := range generated {
		generated[i] = byte(i % 251)
	}

	return generated
}

func setupTestFile() (string, error) {
	tempDir := os.TempDir()
	testFilePath := filepath.Join(tempDir, "test_file.txt")
	testData := []byte("This is test data for storage testing")

	err := os.WriteFile(testFilePath, testData, 0o644)
	if err != nil {
		return "", fmt.Errorf("failed to create test file: %w", err)
	}

	return testFilePath, nil
}

// setupS3Container creates the test bucket on the MinIO server at address (host:port).
func setupS3Container(ctx context.Context, address string) (*S3Container, error) {
	accessKey := containers.MinioRootUser
	secretKey := containers.MinioRootPassword
	bucketName := "test-bucket"
	region := containers.MinioRegion

	minioClient, err := minio.New(address, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		if err := minioClient.MakeBucket(
			ctx,
			bucketName,
			minio.MakeBucketOptions{Region: region},
		); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &S3Container{
		endpoint:   address,
		accessKey:  accessKey,
		secretKey:  secretKey,
		bucketName: bucketName,
		region:     region,
	}, nil
}

const testAzuriteContainerName = "test-container"

func getAzuriteConnectionString(address string) string {
	return fmt.Sprintf(
		"DefaultEndpointsProtocol=http;AccountName=%s;AccountKey=%s;BlobEndpoint=http://%s/%s",
		containers.AzuriteAccountName,
		containers.AzuriteAccountKey,
		address,
		containers.AzuriteAccountName,
	)
}

func createAzuriteBlobContainer(ctx context.Context, address string) error {
	client, err := azblob.NewClientFromConnectionString(getAzuriteConnectionString(address), nil)
	if err != nil {
		return fmt.Errorf("failed to create azblob client: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err = client.CreateContainer(ctx, testAzuriteContainerName, nil)
	if err != nil && !bloberror.HasCode(err, bloberror.ContainerAlreadyExists) {
		return fmt.Errorf("failed to create blob container: %w", err)
	}

	return nil
}

// buildTestStorage points a storage of the given type at the container and prepares whatever the
// server needs first, such as a bucket. Credentials stay in plain text so a caller can still edit
// them; saveTestStorage encrypts them.
func buildTestStorage(t *testing.T, storageType StorageType, container containers.ContainerHandle) *Storage {
	t.Helper()

	storage := &Storage{
		Type: storageType,
		Name: "Test " + string(storageType) + " " + uuid.New().String(),
	}

	address := addressOf(container.Endpoint)

	switch storageType {
	case StorageTypeLocal:
		storage.LocalStorage = &local_storage.LocalStorage{}
	case StorageTypeS3:
		s3Container, err := setupS3Container(t.Context(), address)
		require.NoError(t, err, "failed to set up the S3 container")

		storage.S3Storage = &s3_storage.S3Storage{
			S3Bucket:    s3Container.bucketName,
			S3Region:    s3Container.region,
			S3AccessKey: s3Container.accessKey,
			S3SecretKey: s3Container.secretKey,
			S3Endpoint:  "http://" + address,
		}
	case StorageTypeAzureBlob:
		require.NoError(t, createAzuriteBlobContainer(t.Context(), address))

		storage.AzureBlobStorage = &azure_blob_storage.AzureBlobStorage{
			AuthMethod:    azure_blob_storage.AuthMethodAccountKey,
			AccountName:   containers.AzuriteAccountName,
			AccountKey:    containers.AzuriteAccountKey,
			ContainerName: testAzuriteContainerName,
			Endpoint:      fmt.Sprintf("http://%s/%s", address, containers.AzuriteAccountName),
		}
	case StorageTypeNAS:
		storage.NASStorage = &nas_storage.NASStorage{
			Host:     container.Host,
			Port:     container.Port,
			Share:    containers.SambaShare,
			Username: containers.SambaUsername,
			Password: containers.SambaPassword,
			Path:     "test-files",
		}
	case StorageTypeFTP:
		storage.FTPStorage = &ftp_storage.FTPStorage{
			Host:     container.Host,
			Port:     container.Port,
			Username: containers.FtpUsername,
			Password: containers.FtpPassword,
			Path:     "test-files",
		}
	case StorageTypeSFTP:
		storage.SFTPStorage = &sftp_storage.SFTPStorage{
			Host:              container.Host,
			Port:              container.Port,
			Username:          containers.SftpUsername,
			Password:          containers.SftpPassword,
			SkipHostKeyVerify: true,
			Path:              containers.SftpUploadDir,
		}
	case StorageTypeRclone:
		s3Container, err := setupS3Container(t.Context(), address)
		require.NoError(t, err, "failed to set up the S3 container behind rclone")

		storage.RcloneStorage = &rclone_storage.RcloneStorage{
			ConfigContent: getRcloneMinioConfig(s3Container, s3Container.secretKey),
			RemotePath:    s3Container.bucketName,
		}
	default:
		t.Fatalf("no test setup for storage type %s", storageType)
	}

	return storage
}

func getRcloneMinioConfig(s3Container *S3Container, secretKey string) string {
	return fmt.Sprintf(`[minio]
type = s3
provider = Other
access_key_id = %s
secret_access_key = %s
endpoint = http://%s
acl = private`, s3Container.accessKey, secretKey, s3Container.endpoint)
}

func saveTestStorage(t *testing.T, storage *Storage) *Storage {
	t.Helper()

	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	workspace, err := workspaces_testing.CreateTestWorkspaceDirect("Storage Test Workspace", owner.UserID)
	require.NoError(t, err)

	storage.WorkspaceID = workspace.ID
	require.NoError(t, storage.EncryptSensitiveData(encryption.GetFieldEncryptor()))

	savedStorage, err := getStorageRepository().Save(storage)
	require.NoError(t, err)

	t.Cleanup(func() {
		if _, err := getStorageRepository().FindByID(context.WithoutCancel(t.Context()), savedStorage.ID); err == nil {
			RemoveTestStorage(t.Context(), savedStorage.ID)
		}

		if err := workspaces_testing.RemoveTestWorkspaceDirect(workspace.ID); err != nil {
			t.Logf("failed to remove test workspace: %v", err)
		}
	})

	return savedStorage
}

func createTestStorageOfType(
	t *testing.T,
	storageType StorageType,
	container containers.ContainerHandle,
) *Storage {
	t.Helper()

	return saveTestStorage(t, buildTestStorage(t, storageType, container))
}

func Test_RcloneStorage_DeleteFile_WhenAuthFailsOnLookup_ReturnsErrorAndDoesNotDeleteObject(t *testing.T) {
	ctx := t.Context()

	minioEndpoint := containers.StartMinio(t)

	s3Container, err := setupS3Container(ctx, addressOf(minioEndpoint))
	require.NoError(t, err)

	minioClient, err := minio.New(s3Container.endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3Container.accessKey, s3Container.secretKey, ""),
		Secure: false,
		Region: s3Container.region,
	})
	require.NoError(t, err)

	fileID := uuid.New().String()
	testData := []byte("rclone delete bug repro")

	_, err = minioClient.PutObject(
		ctx,
		s3Container.bucketName,
		fileID,
		bytes.NewReader(testData),
		int64(len(testData)),
		minio.PutObjectOptions{},
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = minioClient.RemoveObject(
			context.Background(),
			s3Container.bucketName,
			fileID,
			minio.RemoveObjectOptions{},
		)
	})

	rcloneStorage := &rclone_storage.RcloneStorage{
		StorageID:     uuid.New(),
		ConfigContent: getRcloneMinioConfig(s3Container, "totally-wrong-secret-key"),
		RemotePath:    s3Container.bucketName,
	}

	encryptor := encryption.GetFieldEncryptor()

	err = rcloneStorage.DeleteFile(t.Context(), encryptor, logger.GetLogger(), fileID)
	require.Error(t, err)

	_, statErr := minioClient.StatObject(
		ctx,
		s3Container.bucketName,
		fileID,
		minio.StatObjectOptions{},
	)
	assert.NoError(t, statErr)
}

func Test_StorageUpdate_WhenExistingStorageHasNilS3_AssignsIncomingS3(t *testing.T) {
	storageID := uuid.New()

	existing := &Storage{
		ID:        storageID,
		Type:      StorageTypeS3,
		Name:      "old name",
		S3Storage: nil,
	}

	incoming := &Storage{
		ID:   storageID,
		Type: StorageTypeS3,
		Name: "new name",
		S3Storage: &s3_storage.S3Storage{
			StorageID:   storageID,
			S3Bucket:    "my-bucket",
			S3Region:    "us-east-1",
			S3AccessKey: "access",
			S3SecretKey: "secret",
		},
	}

	existing.Update(incoming)

	assert.Equal(t, "new name", existing.Name)
	assert.NotNil(t, existing.S3Storage)
	assert.Equal(t, "my-bucket", existing.S3Storage.S3Bucket)
	assert.Equal(t, "us-east-1", existing.S3Storage.S3Region)
}

func Test_StorageUpdate_WhenExistingS3IsNil_ValidateDoesNotPanic(t *testing.T) {
	storageID := uuid.New()
	encryptor := encryption.GetFieldEncryptor()

	existing := &Storage{
		ID:        storageID,
		Type:      StorageTypeS3,
		Name:      "test",
		S3Storage: nil,
	}

	incoming := &Storage{
		ID:   storageID,
		Type: StorageTypeS3,
		Name: "test",
		S3Storage: &s3_storage.S3Storage{
			StorageID:   storageID,
			S3Bucket:    "my-bucket",
			S3Region:    "us-east-1",
			S3AccessKey: "access",
			S3SecretKey: "secret",
		},
	}

	existing.Update(incoming)

	assert.NotPanics(t, func() {
		_ = existing.Validate(encryptor)
	})
}
