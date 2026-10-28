package catalog

type BackupEncryption string

const (
	BackupEncryptionNone      BackupEncryption = "NONE"
	BackupEncryptionEncrypted BackupEncryption = "ENCRYPTED"
)
