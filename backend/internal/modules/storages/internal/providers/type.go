package providers

// Type is the kind of storage a provider serves. It is fixed when the storage is created.
type Type string

const (
	TypeLocal     Type = "LOCAL"
	TypeS3        Type = "S3"
	TypeNAS       Type = "NAS"
	TypeAzureBlob Type = "AZURE_BLOB"
	TypeFTP       Type = "FTP"
	TypeSFTP      Type = "SFTP"
)
