package apptest

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

type StorageNamespace struct {
	StorageID uuid.UUID
	Bucket    string
	Client    *minio.Client
}

func CreateStorageNamespace(t *testing.T, owner Session, workspaceID uuid.UUID) StorageNamespace {
	t.Helper()

	endpoint := containers.StartMinio(t)
	address := fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)

	client, err := minio.New(address, &minio.Options{
		Creds:  credentials.NewStaticV4(containers.MinioRootUser, containers.MinioRootPassword, ""),
		Region: containers.MinioRegion,
	})
	require.NoError(t, err)
	bucket := "switch-" + uuid.NewString()
	require.NoError(t, client.MakeBucket(t.Context(), bucket, minio.MakeBucketOptions{Region: containers.MinioRegion}))

	var settings apiclient.StorageSettings
	require.NoError(t, settings.FromS3StorageSettings(apiclient.S3StorageSettings{
		Type:        apiclient.S3StorageSettingsTypeS3,
		S3Bucket:    bucket,
		S3Region:    new(containers.MinioRegion),
		S3Endpoint:  new("http://" + address),
		S3AccessKey: new(containers.MinioRootUser),
		S3SecretKey: new(containers.MinioRootPassword),
	}))
	storage := CreateStorage(t, owner, workspaceID, settings)

	return StorageNamespace{StorageID: storage.Id, Bucket: bucket, Client: client}
}

func (namespace StorageNamespace) RequireFile(t *testing.T, fileName string) {
	t.Helper()

	object, err := namespace.Client.StatObject(t.Context(), namespace.Bucket, fileName, minio.StatObjectOptions{})
	require.NoError(t, err)
	require.Positive(t, object.Size)
}

func (namespace StorageNamespace) RequireFileAbsent(t *testing.T, fileName string) {
	t.Helper()
	_, err := namespace.Client.StatObject(t.Context(), namespace.Bucket, fileName, minio.StatObjectOptions{})
	require.Equal(t, http.StatusNotFound, minio.ToErrorResponse(err).StatusCode)
}

func (namespace StorageNamespace) MakeUnavailable(t *testing.T, owner Session) {
	t.Helper()
	namespace.setEndpoint(t, owner, "http://127.0.0.1:1")
}

func (namespace StorageNamespace) setEndpoint(t *testing.T, owner Session, endpoint string) {
	t.Helper()

	response, err := owner.Client.GetStorageWithResponse(t.Context(), namespace.StorageID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)

	s3Settings, err := response.JSON200.Settings.AsS3StorageSettings()
	require.NoError(t, err)
	s3Settings.S3Endpoint = &endpoint
	s3Settings.S3AccessKey = new(containers.MinioRootUser)
	s3Settings.S3SecretKey = new(containers.MinioRootPassword)
	var settings apiclient.StorageSettings
	require.NoError(t, settings.FromS3StorageSettings(s3Settings))

	updated, err := owner.Client.UpdateStorageWithResponse(t.Context(), namespace.StorageID, apiclient.StorageUpdate{
		Name: response.JSON200.Name, Settings: settings,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, updated.StatusCode(), "%s", updated.Body)
}
