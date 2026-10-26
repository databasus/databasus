// Package nas stores files on an SMB share of a network attached storage.
package nas

import "databasus-backend/internal/modules/storages/internal/providers"

type Settings struct {
	Host     string
	Port     int
	Share    string
	Username string
	// Password is a secret.
	Password string
	IsSSL    bool
	// Domain is empty for an account outside a domain.
	Domain string
	// Path is the folder inside the share, empty for its root.
	Path string
}

func (Settings) GetType() providers.Type {
	return providers.TypeNAS
}
