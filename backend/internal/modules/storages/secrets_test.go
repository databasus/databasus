package storages_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

const sftpPrivateKey = "-----BEGIN OPENSSH PRIVATE KEY-----\noriginal-private-key\n-----END OPENSSH PRIVATE KEY-----"

func TestCreateStorage(t *testing.T) {
	t.Parallel()

	t.Run("returns the saved S3 storage without its keys", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)

		response, err := tm.owner.Client.CreateStorageWithResponse(t.Context(), apiclient.StorageCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Offsite",
			Settings: newS3Settings(t, apiclient.S3StorageSettings{
				S3Bucket:    "backups",
				S3Region:    new("eu-central-1"),
				S3AccessKey: new("AKIA-SECRET-ACCESS"),
				S3SecretKey: new("secret-key-value"),
			}),
		})

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
		assert.NotContains(t, string(response.Body), "AKIA-SECRET-ACCESS")
		assert.NotContains(t, string(response.Body), "secret-key-value")
		assert.NotContains(t, string(response.Body), "enc:")
		assert.Equal(t, "Offsite", response.JSON201.Name)
		assert.Equal(t, tm.workspace.Id, response.JSON201.WorkspaceId)

		createdS3, err := response.JSON201.Settings.AsS3StorageSettings()
		require.NoError(t, err)
		assert.Equal(t, "backups", createdS3.S3Bucket)
		assert.Equal(t, new("eu-central-1"), createdS3.S3Region)
		assert.Equal(t, "Offsite", getStorage(t, tm.owner, response.JSON201.Id).Name)
	})
}

func TestListStorages(t *testing.T) {
	t.Parallel()

	t.Run("lists an SFTP storage without its password and private key", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createStorage(t, tm.owner, apiclient.StorageCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Archive",
			Settings: newSftpSettings(t, apiclient.SftpStorageSettings{
				AuthMethods: []apiclient.SftpAuthMethod{
					apiclient.SftpAuthMethodPASSWORD,
					apiclient.SftpAuthMethodPRIVATEKEY,
				},
				Host:       "sftp.example.com",
				Port:       22,
				Username:   "backup",
				Password:   new("sftp-password-value"),
				PrivateKey: new(sftpPrivateKey),
				Path:       new("/archive"),
			}),
		})

		response, err := tm.owner.Client.ListStoragesWithResponse(t.Context(), &apiclient.ListStoragesParams{
			WorkspaceId: tm.workspace.Id,
			Limit:       nil,
			After:       nil,
		})

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.NotContains(t, string(response.Body), "sftp-password-value")
		assert.NotContains(t, string(response.Body), "original-private-key")
		require.Len(t, response.JSON200.Items, 1)

		listedSftp, err := response.JSON200.Items[0].Settings.AsSftpStorageSettings()
		require.NoError(t, err)
		assert.Equal(t, "sftp.example.com", listedSftp.Host)
		assert.Equal(t, 22, listedSftp.Port)
		assert.Equal(t, "backup", listedSftp.Username)
		assert.Equal(t, new("/archive"), listedSftp.Path)
		assert.Nil(t, listedSftp.Password)
		assert.Nil(t, listedSftp.PrivateKey)
	})
}

func TestUpdateStorageSecrets(t *testing.T) {
	t.Parallel()

	t.Run("writes the next backup with the stored keys after a rename without them", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		bucket := startMinioBucket(t)
		storedAccessKey := tm.app.EncryptSecret(t, containers.MinioRootUser)
		storedSecretKey := tm.app.EncryptSecret(t, containers.MinioRootPassword)
		storageID := insertStoredS3Storage(t, tm, storedS3Storage{
			bucket:             bucket,
			encryptedAccessKey: storedAccessKey,
			encryptedSecretKey: storedSecretKey,
		})

		response := updateStorage(t, tm.owner, storageID, apiclient.StorageUpdate{
			Name: "Renamed offsite",
			Settings: newS3Settings(t, apiclient.S3StorageSettings{
				S3Bucket:   bucket.name,
				S3Region:   new(containers.MinioRegion),
				S3Endpoint: new(bucket.endpointURL),
			}),
		})

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, "Renamed offsite", getStorage(t, tm.owner, storageID).Name)
		assert.Equal(t, []string{storedAccessKey + " " + storedSecretKey}, getStoredS3Keys(t, tm.app, storageID))

		backupFileName := "backup-" + uuid.NewString()
		tm.app.WriteUnpublishedFile(t, storageID, backupFileName)
		_, err := bucket.client.StatObject(t.Context(), bucket.name, backupFileName, minio.StatObjectOptions{})
		require.NoError(t, err, "the backup file must reach the bucket with the stored keys")
	})

	t.Run("returns 400 invalid_request and keeps the prefix when the S3 prefix changes", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdStorage := createStorage(t, tm.owner, apiclient.StorageCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Offsite",
			Settings: newS3Settings(t, apiclient.S3StorageSettings{
				S3Bucket:    "backups",
				S3AccessKey: new("access"),
				S3SecretKey: new("secret"),
				S3Prefix:    new("databasus/"),
			}),
		})

		response := updateStorage(t, tm.owner, createdStorage.Id, apiclient.StorageUpdate{
			Name: "Offsite",
			Settings: newS3Settings(t, apiclient.S3StorageSettings{
				S3Bucket: "backups",
				S3Prefix: new("elsewhere/"),
			}),
		})

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		readS3, err := getStorage(t, tm.owner, createdStorage.Id).Settings.AsS3StorageSettings()
		require.NoError(t, err)
		assert.Equal(t, new("databasus/"), readS3.S3Prefix)
	})
}

// storedSecretColumn is a provider column that holds a secret, with the value the user entered.
type storedSecretColumn struct {
	table     string
	column    string
	plaintext string
}

func TestStorageSecretsLifecycle(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		isAdminRequired bool
		createSettings  func(t *testing.T) apiclient.StorageSettings
		updateSettings  func(t *testing.T) apiclient.StorageSettings
		secretColumns   []storedSecretColumn
	}{
		{
			name: "S3",
			createSettings: func(t *testing.T) apiclient.StorageSettings {
				t.Helper()

				return newS3Settings(t, apiclient.S3StorageSettings{
					S3Bucket:    "test-bucket",
					S3Region:    new("us-east-1"),
					S3AccessKey: new("original-access-key"),
					S3SecretKey: new("original-secret-key"),
					S3Endpoint:  new("https://s3.amazonaws.com"),
				})
			},
			updateSettings: func(t *testing.T) apiclient.StorageSettings {
				t.Helper()

				return newS3Settings(t, apiclient.S3StorageSettings{
					S3Bucket:   "updated-bucket",
					S3Region:   new("us-west-2"),
					S3Endpoint: new("https://s3.amazonaws.com"),
				})
			},
			secretColumns: []storedSecretColumn{
				{table: "storages_s3", column: "s3_access_key", plaintext: "original-access-key"},
				{table: "storages_s3", column: "s3_secret_key", plaintext: "original-secret-key"},
			},
		},
		{
			name: "Google Drive",
			createSettings: func(t *testing.T) apiclient.StorageSettings {
				t.Helper()

				return newGoogleDriveSettings(t, apiclient.GoogleDriveStorageSettings{
					ClientId:     "client-id",
					ClientSecret: new("original-client-secret"),
					TokenJson:    new(`{"access_token":"access","refresh_token":"original-refresh"}`),
				})
			},
			updateSettings: func(t *testing.T) apiclient.StorageSettings {
				t.Helper()

				return newGoogleDriveSettings(t, apiclient.GoogleDriveStorageSettings{ClientId: "updated-client-id"})
			},
			secretColumns: []storedSecretColumn{
				{table: "storages_google_drive", column: "client_secret", plaintext: "original-client-secret"},
				{
					table:     "storages_google_drive",
					column:    "token_json",
					plaintext: `{"access_token":"access","refresh_token":"original-refresh"}`,
				},
			},
		},
		{
			name: "NAS",
			createSettings: func(t *testing.T) apiclient.StorageSettings {
				t.Helper()

				return newNasSettings(t, apiclient.NasStorageSettings{
					Host:     "nas.example.com",
					Port:     445,
					Share:    "backups",
					Username: "testuser",
					Password: new("original-password"),
					Domain:   new("WORKGROUP"),
					Path:     new("/test"),
				})
			},
			updateSettings: func(t *testing.T) apiclient.StorageSettings {
				t.Helper()

				return newNasSettings(t, apiclient.NasStorageSettings{
					Host:     "nas.example.com",
					Port:     445,
					Share:    "backups2",
					Username: "testuser2",
					Domain:   new("WORKGROUP2"),
					Path:     new("/test2"),
				})
			},
			secretColumns: []storedSecretColumn{
				{table: "storages_nas", column: "password", plaintext: "original-password"},
			},
		},
		{
			name: "Azure Blob with a connection string",
			createSettings: func(t *testing.T) apiclient.StorageSettings {
				t.Helper()

				return newAzureBlobSettings(t, apiclient.AzureBlobStorageSettings{
					AuthMethod:       apiclient.AzureBlobAuthMethodCONNECTIONSTRING,
					ConnectionString: new("original-connection-string"),
					ContainerName:    "test-container",
					Prefix:           new("backups/"),
				})
			},
			updateSettings: func(t *testing.T) apiclient.StorageSettings {
				t.Helper()

				return newAzureBlobSettings(t, apiclient.AzureBlobStorageSettings{
					AuthMethod:    apiclient.AzureBlobAuthMethodCONNECTIONSTRING,
					ContainerName: "updated-container",
					Endpoint:      new("https://custom.blob.core.windows.net"),
					Prefix:        new("backups/"),
				})
			},
			secretColumns: []storedSecretColumn{
				{table: "storages_azure_blob", column: "connection_string", plaintext: "original-connection-string"},
			},
		},
		{
			name: "Azure Blob with an account key",
			createSettings: func(t *testing.T) apiclient.StorageSettings {
				t.Helper()

				return newAzureBlobSettings(t, apiclient.AzureBlobStorageSettings{
					AuthMethod:    apiclient.AzureBlobAuthMethodACCOUNTKEY,
					AccountName:   new("testaccount"),
					AccountKey:    new("original-account-key"),
					ContainerName: "test-container",
				})
			},
			updateSettings: func(t *testing.T) apiclient.StorageSettings {
				t.Helper()

				return newAzureBlobSettings(t, apiclient.AzureBlobStorageSettings{
					AuthMethod:    apiclient.AzureBlobAuthMethodACCOUNTKEY,
					AccountName:   new("testaccount"),
					ContainerName: "updated-container",
				})
			},
			secretColumns: []storedSecretColumn{
				{table: "storages_azure_blob", column: "account_key", plaintext: "original-account-key"},
			},
		},
		{
			name: "FTP",
			createSettings: func(t *testing.T) apiclient.StorageSettings {
				t.Helper()

				return newFtpSettings(t, apiclient.FtpStorageSettings{
					Host:     "ftp.example.com",
					Port:     21,
					Username: "testuser",
					Password: new("original-password"),
					Path:     new("/backups"),
				})
			},
			updateSettings: func(t *testing.T) apiclient.StorageSettings {
				t.Helper()

				return newFtpSettings(t, apiclient.FtpStorageSettings{
					Host:     "ftp.example.com",
					Port:     21,
					Username: "testuser2",
					Path:     new("/backups2"),
				})
			},
			secretColumns: []storedSecretColumn{
				{table: "storages_ftp", column: "password", plaintext: "original-password"},
			},
		},
		{
			name: "SFTP",
			createSettings: func(t *testing.T) apiclient.StorageSettings {
				t.Helper()

				return newSftpSettings(t, apiclient.SftpStorageSettings{
					AuthMethods: []apiclient.SftpAuthMethod{
						apiclient.SftpAuthMethodPASSWORD,
						apiclient.SftpAuthMethodPRIVATEKEY,
					},
					Host:       "sftp.example.com",
					Port:       22,
					Username:   "testuser",
					Password:   new("original-password"),
					PrivateKey: new(sftpPrivateKey),
					Path:       new("/backups"),
				})
			},
			updateSettings: func(t *testing.T) apiclient.StorageSettings {
				t.Helper()

				return newSftpSettings(t, apiclient.SftpStorageSettings{
					AuthMethods: []apiclient.SftpAuthMethod{
						apiclient.SftpAuthMethodPASSWORD,
						apiclient.SftpAuthMethodPRIVATEKEY,
					},
					Host:     "sftp.example.com",
					Port:     22,
					Username: "testuser2",
					Path:     new("/backups2"),
				})
			},
			secretColumns: []storedSecretColumn{
				{table: "storages_sftp", column: "password", plaintext: "original-password"},
				{table: "storages_sftp", column: "private_key", plaintext: sftpPrivateKey},
			},
		},
		{
			name:            "rclone",
			isAdminRequired: true,
			createSettings: func(t *testing.T) apiclient.StorageSettings {
				t.Helper()

				return newRcloneSettings(t, apiclient.RcloneStorageSettings{
					ConfigContent: new(rcloneConfig),
					RemotePath:    new("/backups"),
				})
			},
			updateSettings: func(t *testing.T) apiclient.StorageSettings {
				t.Helper()

				return newRcloneSettings(t, apiclient.RcloneStorageSettings{RemotePath: new("/backups2")})
			},
			secretColumns: []storedSecretColumn{
				{table: "storages_rclone", column: "config_content", plaintext: rcloneConfig},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("keeps the %s secrets encrypted and hidden through a change without them", testCase.name),
			func(t *testing.T) {
				t.Parallel()

				tm := newTeam(t)
				author := tm.owner
				if testCase.isAdminRequired {
					author = tm.admin
				}

				createResponse, err := author.Client.CreateStorageWithResponse(t.Context(), apiclient.StorageCreate{
					WorkspaceId: tm.workspace.Id,
					Name:        "Created " + testCase.name,
					Settings:    testCase.createSettings(t),
				})
				require.NoError(t, err)
				require.Equal(t, http.StatusCreated, createResponse.StatusCode(), string(createResponse.Body))
				storageID := createResponse.JSON201.Id

				storedValues := requireSecretsStored(t, tm.app, storageID, testCase.secretColumns)
				requireSecretsHidden(t, createResponse.Body, testCase.secretColumns)

				updateResponse := updateStorage(t, author, storageID, apiclient.StorageUpdate{
					Name:     "Updated " + testCase.name,
					Settings: testCase.updateSettings(t),
				})

				require.Equal(t, http.StatusOK, updateResponse.StatusCode(), string(updateResponse.Body))
				requireSecretsHidden(t, updateResponse.Body, testCase.secretColumns)
				assert.Equal(t, storedValues, requireSecretsStored(t, tm.app, storageID, testCase.secretColumns))

				getResponse, err := author.Client.GetStorageWithResponse(t.Context(), storageID)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, getResponse.StatusCode(), string(getResponse.Body))
				assert.Equal(t, "Updated "+testCase.name, getResponse.JSON200.Name)
				requireSecretsHidden(t, getResponse.Body, testCase.secretColumns)
			})
	}
}

// requireSecretsStored checks that every secret column holds the entered value encrypted, and returns
// the stored values.
func requireSecretsStored(
	t *testing.T,
	app *apptest.App,
	storageID uuid.UUID,
	secretColumns []storedSecretColumn,
) []string {
	t.Helper()

	storedValues := make([]string, 0, len(secretColumns))

	for _, secretColumn := range secretColumns {
		var columnValues []string
		app.QuerySQL(t, &columnValues,
			"SELECT "+secretColumn.column+" FROM "+secretColumn.table+" WHERE storage_id = ?", storageID)
		require.Len(t, columnValues, 1, "%s.%s", secretColumn.table, secretColumn.column)
		require.Regexp(t, "^enc:", columnValues[0], "%s.%s", secretColumn.table, secretColumn.column)
		require.Equal(t, secretColumn.plaintext, app.DecryptSecret(t, columnValues[0]),
			"%s.%s", secretColumn.table, secretColumn.column)

		storedValues = append(storedValues, columnValues[0])
	}

	return storedValues
}

func requireSecretsHidden(t *testing.T, body []byte, secretColumns []storedSecretColumn) {
	t.Helper()

	assert.NotContains(t, string(body), "enc:")

	for _, secretColumn := range secretColumns {
		assert.NotContains(t, string(body), secretColumn.plaintext)
	}
}

type minioBucket struct {
	name        string
	endpointURL string
	client      *minio.Client
}

func startMinioBucket(t *testing.T) minioBucket {
	t.Helper()

	endpoint := containers.StartMinio(t)
	address := fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
	client, err := minio.New(address, &minio.Options{
		Creds:  credentials.NewStaticV4(containers.MinioRootUser, containers.MinioRootPassword, ""),
		Region: containers.MinioRegion,
		Secure: false,
	})
	require.NoError(t, err)

	bucketName := "storages-" + uuid.NewString()[:8]
	require.NoError(t, client.MakeBucket(t.Context(), bucketName, minio.MakeBucketOptions{
		Region: containers.MinioRegion,
	}))

	return minioBucket{name: bucketName, endpointURL: "http://" + address, client: client}
}

type storedS3Storage struct {
	bucket             minioBucket
	encryptedAccessKey string
	encryptedSecretKey string
}

// insertStoredS3Storage writes the storage in the shape the database keeps it, as a release before
// this one left it, so the test starts from keys the API never saw.
func insertStoredS3Storage(t *testing.T, tm team, stored storedS3Storage) uuid.UUID {
	t.Helper()

	storageID := uuid.New()
	tm.app.ExecuteSQL(t, "INSERT INTO storages (id, workspace_id, name, type) VALUES (?, ?, 'Offsite', 'S3')",
		storageID, tm.workspace.Id)
	tm.app.ExecuteSQL(t, `INSERT INTO storages_s3
		(storage_id, s3_bucket, s3_region, s3_access_key, s3_secret_key, s3_endpoint)
		VALUES (?, ?, ?, ?, ?, ?)`,
		storageID, stored.bucket.name, containers.MinioRegion, stored.encryptedAccessKey,
		stored.encryptedSecretKey, stored.bucket.endpointURL)

	return storageID
}

func getStoredS3Keys(t *testing.T, app *apptest.App, storageID uuid.UUID) []string {
	t.Helper()

	var storedKeys []string
	app.QuerySQL(t, &storedKeys,
		"SELECT s3_access_key || ' ' || s3_secret_key FROM storages_s3 WHERE storage_id = ?", storageID)

	return storedKeys
}

func TestSftpAuthenticationMetadata(t *testing.T) {
	t.Parallel()
	t.Run("reports private key authentication when saved credentials are hidden", func(t *testing.T) {
		t.Parallel()
		storageTeam := newTeam(t)
		createdStorage := createStorage(t, storageTeam.owner, apiclient.StorageCreate{
			WorkspaceId: storageTeam.workspace.Id,
			Name:        "Key archive",
			Settings: newSftpSettings(t, apiclient.SftpStorageSettings{
				AuthMethods: []apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPRIVATEKEY},
				Host:        "sftp.example.com", Port: 22, Username: "backup", PrivateKey: new(sftpPrivateKey),
			}),
		})

		response, err := storageTeam.owner.Client.GetStorageWithResponse(t.Context(), createdStorage.Id)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

		var storageBody struct {
			Settings map[string]any `json:"settings"`
		}
		require.NoError(t, json.Unmarshal(response.Body, &storageBody))
		assert.Equal(t, []any{"PRIVATE_KEY"}, storageBody.Settings["authMethods"])
		assert.NotContains(t, storageBody.Settings, "password")
		assert.NotContains(t, storageBody.Settings, "privateKey")
	})
}
