package mariadb

import "databasus-backend/internal/lib/sshtunnel"

// Settings reach one MariaDB database. The password and the tunnel secrets are in clear text.
type Settings struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	// IsHTTPS encrypts the connection without verifying the server certificate.
	IsHTTPS bool
	// SSHTunnel is nil when the database is reached directly.
	SSHTunnel *sshtunnel.Settings
}
