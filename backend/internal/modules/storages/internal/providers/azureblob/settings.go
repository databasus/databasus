// Package azureblob stores files as block blobs in a container of Azure Blob Storage.
package azureblob

import "databasus-backend/internal/modules/storages/internal/providers"

type Settings struct {
	AuthMethod AuthMethod
	// ConnectionString is a secret, used with AuthMethodConnectionString.
	ConnectionString string
	// AccountName is used with AuthMethodAccountKey.
	AccountName string
	// AccountKey is a secret, used with AuthMethodAccountKey.
	AccountKey    string
	ContainerName string
	// Endpoint is empty for the account's default blob endpoint.
	Endpoint string
	// Prefix is fixed once the storage exists, because every stored blob is named under it.
	Prefix string
}

func (Settings) GetType() providers.Type {
	return providers.TypeAzureBlob
}

// AuthMethod is how the provider signs in to the account.
type AuthMethod string

const (
	AuthMethodConnectionString AuthMethod = "CONNECTION_STRING"
	AuthMethodAccountKey       AuthMethod = "ACCOUNT_KEY"
)
