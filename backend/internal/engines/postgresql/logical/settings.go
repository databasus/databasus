// Package logical is the logical PostgreSQL source: connecting to one database, detecting its
// version and size, checking what pg_dump would need to read, TimescaleDB and the read-only user.
// It is imported as pglogical.
package logical

import (
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/sshtunnel"
)

// Settings reach one PostgreSQL database. The password, the certificates and the tunnel secrets
// are in clear text.
type Settings struct {
	Host string
	// HostAddr is the address the connection goes to when it differs from Host, such as the local
	// end of an SSH tunnel; TLS and .pgpass still match Host.
	HostAddr string
	Port     int
	Username string
	Password string
	Database string
	TLS      pgshared.TLS
	// SSHTunnel is nil when the database is reached directly.
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
