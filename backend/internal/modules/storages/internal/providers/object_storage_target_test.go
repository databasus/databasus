package providers_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/modules/storages/internal/providers"
	"databasus-backend/internal/modules/storages/internal/providers/providertest"
	"databasus-backend/internal/modules/storages/internal/providers/s3"
)

// objectStorageTarget is one bucket the object storage scenario writes to: the S3 provider with the
// settings the storage form saves, and a raw client that lists and removes what the scenario leaves.
type objectStorageTarget struct {
	provider  providers.Provider
	settings  s3.Settings
	rawClient *minio.Client
	redactor  keyRedactor
}

func (target objectStorageTarget) describe() string {
	return fmt.Sprintf("bucket %s at %s", target.settings.Bucket, target.settings.Endpoint)
}

func (target objectStorageTarget) getLocation(prefix string) providers.Location {
	settings := target.settings
	settings.Prefix = prefix

	return providers.Location{StorageID: uuid.New(), Settings: settings}
}

// newNeonObjectStorageTarget configures the bucket as the Neon Object Storage form does: path-style
// addressing and no storage class.
func newNeonObjectStorageTarget(t *testing.T, settings neonObjectStorageSettings) objectStorageTarget {
	t.Helper()

	endpointURL, err := url.Parse(settings.endpoint)
	require.NoError(t, err, "parse %s", neonObjectStorageEndpointVariable)

	rawClient, err := minio.New(endpointURL.Host, &minio.Options{
		Creds:        credentials.NewStaticV4(settings.accessKey, settings.secretKey, ""),
		Secure:       endpointURL.Scheme != "http",
		Region:       settings.region,
		BucketLookup: minio.BucketLookupPath,
	})
	require.NoError(t, err)

	return newObjectStorageTarget(t, objectStorageSpec{
		endpoint:  settings.endpoint,
		region:    settings.region,
		bucket:    settings.bucket,
		accessKey: settings.accessKey,
		secretKey: settings.secretKey,
		rawClient: rawClient,
	})
}

// startMinioObjectStorageTarget is the scenario's twin on a MinIO container, configured the same way.
// The provider authenticates with secretKey, so a twin can be given a key MinIO refuses while its raw
// client still cleans up.
func startMinioObjectStorageTarget(t *testing.T, secretKey string) objectStorageTarget {
	t.Helper()

	address := providertest.GetAddress(containers.StartMinioContainer(t).Endpoint)
	rawClient := createTestBucket(t, address)

	return newObjectStorageTarget(t, objectStorageSpec{
		endpoint:  "http://" + address,
		region:    containers.MinioRegion,
		bucket:    testBucketName,
		accessKey: containers.MinioRootUser,
		secretKey: secretKey,
		rawClient: rawClient,
	})
}

// withRawClientThrough sends the target's own listing and removal through address, such as a proxy
// that refuses them.
func (target objectStorageTarget) withRawClientThrough(t *testing.T, address string) objectStorageTarget {
	t.Helper()

	rawClient, err := minio.New(address, &minio.Options{
		Creds:  credentials.NewStaticV4(containers.MinioRootUser, containers.MinioRootPassword, ""),
		Secure: false,
		Region: containers.MinioRegion,
	})
	require.NoError(t, err)

	target.rawClient = rawClient

	return target
}

type objectStorageSpec struct {
	endpoint  string
	region    string
	bucket    string
	accessKey string
	secretKey string
	rawClient *minio.Client
}

func newObjectStorageTarget(t *testing.T, spec objectStorageSpec) objectStorageTarget {
	t.Helper()

	fieldEncryptor := providertest.NewFieldEncryptor(t)

	return objectStorageTarget{
		provider: s3.New(s3.Deps{Repositories: s3.Repositories{}, FieldEncryptor: fieldEncryptor}),
		settings: s3.Settings{
			Bucket:               spec.bucket,
			Region:               spec.region,
			AccessKey:            providertest.Encrypt(t, fieldEncryptor, spec.accessKey),
			SecretKey:            providertest.Encrypt(t, fieldEncryptor, spec.secretKey),
			Endpoint:             spec.endpoint,
			Prefix:               "",
			IsVirtualHostedStyle: false,
			IsSkipTLSVerify:      false,
			StorageClass:         "",
		},
		rawClient: spec.rawClient,
		redactor:  newKeyRedactor(spec.accessKey, spec.secretKey),
	}
}
