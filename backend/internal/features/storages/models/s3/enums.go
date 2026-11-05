package s3_storage

type S3AuthType string

const (
	S3AuthTypeKeys    S3AuthType = "KEYS"
	S3AuthTypeIAMRole S3AuthType = "IAM_ROLE"
)

type S3StorageClass string

const (
	S3StorageClassDefault            S3StorageClass = ""
	S3StorageClassStandard           S3StorageClass = "STANDARD"
	S3StorageClassStandardIA         S3StorageClass = "STANDARD_IA"
	S3StorageClassOnezoneIA          S3StorageClass = "ONEZONE_IA"
	S3StorageClassIntelligentTiering S3StorageClass = "INTELLIGENT_TIERING"
	S3StorageClassReducedRedundancy  S3StorageClass = "REDUCED_REDUNDANCY"
	S3StorageClassGlacierIR          S3StorageClass = "GLACIER_IR"
)
