// Package mongodb backs up one MongoDB database with mongodump.
package mongodb

import (
	mongodbengine "databasus-backend/internal/engines/mongodb"
	"databasus-backend/internal/modules/databases/internal/providers"
)

type Settings struct {
	// Version is what the server reported at the last save.
	Version mongodbengine.Version
	Host    string
	// Port is 0 for an SRV connection, which resolves its hosts and ports from DNS.
	Port     int
	Username string
	// Password is a secret.
	Password     string
	Database     string
	AuthDatabase string
	// IsHTTPS turns TLS on without verifying the server certificate.
	IsHTTPS            bool
	IsSrv              bool
	IsDirectConnection bool
	CPUCount           int
	SSHTunnel          providers.SSHTunnelSettings
	ExcludeCollections []string
}

func (Settings) GetType() providers.Type {
	return providers.TypeMongodb
}
