package providers_test

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

	"databasus-backend/internal/lib/background"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/modules/storages/internal/providers"
	"databasus-backend/internal/modules/storages/internal/providers/azureblob"
	"databasus-backend/internal/modules/storages/internal/providers/ftp"
	"databasus-backend/internal/modules/storages/internal/providers/local"
	"databasus-backend/internal/modules/storages/internal/providers/nas"
	"databasus-backend/internal/modules/storages/internal/providers/providertest"
	"databasus-backend/internal/modules/storages/internal/providers/rclone"
	"databasus-backend/internal/modules/storages/internal/providers/s3"
	"databasus-backend/internal/modules/storages/internal/providers/sftp"
)

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

// The deadline plus the three seconds the spec allows a write to outlive it.
const (
	silentStorageWriteDeadline = 5 * time.Second
	silentStorageReturnLimit   = 8 * time.Second
)

const (
	testBucketName         = "test-bucket"
	testAzureContainerName = "test-container"
)

var rcloneSecretLine = regexp.MustCompile(`(?m)^secret_access_key = .*$`)

func TestFileOperations(t *testing.T) {
	smallContent := []byte("This is test data for storage testing")
	largeContent := generateFileContent(256 * 1024)

	for _, target := range getProviderTargets(true) {
		t.Run(target.name, func(t *testing.T) {
			testedStorage := target.build(t, target.startContainer(t))

			t.Run("connects when the settings are right", func(t *testing.T) {
				assert.NoError(t, testedStorage.provider.TestConnection(t.Context(), testedStorage.location))
			})

			t.Run("accepts the settings", func(t *testing.T) {
				assert.NoError(t, testedStorage.provider.Validate(testedStorage.location.Settings))
			})

			t.Run("returns the saved content when a file is read back", func(t *testing.T) {
				fileName := uuid.NewString()
				require.NoError(t, testedStorage.save(t, fileName, smallContent))

				assert.Equal(t, smallContent, testedStorage.read(t, fileName))
			})

			// Reads run through a ResumingReader that stops at the size the backend declared, so a
			// backend reporting the wrong length truncates the download. One inline byte of test
			// data is too small to catch that.
			t.Run("returns the whole content when a large file is read back", func(t *testing.T) {
				fileName := uuid.NewString()
				require.NoError(t, testedStorage.save(t, fileName, largeContent))

				t.Cleanup(func() {
					_ = testedStorage.provider.DeleteFile(
						context.WithoutCancel(t.Context()), testedStorage.access(fileName),
					)
				})

				assert.Equal(t, largeContent, testedStorage.read(t, fileName))
			})

			t.Run("leaves nothing to read after the file is deleted", func(t *testing.T) {
				fileName := uuid.NewString()
				require.NoError(t, testedStorage.save(t, fileName, smallContent))

				require.NoError(t, testedStorage.provider.DeleteFile(t.Context(), testedStorage.access(fileName)))

				openedFile, err := testedStorage.provider.OpenFile(t.Context(), testedStorage.access(fileName))
				require.Error(t, err, "a deleted file must not open")

				if openedFile != nil {
					_ = openedFile.Close()
				}
			})

			t.Run("succeeds when the deleted file never existed", func(t *testing.T) {
				err := testedStorage.provider.DeleteFile(t.Context(), testedStorage.access(uuid.NewString()))

				assert.NoError(t, err)
			})
		})
	}
}

func TestSaveFile(t *testing.T) {
	t.Run("pays no handshake per file and no round trip per block when the storage answers 50ms late",
		func(t *testing.T) {
			for _, target := range getProviderTargets(false) {
				t.Run(target.name, func(t *testing.T) {
					t.Parallel()

					container := target.startContainer(t)
					containers.AddEgressDelay(t, container, roundTripEgressDelay)
					t.Cleanup(func() { containers.ClearEgressShaping(t, container) })

					// FTP data connections still go straight to the container's passive ports, so
					// only control connections, one per session, cross the forwarder.
					forwarder := providertest.StartCountingForwarder(t, providertest.GetAddress(container.Endpoint))
					forwardedContainer := container
					forwardedContainer.Endpoint = containers.Endpoint{Host: forwarder.Host, Port: forwarder.Port}

					testedStorage := target.build(t, forwardedContainer)
					metadataContent := generateRandomContent(t, metadataSizedFileBytes)
					segmentContent := generateRandomContent(t, segmentSizedFileBytes)

					// The first write opens the session, and its handshake is not what the test
					// measures.
					require.NoError(t, testedStorage.saveUnique(t, metadataContent))
					connectionsAfterFirstWrite := forwarder.GetAcceptedCount()

					for range reusedFileWriteCount {
						require.NoError(t, testedStorage.saveUnique(t, metadataContent))
					}

					assert.Equal(
						t,
						connectionsAfterFirstWrite,
						forwarder.GetAcceptedCount(),
						"%d files written one after another must reuse the connection the first write opened",
						reusedFileWriteCount,
					)

					startedAt := time.Now()
					require.NoError(t, testedStorage.saveUnique(t, segmentContent))
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
		})

	t.Run("fails the next write until the credentials are corrected when they change to wrong ones",
		func(t *testing.T) {
			for _, target := range getProviderTargets(false) {
				t.Run(target.name, func(t *testing.T) {
					t.Parallel()

					testedStorage := target.build(t, target.startContainer(t))
					content := generateRandomContent(t, metadataSizedFileBytes)

					require.NoError(t, testedStorage.saveUnique(t, content))

					testedStorage.location = testedStorage.withSecret(t, testedStorage.wrongSecret)
					require.Error(
						t,
						testedStorage.saveUnique(t, content),
						"a write after the credentials changed must use the new, wrong ones",
					)

					testedStorage.location = testedStorage.withSecret(t, testedStorage.correctSecret)
					require.NoError(
						t,
						testedStorage.saveUnique(t, content),
						"a write after the credentials were corrected must use the corrected ones",
					)
				})
			}
		})

	t.Run("fails by the deadline and redials when the storage goes silent mid-connection", func(t *testing.T) {
		for _, target := range getProviderTargets(false) {
			if target.name == "Rclone" {
				continue
			}

			t.Run(target.name, func(t *testing.T) {
				t.Parallel()

				container := target.startContainer(t)
				testedStorage := target.build(t, container)
				content := generateRandomContent(t, segmentSizedFileBytes)

				require.NoError(t, testedStorage.saveUnique(t, content))

				containers.SetEgressLoss(t, container)
				t.Cleanup(func() { containers.ClearEgressShaping(t, container) })

				writeCtx, cancel := context.WithTimeout(t.Context(), silentStorageWriteDeadline)
				defer cancel()

				writeResult := make(chan error, 1)
				go func() {
					writeResult <- testedStorage.provider.SaveFile(
						writeCtx, testedStorage.access(uuid.NewString()), bytes.NewReader(content),
					)
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
					testedStorage.saveUnique(t, content),
					"the next write must not reuse the connection that went silent",
				)
			})
		}
	})
}

func TestRcloneDeleteFile(t *testing.T) {
	t.Run("returns an error and keeps the object when the sign-in fails on the lookup", func(t *testing.T) {
		address := providertest.GetAddress(containers.StartMinio(t))
		minioClient := createTestBucket(t, address)
		fileName := uuid.NewString()
		testData := []byte("rclone delete bug repro")

		_, err := minioClient.PutObject(
			t.Context(),
			testBucketName,
			fileName,
			bytes.NewReader(testData),
			int64(len(testData)),
			minio.PutObjectOptions{},
		)
		require.NoError(t, err)

		fieldEncryptor := providertest.NewFieldEncryptor(t)
		provider := rclone.New(rclone.Deps{Repositories: rclone.Repositories{}, FieldEncryptor: fieldEncryptor})
		location := providers.Location{
			StorageID: uuid.New(),
			Settings: rclone.Settings{
				ConfigContent: getRcloneMinioConfig(address, "totally-wrong-secret-key"),
				RemotePath:    testBucketName,
			},
		}

		err = provider.DeleteFile(t.Context(), providers.FileAccess{
			Location: location,
			FileName: fileName,
			Logger:   providertest.NewDiscardLogger(),
		})
		require.Error(t, err)

		_, statErr := minioClient.StatObject(t.Context(), testBucketName, fileName, minio.StatObjectOptions{})
		assert.NoError(t, statErr)
	})
}

type providerTarget struct {
	name           string
	startContainer func(t *testing.T) containers.ContainerHandle
	build          func(t *testing.T, container containers.ContainerHandle) *testedStorage
}

// testedStorage is a provider with the location of one storage, whose secrets are encrypted as a
// repository returns them. Every file the test writes is deleted when the test ends.
type testedStorage struct {
	provider      providers.Provider
	location      providers.Location
	correctSecret string
	wrongSecret   string
	// withSecretOf returns the settings with their credential replaced by secret.
	withSecretOf func(secret string) providers.Settings

	mutex          sync.Mutex
	savedFileNames []string
}

func getProviderTargets(isLocalIncluded bool) []providerTarget {
	targets := []providerTarget{
		{"S3", containers.StartMinioContainer, buildS3Storage},
		{"AzureBlob", containers.StartAzuriteContainer, buildAzureBlobStorage},
		{"SFTP", containers.StartSftpContainer, buildSftpStorage},
		{"FTP", containers.StartFtpContainer, buildFtpStorage},
		{"NAS", containers.StartSambaContainer, buildNasStorage},
		{"Rclone", containers.StartMinioContainer, buildRcloneStorage},
	}

	if isLocalIncluded {
		targets = append([]providerTarget{{
			name:           "Local",
			startContainer: func(*testing.T) containers.ContainerHandle { return containers.ContainerHandle{} },
			build:          buildLocalStorage,
		}}, targets...)
	}

	return targets
}

func newTestedStorage(t *testing.T, provider providers.Provider, settings providers.Settings) *testedStorage {
	t.Helper()

	storage := &testedStorage{
		provider:       provider,
		location:       providers.Location{StorageID: uuid.New(), Settings: settings},
		correctSecret:  "",
		wrongSecret:    "wrong-secret-" + uuid.NewString(),
		withSecretOf:   nil,
		mutex:          sync.Mutex{},
		savedFileNames: nil,
	}

	t.Cleanup(func() { storage.deleteSavedFiles(context.WithoutCancel(t.Context())) })

	return storage
}

func (s *testedStorage) access(fileName string) providers.FileAccess {
	return providers.FileAccess{Location: s.location, FileName: fileName, Logger: providertest.NewDiscardLogger()}
}

func (s *testedStorage) withSecret(t *testing.T, secret string) providers.Location {
	t.Helper()

	return providers.Location{StorageID: s.location.StorageID, Settings: s.withSecretOf(secret)}
}

func (s *testedStorage) save(t *testing.T, fileName string, content []byte) error {
	t.Helper()

	s.mutex.Lock()
	s.savedFileNames = append(s.savedFileNames, fileName)
	s.mutex.Unlock()

	return s.provider.SaveFile(t.Context(), s.access(fileName), bytes.NewReader(content))
}

func (s *testedStorage) saveUnique(t *testing.T, content []byte) error {
	t.Helper()

	return s.save(t, "test-file-"+uuid.NewString(), content)
}

func (s *testedStorage) read(t *testing.T, fileName string) []byte {
	t.Helper()

	openedFile, err := s.provider.OpenFile(t.Context(), s.access(fileName))
	require.NoError(t, err)

	defer func() { _ = openedFile.Close() }()

	content, err := io.ReadAll(openedFile)
	require.NoError(t, err)

	return content
}

// deleteSavedFiles runs with the location the storage ended with, which a test of changed
// credentials leaves on the correct ones.
func (s *testedStorage) deleteSavedFiles(ctx context.Context) {
	s.mutex.Lock()
	fileNames := s.savedFileNames
	s.savedFileNames = nil
	s.mutex.Unlock()

	for _, fileName := range fileNames {
		_ = s.provider.DeleteFile(ctx, s.access(fileName))
	}
}

func buildLocalStorage(t *testing.T, _ containers.ContainerHandle) *testedStorage {
	t.Helper()

	folders := providers.LocalFolders{
		TempFolder: filepath.Join(t.TempDir(), "temp"),
		DataFolder: filepath.Join(t.TempDir(), "data"),
	}
	require.NoError(t, os.MkdirAll(folders.TempFolder, 0o755))

	provider := local.New(local.Deps{Repositories: local.Repositories{}, Folders: folders})

	return newTestedStorage(t, provider, local.Settings{})
}

func buildS3Storage(t *testing.T, container containers.ContainerHandle) *testedStorage {
	t.Helper()

	address := providertest.GetAddress(container.Endpoint)
	createTestBucket(t, address)

	fieldEncryptor := providertest.NewFieldEncryptor(t)
	provider := s3.New(s3.Deps{Repositories: s3.Repositories{}, FieldEncryptor: fieldEncryptor})
	settingsWithSecret := func(secret string) providers.Settings {
		return s3.Settings{
			Bucket:               testBucketName,
			Region:               containers.MinioRegion,
			AccessKey:            providertest.Encrypt(t, fieldEncryptor, containers.MinioRootUser),
			SecretKey:            providertest.Encrypt(t, fieldEncryptor, secret),
			Endpoint:             "http://" + address,
			Prefix:               "",
			IsVirtualHostedStyle: false,
			IsSkipTLSVerify:      false,
			StorageClass:         "STANDARD",
		}
	}

	return withSecrets(newTestedStorage(t, provider, settingsWithSecret(containers.MinioRootPassword)),
		containers.MinioRootPassword, settingsWithSecret)
}

func buildAzureBlobStorage(t *testing.T, container containers.ContainerHandle) *testedStorage {
	t.Helper()

	address := providertest.GetAddress(container.Endpoint)
	createAzuriteBlobContainer(t, address)

	fieldEncryptor := providertest.NewFieldEncryptor(t)
	provider := azureblob.New(azureblob.Deps{Repositories: azureblob.Repositories{}, FieldEncryptor: fieldEncryptor})
	settingsWithSecret := func(secret string) providers.Settings {
		return azureblob.Settings{
			AuthMethod:       azureblob.AuthMethodAccountKey,
			ConnectionString: "",
			AccountName:      containers.AzuriteAccountName,
			AccountKey:       providertest.Encrypt(t, fieldEncryptor, secret),
			ContainerName:    testAzureContainerName,
			Endpoint:         fmt.Sprintf("http://%s/%s", address, containers.AzuriteAccountName),
			Prefix:           "",
		}
	}

	testedAzureStorage := withSecrets(newTestedStorage(t, provider, settingsWithSecret(containers.AzuriteAccountKey)),
		containers.AzuriteAccountKey, settingsWithSecret)

	// An account key must stay valid base64, or the client fails on decoding it and the test would
	// never reach the server.
	testedAzureStorage.wrongSecret = base64.StdEncoding.EncodeToString(generateRandomContent(t, 64))

	return testedAzureStorage
}

func buildNasStorage(t *testing.T, container containers.ContainerHandle) *testedStorage {
	t.Helper()

	fieldEncryptor := providertest.NewFieldEncryptor(t)
	provider := nas.New(nas.Deps{
		Repositories:   nas.Repositories{},
		FieldEncryptor: fieldEncryptor,
		ChunkWrites:    background.New(background.Deps{Logger: providertest.NewDiscardLogger()}),
	})
	settingsWithSecret := func(secret string) providers.Settings {
		return nas.Settings{
			Host:     container.Host,
			Port:     container.Port,
			Share:    containers.SambaShare,
			Username: containers.SambaUsername,
			Password: providertest.Encrypt(t, fieldEncryptor, secret),
			IsSSL:    false,
			Domain:   "",
			Path:     "test-files",
		}
	}

	return withSecrets(newTestedStorage(t, provider, settingsWithSecret(containers.SambaPassword)),
		containers.SambaPassword, settingsWithSecret)
}

func buildFtpStorage(t *testing.T, container containers.ContainerHandle) *testedStorage {
	t.Helper()

	fieldEncryptor := providertest.NewFieldEncryptor(t)
	provider := ftp.New(ftp.Deps{Repositories: ftp.Repositories{}, FieldEncryptor: fieldEncryptor})
	settingsWithSecret := func(secret string) providers.Settings {
		return ftp.Settings{
			Host:            container.Host,
			Port:            container.Port,
			Username:        containers.FtpUsername,
			Password:        providertest.Encrypt(t, fieldEncryptor, secret),
			Path:            "test-files",
			IsSSL:           false,
			IsSkipTLSVerify: false,
		}
	}

	return withSecrets(newTestedStorage(t, provider, settingsWithSecret(containers.FtpPassword)),
		containers.FtpPassword, settingsWithSecret)
}

func buildSftpStorage(t *testing.T, container containers.ContainerHandle) *testedStorage {
	t.Helper()

	fieldEncryptor := providertest.NewFieldEncryptor(t)
	provider := sftp.New(sftp.Deps{Repositories: sftp.Repositories{}, FieldEncryptor: fieldEncryptor})
	settingsWithSecret := func(secret string) providers.Settings {
		return sftp.Settings{
			AuthMethods:         []sftp.AuthMethod{sftp.AuthMethodPassword},
			Host:                container.Host,
			Port:                container.Port,
			Username:            containers.SftpUsername,
			Password:            providertest.Encrypt(t, fieldEncryptor, secret),
			PrivateKey:          "",
			Path:                containers.SftpUploadDir,
			IsSkipHostKeyVerify: true,
		}
	}

	return withSecrets(newTestedStorage(t, provider, settingsWithSecret(containers.SftpPassword)),
		containers.SftpPassword, settingsWithSecret)
}

func buildRcloneStorage(t *testing.T, container containers.ContainerHandle) *testedStorage {
	t.Helper()

	address := providertest.GetAddress(container.Endpoint)
	createTestBucket(t, address)

	fieldEncryptor := providertest.NewFieldEncryptor(t)
	provider := rclone.New(rclone.Deps{Repositories: rclone.Repositories{}, FieldEncryptor: fieldEncryptor})
	settingsWithSecret := func(secret string) providers.Settings {
		configContent := rcloneSecretLine.ReplaceAllLiteralString(
			getRcloneMinioConfig(address, containers.MinioRootPassword), "secret_access_key = "+secret,
		)

		return rclone.Settings{
			ConfigContent: providertest.Encrypt(t, fieldEncryptor, configContent),
			RemotePath:    testBucketName,
		}
	}

	return withSecrets(newTestedStorage(t, provider, settingsWithSecret(containers.MinioRootPassword)),
		containers.MinioRootPassword, settingsWithSecret)
}

func withSecrets(
	storage *testedStorage,
	correctSecret string,
	settingsWithSecret func(secret string) providers.Settings,
) *testedStorage {
	storage.correctSecret = correctSecret
	storage.withSecretOf = settingsWithSecret

	return storage
}

func getRcloneMinioConfig(address, secretKey string) string {
	return fmt.Sprintf(`[minio]
type = s3
provider = Other
access_key_id = %s
secret_access_key = %s
endpoint = http://%s
acl = private`, containers.MinioRootUser, secretKey, address)
}

// createTestBucket creates the test bucket on the MinIO server at address (host:port).
func createTestBucket(t *testing.T, address string) *minio.Client {
	t.Helper()

	minioClient, err := minio.New(address, &minio.Options{
		Creds:  credentials.NewStaticV4(containers.MinioRootUser, containers.MinioRootPassword, ""),
		Secure: false,
		Region: containers.MinioRegion,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	isFound, err := minioClient.BucketExists(ctx, testBucketName)
	require.NoError(t, err)

	if !isFound {
		require.NoError(t, minioClient.MakeBucket(ctx, testBucketName, minio.MakeBucketOptions{
			Region: containers.MinioRegion,
		}))
	}

	return minioClient
}

func createAzuriteBlobContainer(t *testing.T, address string) {
	t.Helper()

	connectionString := fmt.Sprintf(
		"DefaultEndpointsProtocol=http;AccountName=%s;AccountKey=%s;BlobEndpoint=http://%s/%s",
		containers.AzuriteAccountName,
		containers.AzuriteAccountKey,
		address,
		containers.AzuriteAccountName,
	)

	client, err := azblob.NewClientFromConnectionString(connectionString, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	_, err = client.CreateContainer(ctx, testAzureContainerName, nil)
	if err != nil && !bloberror.HasCode(err, bloberror.ContainerAlreadyExists) {
		require.NoError(t, err)
	}
}

func generateRandomContent(t *testing.T, size int) []byte {
	t.Helper()

	content := make([]byte, size)
	_, err := rand.Read(content)
	require.NoError(t, err)

	return content
}

func generateFileContent(size int) []byte {
	generated := make([]byte, size)
	for i := range generated {
		generated[i] = byte(i % 251)
	}

	return generated
}
