package s3

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/modules/storages/internal/providers/providertest"
)

func TestStorageClass(t *testing.T) {
	t.Parallel()

	for _, storageClass := range []string{"", "COLD", "DEEP_ARCHIVE"} {
		className := storageClass
		if className == "" {
			className = "empty"
		}
		t.Run("forwards the configured class without substitution when it is "+className, func(t *testing.T) {
			t.Parallel()
			address := providertest.GetAddress(containers.StartMinio(t))
			rawClient := newRawClient(t, address)
			createTestBucket(t, rawClient)
			isRejected := storageClass != ""
			proxy := providertest.StartS3ClassProxy(t, "http://"+address, isRejected)
			testedBucket := newTestBucket(t, Settings{
				Bucket:       testBucketName,
				Region:       containers.MinioRegion,
				Endpoint:     proxy.URL,
				StorageClass: storageClass,
			})

			uploadError := testedBucket.provider.SaveFile(
				t.Context(),
				testedBucket.access("payload"),
				strings.NewReader("payload"),
			)
			connectionError := testedBucket.provider.TestConnection(t.Context(), testedBucket.location())

			if isRejected {
				require.Error(t, uploadError)
				assert.Contains(t, uploadError.Error(), "The storage class is not supported")
				require.Error(t, connectionError)
				assert.Contains(t, connectionError.Error(), "The storage class is not supported")
			} else {
				require.NoError(t, uploadError)
				require.NoError(t, connectionError)
				assert.Equal(t, "payload", string(readWholeFile(t, testedBucket, "payload")))
			}

			writeHeaders := proxy.GetWriteHeaders()
			require.Len(t, writeHeaders, 2)
			for _, header := range writeHeaders {
				if storageClass == "" {
					assert.NotContains(t, header, "X-Amz-Storage-Class")
				} else {
					assert.Equal(t, storageClass, header.Get("X-Amz-Storage-Class"))
				}
			}
		})
	}
}
