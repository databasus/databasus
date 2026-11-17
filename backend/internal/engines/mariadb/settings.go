package mariadb

import (
	"databasus-backend/internal/engines/mysqlfamily"
	"databasus-backend/internal/lib/sshtunnel"
)

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

// Validate refuses a value that the client tools or the driver could read as more than a value,
// with a mysqlfamily.InvalidSettingError naming the setting.
func (s Settings) Validate() error {
	return mysqlfamily.ConnectionValues{
		Host:     s.Host,
		Username: s.Username,
		Password: s.Password,
		Database: s.Database,
	}.Validate()
}
