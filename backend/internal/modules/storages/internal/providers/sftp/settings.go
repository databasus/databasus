// Package sftp stores files on a server over SFTP, signing in with a password or a private key.
package sftp

import "databasus-backend/internal/modules/storages/internal/providers"

type AuthMethod string

const (
	AuthMethodPassword   AuthMethod = "PASSWORD"
	AuthMethodPrivateKey AuthMethod = "PRIVATE_KEY"
)

type Settings struct {
	Host                string
	Port                int
	Username            string
	AuthMethods         []AuthMethod
	Password            string
	PrivateKey          string
	Path                string
	IsSkipHostKeyVerify bool
}

func (Settings) GetType() providers.Type {
	return providers.TypeSFTP
}
