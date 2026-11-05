package storages_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/modules/storages/internal/providers/providertest"
)

func TestS3StorageClassConnection(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name           string
		storageClass   *apiclient.S3StorageClass
		isRejectWrites bool
	}{
		{name: "uses the bucket default without an explicit class", storageClass: nil},
		{name: "uses the bucket default with an empty class", storageClass: new("")},
		{name: "reports provider rejection without replacing COLD", storageClass: new("COLD"), isRejectWrites: true},
		{name: "reports provider rejection without replacing DEEP_ARCHIVE", storageClass: new("DEEP_ARCHIVE"), isRejectWrites: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			tm := newTeam(t)
			bucket := startMinioBucket(t)
			proxy := providertest.StartS3ClassProxy(t, bucket.endpointURL, scenario.isRejectWrites)
			storage := createStorage(t, tm.owner, apiclient.StorageCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Offsite",
				Settings: newS3Settings(t, apiclient.S3StorageSettings{
					S3Bucket:       bucket.name,
					S3Region:       new(containers.MinioRegion),
					S3Endpoint:     new(proxy.URL),
					S3AccessKey:    new(containers.MinioRootUser),
					S3SecretKey:    new(containers.MinioRootPassword),
					S3StorageClass: scenario.storageClass,
				}),
			})

			connection := testSavedStorage(t, tm.owner, storage.Id)

			assert.Equal(t, !scenario.isRejectWrites, connection.IsConnected)
			if scenario.isRejectWrites {
				require.NotNil(t, connection.Error)
				assert.Contains(t, *connection.Error, "The storage class is not supported")
			} else {
				assert.Nil(t, connection.Error)
			}

			writeHeaders := proxy.GetWriteHeaders()
			require.Len(t, writeHeaders, 1, "a nonretryable rejection must not send another class")

			savedSettings, err := getStorage(t, tm.owner, storage.Id).Settings.AsS3StorageSettings()
			require.NoError(t, err)

			if scenario.storageClass == nil || *scenario.storageClass == "" {
				assert.NotContains(t, writeHeaders[0], "X-Amz-Storage-Class")
				assert.Nil(t, savedSettings.S3StorageClass)
			} else {
				assert.Equal(t, *scenario.storageClass, writeHeaders[0].Get("X-Amz-Storage-Class"))
				require.NotNil(t, savedSettings.S3StorageClass)
				assert.Equal(t, *scenario.storageClass, *savedSettings.S3StorageClass)
			}
		})
	}
}
