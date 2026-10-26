// Package ftp stores files on an FTP server, optionally over explicit TLS.
package ftp

import "databasus-backend/internal/modules/storages/internal/providers"

type Settings struct {
	Host     string
	Port     int
	Username string
	// Password is a secret.
	Password string
	// Path is the folder on the server, empty for the login folder.
	Path            string
	IsSSL           bool
	IsSkipTLSVerify bool
}

func (Settings) GetType() providers.Type {
	return providers.TypeFTP
}
