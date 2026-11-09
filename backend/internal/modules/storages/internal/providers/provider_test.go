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
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/storages/internal/providers"
	"databasus-backend/internal/modules/storages/internal/providers/azureblob"
	"databasus-backend/internal/modules/storages/internal/providers/ftp"
	"databasus-backend/internal/modules/storages/internal/providers/local"
	"databasus-backend/internal/modules/storages/internal/providers/nas"
	"databasus-backend/internal/modules/storages/internal/providers/providertest"
	"databasus-backend/internal/modules/storages/internal/providers/s3"
	"databasus-backend/internal/modules/storages/internal/providers/sftp"
)

const (
	reusedFileWriteCount   = 10
	segmentSizedFileBytes  = 16 * 1024 * 1024
	metadataSizedFileBytes = 1024
)

const (
	testBucketName         = "test-bucket"
	testAzureContainerName = "test-container"
)

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
	t.Run("reuses the connection when consecutive files reach the same storage",
		func(t *testing.T) {
			for _, target := range getProviderTargets(false) {
				t.Run(target.name, func(t *testing.T) {
					t.Parallel()

					container := target.startContainer(t)
					// FTP data connections still go straight to the container's passive ports, so
					// only control connections, one per session, cross the forwarder.
					forwarder := providertest.StartCountingForwarder(t, providertest.GetAddress(container.Endpoint))
					forwardedContainer := container
					forwardedContainer.Endpoint = containers.Endpoint{Host: forwarder.Host, Port: forwarder.Port}

					testedStorage := target.build(t, forwardedContainer)
					metadataContent := generateRandomContent(t, metadataSizedFileBytes)
					segmentContent := generateRandomContent(t, segmentSizedFileBytes)

					require.NoError(t, testedStorage.saveUnique(t, metadataContent))
					connectionsAfterFirstWrite := forwarder.GetAcceptedCount()
					require.Positive(t, connectionsAfterFirstWrite)

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

					fileName := "segment-" + uuid.NewString()
					require.NoError(t, testedStorage.save(t, fileName, segmentContent))
					assert.Equal(t, segmentContent, testedStorage.read(t, fileName))
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

	t.Run("cancels the write and redials when a storage response is withheld", func(t *testing.T) {
		for _, target := range getProviderTargets(false) {
			t.Run(target.name, func(t *testing.T) {
				t.Parallel()

				container := target.startContainer(t)
				forwarder := providertest.StartCountingForwarder(t, providertest.GetAddress(container.Endpoint))
				container.Endpoint = containers.Endpoint{Host: forwarder.Host, Port: forwarder.Port}
				testedStorage := target.build(t, container)
				content := generateRandomContent(t, segmentSizedFileBytes)
				require.NoError(t, testedStorage.saveUnique(t, content))
				connectionsBeforeCancel := forwarder.GetAcceptedCount()
				closedBeforeCancel := forwarder.GetClosedCount()

				gate := forwarder.PauseResponses(t)
				writeCtx, cancel := context.WithCancel(t.Context())
				writeErrors := make(chan error, 1)
				fileName := "cancelled-" + uuid.NewString()
				testedStorage.savedFileNames = append(testedStorage.savedFileNames, fileName)
				var writes sync.WaitGroup
				defer func() { cancel(); gate.Release(); writes.Wait() }()
				writes.Go(func() {
					writeErrors <- testedStorage.provider.SaveFile(writeCtx, testedStorage.access(fileName), bytes.NewReader(content))
				})

				gate.WaitUntilBlocked(t)
				cancel()

				testwait.Until(t, "the cancelled storage write to return", func(context.Context) error {
					select {
					case writeError := <-writeErrors:
						require.Error(t, writeError, "a write with a withheld response must fail after cancellation")
						return nil
					default:
						return fmt.Errorf("the cancelled write is still running")
					}
				})

				testwait.Until(t, "the cancelled storage connection to close", func(context.Context) error {
					if forwarder.GetClosedCount() <= closedBeforeCancel {
						return fmt.Errorf("the cancelled connection is still open")
					}

					return nil
				})

				gate.Release()
				recoveredFileName := "recovered-" + uuid.NewString()
				require.NoError(t, testedStorage.save(t, recoveredFileName, content))
				assert.Greater(t, forwarder.GetAcceptedCount(), connectionsBeforeCancel,
					"the write after cancellation must open a new connection")
				assert.Equal(t, content, testedStorage.read(t, recoveredFileName))
			})
		}
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

func withSecrets(
	storage *testedStorage,
	correctSecret string,
	settingsWithSecret func(secret string) providers.Settings,
) *testedStorage {
	storage.correctSecret = correctSecret
	storage.withSecretOf = settingsWithSecret

	return storage
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
