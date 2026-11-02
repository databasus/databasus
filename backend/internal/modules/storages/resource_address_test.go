package storages_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

const (
	storedStoragePassword = "stored-storage-password"
	storedAccountKey      = "stored-account-key"
	storedS3AccessKey     = "stored-access-key"
	storedS3SecretKey     = "stored-secret-key"
	storedEndpoint        = "https://s3.example.com"
)

// addressChangeCase is a saved storage and a change of its resource address that leaves the secret
// out. The moved settings point at the listener the test watches when the provider can connect to it.
type addressChangeCase struct {
	name          string
	newStored     func(t *testing.T) apiclient.StorageSettings
	newMoved      func(t *testing.T, moved silentListener) apiclient.StorageSettings
	refusedField  string
	secretColumns []storedSecretColumn
}

func newStoredS3(t *testing.T) apiclient.StorageSettings {
	t.Helper()

	return newS3Settings(t, apiclient.S3StorageSettings{
		S3Bucket:    "backups",
		S3Endpoint:  new(storedEndpoint),
		S3AccessKey: new(storedS3AccessKey),
		S3SecretKey: new(storedS3SecretKey),
	})
}

func newStoredAccountKeyAzure(t *testing.T) apiclient.StorageSettings {
	t.Helper()

	return newAzureBlobSettings(t, apiclient.AzureBlobStorageSettings{
		AuthMethod:    apiclient.AzureBlobAuthMethodACCOUNTKEY,
		AccountName:   new("storedaccount"),
		AccountKey:    new(storedAccountKey),
		ContainerName: "backups",
		Endpoint:      new(storedEndpoint),
	})
}

func newStoredFtp(t *testing.T) apiclient.StorageSettings {
	t.Helper()

	return newFtpSettings(t, apiclient.FtpStorageSettings{
		Host:     "ftp.example.com",
		Port:     21,
		Username: "backup",
		Password: new(storedStoragePassword),
	})
}

func newStoredNas(t *testing.T) apiclient.StorageSettings {
	t.Helper()

	return newNasSettings(t, apiclient.NasStorageSettings{
		Host:     "nas.example.com",
		Port:     445,
		Share:    "backups",
		Username: "backup",
		Password: new(storedStoragePassword),
	})
}

func newStoredSftp(t *testing.T) apiclient.StorageSettings {
	t.Helper()

	return newSftpSettings(t, apiclient.SftpStorageSettings{
		AuthMethods: []apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPASSWORD, apiclient.SftpAuthMethodPRIVATEKEY},
		Host:        "sftp.example.com",
		Port:        22,
		Username:    "backup",
		Password:    new(storedStoragePassword),
		PrivateKey:  new(sftpPrivateKey),
	})
}

func newMovedSftp(t *testing.T, host string, port int, password *string) apiclient.StorageSettings {
	t.Helper()

	return newSftpSettings(t, apiclient.SftpStorageSettings{
		AuthMethods: []apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPASSWORD, apiclient.SftpAuthMethodPRIVATEKEY},
		Host:        host,
		Port:        port,
		Username:    "backup",
		Password:    password,
	})
}

func newStoredKeyOnlySftp(t *testing.T) apiclient.StorageSettings {
	t.Helper()

	return newSftpSettings(t, apiclient.SftpStorageSettings{
		AuthMethods: []apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPRIVATEKEY},
		Host:        "sftp.example.com",
		Port:        22,
		Username:    "backup",
		PrivateKey:  new(sftpPrivateKey),
	})
}

func newMovedS3(t *testing.T, endpoint string) apiclient.StorageSettings {
	t.Helper()

	return newS3Settings(t, apiclient.S3StorageSettings{S3Bucket: "backups", S3Endpoint: new(endpoint)})
}

func listAddressChangeCases() []addressChangeCase {
	s3KeyColumns := []storedSecretColumn{
		{table: "storages_s3", column: "s3_access_key", plaintext: storedS3AccessKey},
		{table: "storages_s3", column: "s3_secret_key", plaintext: storedS3SecretKey},
	}
	accountKeyColumns := []storedSecretColumn{
		{table: "storages_azure_blob", column: "account_key", plaintext: storedAccountKey},
	}
	ftpPasswordColumns := []storedSecretColumn{
		{table: "storages_ftp", column: "password", plaintext: storedStoragePassword},
	}
	nasPasswordColumns := []storedSecretColumn{
		{table: "storages_nas", column: "password", plaintext: storedStoragePassword},
	}
	sftpColumns := []storedSecretColumn{
		{table: "storages_sftp", column: "password", plaintext: storedStoragePassword},
		{table: "storages_sftp", column: "private_key", plaintext: sftpPrivateKey},
	}

	return []addressChangeCase{
		{
			name:      "the s3 endpoint",
			newStored: newStoredS3,
			newMoved: func(t *testing.T, moved silentListener) apiclient.StorageSettings {
				t.Helper()

				return newMovedS3(t, fmt.Sprintf("http://%s:%d", moved.host, moved.port))
			},
			refusedField:  "settings.s3AccessKey",
			secretColumns: s3KeyColumns,
		},
		{
			name:      "the scheme of the s3 endpoint",
			newStored: newStoredS3,
			newMoved: func(t *testing.T, _ silentListener) apiclient.StorageSettings {
				t.Helper()

				return newMovedS3(t, "http://s3.example.com")
			},
			refusedField:  "settings.s3AccessKey",
			secretColumns: s3KeyColumns,
		},
		{
			name:      "the azure endpoint",
			newStored: newStoredAccountKeyAzure,
			newMoved: func(t *testing.T, moved silentListener) apiclient.StorageSettings {
				t.Helper()

				return newAzureBlobSettings(t, apiclient.AzureBlobStorageSettings{
					AuthMethod:    apiclient.AzureBlobAuthMethodACCOUNTKEY,
					ContainerName: "backups",
					Endpoint:      new(fmt.Sprintf("http://%s:%d", moved.host, moved.port)),
				})
			},
			refusedField:  "settings.accountKey",
			secretColumns: accountKeyColumns,
		},
		{
			name:      "the azure account name",
			newStored: newStoredAccountKeyAzure,
			newMoved: func(t *testing.T, _ silentListener) apiclient.StorageSettings {
				t.Helper()

				return newAzureBlobSettings(t, apiclient.AzureBlobStorageSettings{
					AuthMethod:    apiclient.AzureBlobAuthMethodACCOUNTKEY,
					AccountName:   new("movedaccount"),
					ContainerName: "backups",
					Endpoint:      new(storedEndpoint),
				})
			},
			refusedField:  "settings.accountKey",
			secretColumns: accountKeyColumns,
		},
		{
			name:      "the ftp host",
			newStored: newStoredFtp,
			newMoved: func(t *testing.T, moved silentListener) apiclient.StorageSettings {
				t.Helper()

				return newFtpSettings(t, apiclient.FtpStorageSettings{
					Host: moved.host, Port: moved.port, Username: "backup",
				})
			},
			refusedField:  "settings.password",
			secretColumns: ftpPasswordColumns,
		},
		{
			name:      "the ftp port",
			newStored: newStoredFtp,
			newMoved: func(t *testing.T, _ silentListener) apiclient.StorageSettings {
				t.Helper()

				return newFtpSettings(t, apiclient.FtpStorageSettings{
					Host: "ftp.example.com", Port: 2121, Username: "backup",
				})
			},
			refusedField:  "settings.password",
			secretColumns: ftpPasswordColumns,
		},
		{
			name:      "the nas host",
			newStored: newStoredNas,
			newMoved: func(t *testing.T, moved silentListener) apiclient.StorageSettings {
				t.Helper()

				return newNasSettings(t, apiclient.NasStorageSettings{
					Host: moved.host, Port: moved.port, Share: "backups", Username: "backup",
				})
			},
			refusedField:  "settings.password",
			secretColumns: nasPasswordColumns,
		},
		{
			name:      "the nas port",
			newStored: newStoredNas,
			newMoved: func(t *testing.T, _ silentListener) apiclient.StorageSettings {
				t.Helper()

				return newNasSettings(t, apiclient.NasStorageSettings{
					Host: "nas.example.com", Port: 1445, Share: "backups", Username: "backup",
				})
			},
			refusedField:  "settings.password",
			secretColumns: nasPasswordColumns,
		},
		{
			name:      "the host of a key-only sftp storage",
			newStored: newStoredKeyOnlySftp,
			newMoved: func(t *testing.T, moved silentListener) apiclient.StorageSettings {
				t.Helper()

				return newSftpSettings(t, apiclient.SftpStorageSettings{
					AuthMethods: []apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPRIVATEKEY},
					Host:        moved.host,
					Port:        moved.port,
					Username:    "backup",
				})
			},
			refusedField: "settings.privateKey",
			secretColumns: []storedSecretColumn{
				{table: "storages_sftp", column: "private_key", plaintext: sftpPrivateKey},
			},
		},
		{
			name:      "the sftp host",
			newStored: newStoredSftp,
			newMoved: func(t *testing.T, moved silentListener) apiclient.StorageSettings {
				t.Helper()

				return newMovedSftp(t, moved.host, moved.port, nil)
			},
			refusedField:  "settings.password",
			secretColumns: sftpColumns,
		},
		{
			name:      "the sftp port",
			newStored: newStoredSftp,
			newMoved: func(t *testing.T, _ silentListener) apiclient.StorageSettings {
				t.Helper()

				return newMovedSftp(t, "sftp.example.com", 2222, nil)
			},
			refusedField:  "settings.password",
			secretColumns: sftpColumns,
		},
		{
			name:      "the sftp host with only the password sent",
			newStored: newStoredSftp,
			newMoved: func(t *testing.T, moved silentListener) apiclient.StorageSettings {
				t.Helper()

				return newMovedSftp(t, moved.host, moved.port, new("new-password"))
			},
			refusedField:  "settings.privateKey",
			secretColumns: sftpColumns,
		},
	}
}

func marshalStorageSettings(t *testing.T, settings apiclient.StorageSettings) string {
	t.Helper()

	marshaledSettings, err := json.Marshal(settings)
	require.NoError(t, err)

	return string(marshaledSettings)
}

func TestChangeResourceAddress(t *testing.T) {
	t.Parallel()

	for _, changeCase := range listAddressChangeCases() {
		t.Run("returns 400 invalid_request and changes nothing when a member changes "+changeCase.name,
			func(t *testing.T) {
				t.Parallel()

				tm := newTeam(t)
				member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
				moved := startSilentListener(t)
				createdStorage := createStorage(t, tm.owner, apiclient.StorageCreate{
					WorkspaceId: tm.workspace.Id,
					Name:        "Stored",
					Settings:    changeCase.newStored(t),
				})
				settingsBefore := marshalStorageSettings(t, getStorage(t, member, createdStorage.Id).Settings)
				storedValues := requireSecretsStored(t, tm.app, createdStorage.Id, changeCase.secretColumns)

				response := updateStorage(t, member, createdStorage.Id, apiclient.StorageUpdate{
					Name:     "Moved",
					Settings: changeCase.newMoved(t, moved),
				})

				requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
					apiclient.ProblemCodeInvalidRequest)
				requireFieldError(t, response.Body, changeCase.refusedField)
				storedStorage := getStorage(t, member, createdStorage.Id)
				assert.Equal(t, "Stored", storedStorage.Name)
				assert.Equal(t, settingsBefore, marshalStorageSettings(t, storedStorage.Settings))
				assert.Equal(
					t,
					storedValues,
					requireSecretsStored(t, tm.app, createdStorage.Id, changeCase.secretColumns),
				)
			})

		t.Run("returns 400 invalid_request and connects nowhere when a member tests "+changeCase.name,
			func(t *testing.T) {
				t.Parallel()

				tm := newTeam(t)
				member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
				moved := startSilentListener(t)
				createdStorage := createStorage(t, tm.owner, apiclient.StorageCreate{
					WorkspaceId: tm.workspace.Id,
					Name:        "Stored",
					Settings:    changeCase.newStored(t),
				})

				response, err := member.Client.TestStorageSettingsWithResponse(t.Context(),
					apiclient.StorageSettingsTest{
						WorkspaceId: tm.workspace.Id,
						StorageId:   &createdStorage.Id,
						Settings:    changeCase.newMoved(t, moved),
					})

				require.NoError(t, err)
				requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
					apiclient.ProblemCodeInvalidRequest)
				requireFieldError(t, response.Body, changeCase.refusedField)
				assert.False(t, moved.isConnectionPending(t))
			})
	}

	t.Run("stores the new password when it comes with the new host", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		createdStorage := createStorage(t, tm.owner, apiclient.StorageCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Stored",
			Settings:    newStoredFtp(t),
		})

		response := updateStorage(t, member, createdStorage.Id, apiclient.StorageUpdate{
			Name: "Moved",
			Settings: newFtpSettings(t, apiclient.FtpStorageSettings{
				Host: "moved.example.com", Port: 21, Username: "backup", Password: new("new-password"),
			}),
		})

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		movedFtp, err := getStorage(t, member, createdStorage.Id).Settings.AsFtpStorageSettings()
		require.NoError(t, err)
		assert.Equal(t, "moved.example.com", movedFtp.Host)
		requireSecretsStored(t, tm.app, createdStorage.Id, []storedSecretColumn{
			{table: "storages_ftp", column: "password", plaintext: "new-password"},
		})
	})

	t.Run("keeps the keys when the s3 storage class, bucket and region change at the same endpoint",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
			createdStorage := createStorage(t, tm.owner, apiclient.StorageCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Stored",
				Settings:    newStoredS3(t),
			})

			response := updateStorage(t, member, createdStorage.Id, apiclient.StorageUpdate{
				Name: "Stored",
				Settings: newS3Settings(t, apiclient.S3StorageSettings{
					S3Bucket:       "moved-bucket",
					S3Region:       new("eu-central-1"),
					S3Endpoint:     new(storedEndpoint),
					S3StorageClass: new("STANDARD_IA"),
				}),
			})

			require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
			requireSecretsStored(t, tm.app, createdStorage.Id, []storedSecretColumn{
				{table: "storages_s3", column: "s3_access_key", plaintext: storedS3AccessKey},
				{table: "storages_s3", column: "s3_secret_key", plaintext: storedS3SecretKey},
			})
		})

	t.Run("keeps the password when the ftp path changes at the same address", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdStorage := createStorage(t, tm.owner, apiclient.StorageCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Stored",
			Settings:    newStoredFtp(t),
		})

		response := updateStorage(t, tm.owner, createdStorage.Id, apiclient.StorageUpdate{
			Name: "Stored",
			Settings: newFtpSettings(t, apiclient.FtpStorageSettings{
				Host: "ftp.example.com", Port: 21, Username: "backup", Path: new("/moved"),
			}),
		})

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		requireSecretsStored(t, tm.app, createdStorage.Id, []storedSecretColumn{
			{table: "storages_ftp", column: "password", plaintext: storedStoragePassword},
		})
	})

	t.Run("keeps the account name and key when the azure container changes and the request omits them",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			createdStorage := createStorage(t, tm.owner, apiclient.StorageCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Stored",
				Settings:    newStoredAccountKeyAzure(t),
			})

			response := updateStorage(t, tm.owner, createdStorage.Id, apiclient.StorageUpdate{
				Name: "Stored",
				Settings: newAzureBlobSettings(t, apiclient.AzureBlobStorageSettings{
					AuthMethod:    apiclient.AzureBlobAuthMethodACCOUNTKEY,
					ContainerName: "moved-container",
					Endpoint:      new(storedEndpoint),
				}),
			})

			require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
			requireSecretsStored(t, tm.app, createdStorage.Id, []storedSecretColumn{
				{table: "storages_azure_blob", column: "account_key", plaintext: storedAccountKey},
			})
			storedAzure, err := getStorage(t, tm.owner, createdStorage.Id).Settings.AsAzureBlobStorageSettings()
			require.NoError(t, err)
			assert.Equal(t, new("storedaccount"), storedAzure.AccountName)
		})

	t.Run("keeps the azure connection string when the endpoint changes", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdStorage := createStorage(t, tm.owner, apiclient.StorageCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Stored",
			Settings: newAzureBlobSettings(t, apiclient.AzureBlobStorageSettings{
				AuthMethod:       apiclient.AzureBlobAuthMethodCONNECTIONSTRING,
				ConnectionString: new("stored-connection-string"),
				ContainerName:    "backups",
			}),
		})

		response := updateStorage(t, tm.owner, createdStorage.Id, apiclient.StorageUpdate{
			Name: "Stored",
			Settings: newAzureBlobSettings(t, apiclient.AzureBlobStorageSettings{
				AuthMethod:    apiclient.AzureBlobAuthMethodCONNECTIONSTRING,
				ContainerName: "backups",
				Endpoint:      new(storedEndpoint),
			}),
		})

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		requireSecretsStored(t, tm.app, createdStorage.Id, []storedSecretColumn{
			{table: "storages_azure_blob", column: "connection_string", plaintext: "stored-connection-string"},
		})
	})

	t.Run(
		"removes an azure key the settings stop using when the endpoint moves, and refuses account key at that endpoint",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			moved := startSilentListener(t)
			movedEndpoint := fmt.Sprintf("http://%s:%d", moved.host, moved.port)
			createdStorage := createStorage(t, tm.owner, apiclient.StorageCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Stored",
				Settings:    newStoredAccountKeyAzure(t),
			})

			parkResponse := updateStorage(t, tm.owner, createdStorage.Id, apiclient.StorageUpdate{
				Name: "Stored",
				Settings: newAzureBlobSettings(t, apiclient.AzureBlobStorageSettings{
					AuthMethod:       apiclient.AzureBlobAuthMethodCONNECTIONSTRING,
					ConnectionString: new("new-connection-string"),
					ContainerName:    "backups",
					Endpoint:         new(movedEndpoint),
				}),
			})
			require.Equal(t, http.StatusOK, parkResponse.StatusCode(), string(parkResponse.Body))

			returnResponse := updateStorage(t, tm.owner, createdStorage.Id, apiclient.StorageUpdate{
				Name: "Stored",
				Settings: newAzureBlobSettings(t, apiclient.AzureBlobStorageSettings{
					AuthMethod:    apiclient.AzureBlobAuthMethodACCOUNTKEY,
					ContainerName: "backups",
					Endpoint:      new(movedEndpoint),
				}),
			})

			requireProblem(t, returnResponse.StatusCode(), returnResponse.Body, http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest)
			requireFieldError(t, returnResponse.Body, "settings.accountKey")
			assert.False(t, moved.isConnectionPending(t))
		},
	)

	t.Run("removes the excluded sftp password when the host moves with a new key", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdStorage := createStorage(t, tm.owner, apiclient.StorageCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Stored",
			Settings:    newStoredSftp(t),
		})

		response := updateStorage(t, tm.owner, createdStorage.Id, apiclient.StorageUpdate{
			Name: "Stored",
			Settings: newSftpSettings(t, apiclient.SftpStorageSettings{
				AuthMethods: []apiclient.SftpAuthMethod{apiclient.SftpAuthMethodPRIVATEKEY},
				Host:        "moved.example.com",
				Port:        22,
				Username:    "backup",
				PrivateKey:  new(sftpPrivateKey),
			}),
		})

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		var storedPasswords []string
		tm.app.QuerySQL(t, &storedPasswords,
			"SELECT COALESCE(password, '') FROM storages_sftp WHERE storage_id = ?", createdStorage.Id)
		assert.Equal(t, []string{""}, storedPasswords)
	})
}
