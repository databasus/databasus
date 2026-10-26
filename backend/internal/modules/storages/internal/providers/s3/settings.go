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
	StorageClass         StorageClass
}

func (Settings) GetType() providers.Type {
	return providers.TypeS3
}

// StorageClass is the class objects are written with. StorageClassDefault leaves the choice to the
// bucket.
type StorageClass string

const (
	StorageClassDefault            StorageClass = ""
	StorageClassStandard           StorageClass = "STANDARD"
	StorageClassStandardIA         StorageClass = "STANDARD_IA"
	StorageClassOnezoneIA          StorageClass = "ONEZONE_IA"
	StorageClassIntelligentTiering StorageClass = "INTELLIGENT_TIERING"
	StorageClassReducedRedundancy  StorageClass = "REDUCED_REDUNDANCY"
	StorageClassGlacierIR          StorageClass = "GLACIER_IR"
)

func (c StorageClass) isKnown() bool {
	switch c {
	case StorageClassDefault, StorageClassStandard, StorageClassStandardIA, StorageClassOnezoneIA,
		StorageClassIntelligentTiering, StorageClassReducedRedundancy, StorageClassGlacierIR:
		return true
	}

	return false
}
