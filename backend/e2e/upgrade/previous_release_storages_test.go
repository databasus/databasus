package upgrade

import (
	"context"
	"fmt"
	"maps"

	"databasus-backend/e2e/upgrade/upgradecheck"
)

// This file speaks the storage API of the previous release, like previous_release_client_test.go.

// previousStorageRequest carries the settings of one storage type, in the field the previous release
// names after that type.
type previousStorageRequest struct {
	WorkspaceID        string                      `json:"workspaceId"`
	Type               string                      `json:"type"`
	Name               string                      `json:"name"`
	LocalStorage       *struct{}                   `json:"localStorage,omitempty"`
	S3Storage          *previousS3Request          `json:"s3Storage,omitempty"`
	GoogleDriveStorage *previousGoogleDriveRequest `json:"googleDriveStorage,omitempty"`
	NASStorage         *previousNASRequest         `json:"nasStorage,omitempty"`
	AzureBlobStorage   *previousAzureBlobRequest   `json:"azureBlobStorage,omitempty"`
	FTPStorage         *previousFTPRequest         `json:"ftpStorage,omitempty"`
	SFTPStorage        *previousSFTPRequest        `json:"sftpStorage,omitempty"`
	RcloneStorage      *previousRcloneRequest      `json:"rcloneStorage,omitempty"`
}

type previousS3Request struct {
	S3Bucket                string `json:"s3Bucket"`
	S3Region                string `json:"s3Region"`
	S3AccessKey             string `json:"s3AccessKey"`
	S3SecretKey             string `json:"s3SecretKey"`
	S3Endpoint              string `json:"s3Endpoint"`
	S3Prefix                string `json:"s3Prefix"`
	S3UseVirtualHostedStyle bool   `json:"s3UseVirtualHostedStyle"`
	SkipTLSVerify           bool   `json:"skipTLSVerify"`
	S3StorageClass          string `json:"s3StorageClass"`
}

type previousGoogleDriveRequest struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	TokenJSON    string `json:"tokenJson"`
}

type previousNASRequest struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Share    string `json:"share"`
	Username string `json:"username"`
	Password string `json:"password"`
	UseSSL   bool   `json:"useSsl"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
}

type previousAzureBlobRequest struct {
	AuthMethod    string `json:"authMethod"`
	AccountName   string `json:"accountName"`
	AccountKey    string `json:"accountKey"`
	ContainerName string `json:"containerName"`
	Endpoint      string `json:"endpoint"`
	Prefix        string `json:"prefix"`
}

type previousFTPRequest struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Path          string `json:"path"`
	UseSSL        bool   `json:"useSsl"`
	SkipTLSVerify bool   `json:"skipTlsVerify"`
}

type previousSFTPRequest struct {
	Host              string `json:"host"`
	Port              int    `json:"port"`
	Username          string `json:"username"`
	Password          string `json:"password"`
	PrivateKey        string `json:"privateKey"`
	Path              string `json:"path"`
	SkipHostKeyVerify bool   `json:"skipHostKeyVerify"`
}

type previousRcloneRequest struct {
	ConfigContent string `json:"configContent"`
	RemotePath    string `json:"remotePath"`
}

// writtenStorage is a storage the previous release saved: its request and the values the current
// code is expected to list for it. Secrets are not among the values, because the current code never
// returns them.
type writtenStorage struct {
	request        previousStorageRequest
	expectedValues map[string]string
}

func newWrittenStorages(workspaceID string) []writtenStorage {
	writtenStorages := []writtenStorage{
		{
			request: previousStorageRequest{
				WorkspaceID: workspaceID, Type: "LOCAL", Name: storageName, LocalStorage: &struct{}{},
			},
			expectedValues: map[string]string{"type": "LOCAL"},
		},
		{
			request: previousStorageRequest{
				WorkspaceID: workspaceID, Type: "S3", Name: "upgrade s3",
				S3Storage: &previousS3Request{
					S3Bucket:                "upgrade-bucket",
					S3Region:                "eu-central-1",
					S3AccessKey:             "upgrade-access-key",
					S3SecretKey:             "upgrade-secret-key",
					S3Endpoint:              "https://s3.upgrade.invalid",
					S3Prefix:                "upgrade/backups",
					S3UseVirtualHostedStyle: true,
					SkipTLSVerify:           true,
					S3StorageClass:          "STANDARD_IA",
				},
			},
			expectedValues: map[string]string{
				"type":                 "S3",
				"s3Bucket":             "upgrade-bucket",
				"s3Region":             "eu-central-1",
				"s3Endpoint":           "https://s3.upgrade.invalid",
				"s3Prefix":             "upgrade/backups",
				"isVirtualHostedStyle": "true",
				"isSkipTlsVerify":      "true",
				"s3StorageClass":       "STANDARD_IA",
			},
		},
		{
			request: previousStorageRequest{
				WorkspaceID: workspaceID, Type: "GOOGLE_DRIVE", Name: "upgrade google drive",
				GoogleDriveStorage: &previousGoogleDriveRequest{
					ClientID:     "upgrade-client-id.apps.googleusercontent.com",
					ClientSecret: "upgrade-client-secret",
					TokenJSON: `{"access_token":"upgrade-access-token","token_type":"Bearer",` +
						`"refresh_token":"upgrade-refresh-token","expiry":"2030-01-01T00:00:00Z"}`,
				},
			},
			expectedValues: map[string]string{
				"type":     "GOOGLE_DRIVE",
				"clientId": "upgrade-client-id.apps.googleusercontent.com",
			},
		},
		{
			request: previousStorageRequest{
				WorkspaceID: workspaceID, Type: "NAS", Name: "upgrade nas",
				NASStorage: &previousNASRequest{
					Host:     "nas.upgrade.invalid",
					Port:     4455,
					Share:    "upgrade-share",
					Username: "upgrade-nas-user",
					Password: "upgrade-nas-password",
					UseSSL:   true,
					Domain:   "UPGRADE",
					Path:     "databasus/backups",
				},
			},
			expectedValues: map[string]string{
				"type":     "NAS",
				"host":     "nas.upgrade.invalid",
				"port":     "4455",
				"share":    "upgrade-share",
				"username": "upgrade-nas-user",
				"isSsl":    "true",
				"domain":   "UPGRADE",
				"path":     "databasus/backups",
			},
		},
		{
			request: previousStorageRequest{
				WorkspaceID: workspaceID, Type: "AZURE_BLOB", Name: "upgrade azure blob",
				AzureBlobStorage: &previousAzureBlobRequest{
					AuthMethod:    "ACCOUNT_KEY",
					AccountName:   "upgradeaccount",
					AccountKey:    "upgrade-account-key",
					ContainerName: "upgrade-container",
					Endpoint:      "https://upgradeaccount.blob.upgrade.invalid",
					Prefix:        "upgrade/blobs",
				},
			},
			expectedValues: map[string]string{
				"type":          "AZURE_BLOB",
				"authMethod":    "ACCOUNT_KEY",
				"accountName":   "upgradeaccount",
				"containerName": "upgrade-container",
				"endpoint":      "https://upgradeaccount.blob.upgrade.invalid",
				"prefix":        "upgrade/blobs",
			},
		},
		{
			request: previousStorageRequest{
				WorkspaceID: workspaceID, Type: "FTP", Name: "upgrade ftp",
				FTPStorage: &previousFTPRequest{
					Host:          "ftp.upgrade.invalid",
					Port:          2121,
					Username:      "upgrade-ftp-user",
					Password:      "upgrade-ftp-password",
					Path:          "/upgrade/ftp",
					UseSSL:        true,
					SkipTLSVerify: true,
				},
			},
			expectedValues: map[string]string{
				"type":            "FTP",
				"host":            "ftp.upgrade.invalid",
				"port":            "2121",
				"username":        "upgrade-ftp-user",
				"path":            "/upgrade/ftp",
				"isSsl":           "true",
				"isSkipTlsVerify": "true",
			},
		},
		{
			request: previousStorageRequest{
				WorkspaceID: workspaceID, Type: "SFTP", Name: "upgrade sftp",
				SFTPStorage: &previousSFTPRequest{
					Host:              "sftp.upgrade.invalid",
					Port:              2222,
					Username:          "upgrade-sftp-user",
					Password:          "upgrade-sftp-password",
					PrivateKey:        "upgrade-sftp-private-key",
					Path:              "/upgrade/sftp",
					SkipHostKeyVerify: true,
				},
			},
			expectedValues: map[string]string{
				"type":                "SFTP",
				"host":                "sftp.upgrade.invalid",
				"port":                "2222",
				"username":            "upgrade-sftp-user",
				"path":                "/upgrade/sftp",
				"isSkipHostKeyVerify": "true",
			},
		},
		{
			request: previousStorageRequest{
				WorkspaceID: workspaceID, Type: "RCLONE", Name: "upgrade rclone",
				RcloneStorage: &previousRcloneRequest{
					ConfigContent: "[upgrade]\ntype = s3\nprovider = Minio\naccess_key_id = upgrade\n" +
						"secret_access_key = upgrade-secret\nendpoint = https://minio.upgrade.invalid\n",
					RemotePath: "upgrade-bucket/rclone",
				},
			},
			expectedValues: map[string]string{
				"type":       "RCLONE",
				"remotePath": "upgrade-bucket/rclone",
			},
		},
	}

	for _, storageClass := range []string{"COLD", "DEEP_ARCHIVE"} {
		storage := writtenStorages[1]
		s3Settings := *storage.request.S3Storage
		s3Settings.S3StorageClass = storageClass
		storage.request.S3Storage = &s3Settings
		storage.request.Name = "upgrade s3 " + storageClass
		storage.expectedValues = maps.Clone(storage.expectedValues)
		storage.expectedValues["s3StorageClass"] = storageClass
		writtenStorages = append(writtenStorages, storage)
	}

	return writtenStorages
}

// createStorages saves one storage of every type with every setting filled in. It returns the items
// to find after the upgrade and the ID of the local storage, which the backup config uses.
func createStorages(ctx context.Context, api *apiClient, workspaceID string) ([]upgradecheck.Item, string, error) {
	writtenStorages := newWrittenStorages(workspaceID)
	storageItems := make([]upgradecheck.Item, 0, len(writtenStorages))

	var localStorageID string

	for _, storage := range writtenStorages {
		var createdStorage previousCreatedItem

		err := api.post(ctx, "/storages", storage.request, &createdStorage)
		if err != nil {
			return nil, "", fmt.Errorf("create the storage %q: %w", storage.request.Name, err)
		}

		if storage.request.S3Storage != nil {
			var savedStorage struct {
				S3Storage previousS3Request `json:"s3Storage"`
			}
			if err := api.get(ctx, "/storages/"+createdStorage.ID, &savedStorage); err != nil {
				return nil, "", fmt.Errorf("read the previous storage %q: %w", storage.request.Name, err)
			}
			if savedStorage.S3Storage.S3StorageClass != storage.request.S3Storage.S3StorageClass {
				return nil, "", fmt.Errorf("previous storage %q did not preserve its class", storage.request.Name)
			}
		}

		if storage.request.LocalStorage != nil {
			localStorageID = createdStorage.ID
		}

		storageValues := map[string]string{"id": createdStorage.ID}
		maps.Copy(storageValues, storage.expectedValues)

		storageItems = append(storageItems, upgradecheck.Item{
			Kind: "storage", Name: storage.request.Name, Values: storageValues,
		})
	}

	return storageItems, localStorageID, nil
}
