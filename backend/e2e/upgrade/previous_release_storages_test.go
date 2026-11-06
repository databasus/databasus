package upgrade

import (
	"context"
	"fmt"
	"maps"

	"databasus-backend/e2e/upgrade/upgradecheck"
)

// This file speaks the storage API of the previous release, like previous_release_client_test.go.

type previousStorageRequest struct {
	WorkspaceID string `json:"workspaceId"`
	Name        string `json:"name"`
	Settings    any    `json:"settings"`
}

type previousLocalRequest struct {
	Type string `json:"type"`
}

type previousS3Request struct {
	Type                    string `json:"type"`
	S3Bucket                string `json:"s3Bucket"`
	S3Region                string `json:"s3Region"`
	S3AccessKey             string `json:"s3AccessKey"`
	S3SecretKey             string `json:"s3SecretKey"`
	S3Endpoint              string `json:"s3Endpoint"`
	S3Prefix                string `json:"s3Prefix"`
	S3UseVirtualHostedStyle bool   `json:"isVirtualHostedStyle"`
	SkipTLSVerify           bool   `json:"isSkipTlsVerify"`
	S3StorageClass          string `json:"s3StorageClass"`
}

type previousGoogleDriveRequest struct {
	Type         string `json:"type"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	TokenJSON    string `json:"tokenJson"`
}

type previousNASRequest struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Share    string `json:"share"`
	Username string `json:"username"`
	Password string `json:"password"`
	UseSSL   bool   `json:"isSsl"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
}

type previousAzureBlobRequest struct {
	Type          string `json:"type"`
	AuthMethod    string `json:"authMethod"`
	AccountName   string `json:"accountName"`
	AccountKey    string `json:"accountKey"`
	ContainerName string `json:"containerName"`
	Endpoint      string `json:"endpoint"`
	Prefix        string `json:"prefix"`
}

type previousFTPRequest struct {
	Type          string `json:"type"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Path          string `json:"path"`
	UseSSL        bool   `json:"isSsl"`
	SkipTLSVerify bool   `json:"isSkipTlsVerify"`
}

type previousSFTPRequest struct {
	AuthMethods       []string `json:"authMethods"`
	Type              string   `json:"type"`
	Host              string   `json:"host"`
	Port              int      `json:"port"`
	Username          string   `json:"username"`
	Password          string   `json:"password"`
	PrivateKey        string   `json:"privateKey"`
	Path              string   `json:"path"`
	SkipHostKeyVerify bool     `json:"isSkipHostKeyVerify"`
}

type previousRcloneRequest struct {
	Type          string `json:"type"`
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
				WorkspaceID: workspaceID, Name: storageName, Settings: &previousLocalRequest{Type: "LOCAL"},
			},
			expectedValues: map[string]string{"type": "LOCAL"},
		},
		{
			request: previousStorageRequest{
				WorkspaceID: workspaceID, Name: "upgrade s3", Settings: &previousS3Request{
					Type:                    "S3",
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
				WorkspaceID: workspaceID, Name: "upgrade google drive", Settings: &previousGoogleDriveRequest{
					Type:         "GOOGLE_DRIVE",
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
				WorkspaceID: workspaceID, Name: "upgrade nas", Settings: &previousNASRequest{
					Type:     "NAS",
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
				WorkspaceID: workspaceID, Name: "upgrade azure blob", Settings: &previousAzureBlobRequest{
					Type:          "AZURE_BLOB",
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
				WorkspaceID: workspaceID, Name: "upgrade ftp", Settings: &previousFTPRequest{
					Type:          "FTP",
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
				WorkspaceID: workspaceID, Name: "upgrade sftp", Settings: &previousSFTPRequest{
					Type: "SFTP", AuthMethods: []string{"PASSWORD", "PRIVATE_KEY"},
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
				WorkspaceID: workspaceID, Name: "upgrade rclone", Settings: &previousRcloneRequest{
					Type: "RCLONE",
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
		sourceS3Settings, isS3 := storage.request.Settings.(*previousS3Request)
		if !isS3 {
			panic("the storage-class fixture must be S3")
		}
		s3Settings := *sourceS3Settings
		s3Settings.S3StorageClass = storageClass
		storage.request.Settings = &s3Settings
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

		if s3Settings, isS3 := storage.request.Settings.(*previousS3Request); isS3 {
			var savedStorage struct {
				Settings previousS3Request `json:"settings"`
			}
			if err := api.get(ctx, "/storages/"+createdStorage.ID, &savedStorage); err != nil {
				return nil, "", fmt.Errorf("read the previous storage %q: %w", storage.request.Name, err)
			}
			if savedStorage.Settings.S3StorageClass != s3Settings.S3StorageClass {
				return nil, "", fmt.Errorf("previous storage %q did not preserve its class", storage.request.Name)
			}
		}

		if _, isLocal := storage.request.Settings.(*previousLocalRequest); isLocal {
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
