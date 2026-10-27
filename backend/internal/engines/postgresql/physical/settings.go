// Package physical is the physical PostgreSQL source: the replication connection and its
// readiness, physical base backup streams and manifests, WAL receiving, cluster inspection,
// replication slots, forced WAL rotation and replication users. It is imported as pgphysical.
package physical

import (
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/sshtunnel"
)

// Settings reach one PostgreSQL cluster. The password, the certificates and the tunnel secrets are
// in clear text.
type Settings struct {
	Host string
	// HostAddr is the address the connection goes to when it differs from Host, such as the local
	// end of an SSH tunnel; TLS and .pgpass still match Host.
	HostAddr string
	Port     int
	Username string
	Password string
	TLS      pgshared.TLS
	// SSHTunnel is nil when the cluster is reached directly.
	SSHTunnel *sshtunnel.Settings
}

func (s Settings) GetCredentialSpec() pgshared.CredentialSpec {
	return pgshared.CredentialSpec{
		Host:     s.Host,
		HostAddr: s.HostAddr,
		Port:     s.Port,
		Username: s.Username,
		TLS:      s.TLS,
	}
}
