// Package s3 stores files in a bucket of S3 or an S3-compatible server, splitting a large file into
// several objects described by a manifest.
package s3

import "databasus-backend/internal/modules/storages/internal/providers"

type Settings struct {
	Bucket string
	// Region is empty when the client looks the region of the bucket up.
	Region string
	// AccessKey is a secret.
	AccessKey string
	// SecretKey is a secret.
	SecretKey string
	// Endpoint is empty for AWS, and may carry an http:// or https:// scheme.
	Endpoint string
	// Prefix is fixed once the storage exists, because every stored object is named under it.
	Prefix               string
	IsVirtualHostedStyle bool
	IsSkipTLSVerify      bool
	StorageClass         string
}

func (Settings) GetType() providers.Type {
	return providers.TypeS3
}
