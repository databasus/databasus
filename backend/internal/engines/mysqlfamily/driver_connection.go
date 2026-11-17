package mysqlfamily

import (
	"crypto/tls"
	"database/sql"
	"net"
	"strconv"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
)

const driverConnectionTimeout = 15 * time.Second

// DriverConnection is what the Go driver connects with.
type DriverConnection struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	// IsEncrypted encrypts the connection without verifying the server certificate.
	IsEncrypted bool
	// IsCleartextPasswordAllowed lets an encrypted connection send the password to a server that
	// asks for it in clear text.
	IsCleartextPasswordAllowed bool
}

// OpenSingleSessionDB opens a pool of one connection, so every query runs in the same session. The
// driver gets its settings as a structure rather than a connection string, so no value, such as a
// database name holding "?allowAllFiles=true", can add driver parameters.
func OpenSingleSessionDB(connection DriverConnection) (*sql.DB, error) {
	connector, err := mysqldriver.NewConnector(buildDriverConfig(connection))
	if err != nil {
		return nil, err
	}

	db := sql.OpenDB(connector)
	db.SetConnMaxLifetime(driverConnectionTimeout)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	return db, nil
}

func buildDriverConfig(connection DriverConnection) *mysqldriver.Config {
	config := mysqldriver.NewConfig()
	config.User = connection.Username
	config.Passwd = connection.Password
	config.Net = "tcp"
	config.Addr = net.JoinHostPort(connection.Host, strconv.Itoa(connection.Port))
	config.DBName = connection.Database
	config.Timeout = driverConnectionTimeout
	config.ParseTime = true
	config.AllowAllFiles = false

	// The charset option only records the name, so it cannot fail.
	_ = config.Apply(mysqldriver.Charset("utf8mb4", ""))

	if connection.IsEncrypted {
		// nosemgrep: problem-based-packs.insecure-transport.go-stdlib.bypass-tls-verification.bypass-tls-verification
		config.TLS = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		}
		config.AllowCleartextPasswords = connection.IsCleartextPasswordAllowed
	}

	return config
}
