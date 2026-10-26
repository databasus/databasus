// Package sftp stores files on a server over SFTP, signing in with a password or a private key.
package sftp

import "databasus-backend/internal/modules/storages/internal/providers"

type Settings struct {
	Host     string
	Port     int
	Username string
	// Password is a secret, empty when the private key signs in.
	Password string
	// PrivateKey is a secret, empty when the password signs in.
	PrivateKey string
	// Path is the folder on the server, empty for the login folder.
	Path                string
	IsSkipHostKeyVerify bool
}

func (Settings) GetType() providers.Type {
	return providers.TypeSFTP
}
