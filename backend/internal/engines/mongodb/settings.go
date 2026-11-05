package mongodb

import (
	"fmt"
	"net/url"

	"databasus-backend/internal/lib/sshtunnel"
)

const (
	defaultPort             = 27017
	defaultAuthDatabaseName = "admin"
)

// Settings reach one MongoDB database. The password and the tunnel secrets are in clear text.
type Settings struct {
	Host string
	// Port is 0 when the URI falls back to MongoDB's default port; an SRV connection never uses it.
	Port         int
	Username     string
	Password     string
	Database     string
	AuthDatabase string
	// IsTLS turns TLS on without verifying the server certificate.
	IsTLS              bool
	IsSrv              bool
	IsDirectConnection bool
	// SSHTunnel is nil when the database is reached directly.
	SSHTunnel *sshtunnel.Settings
}

// BuildConnectionURI builds a MongoDB connection URI with the database embedded in the path. Used by
// the Go driver and the bundled mongodump invocation: embedding the database in the URI lets us drop
// `--db`, which stricter mongodump versions reject when combined with `--uri`.
func (s Settings) BuildConnectionURI() string {
	return s.buildURI(true)
}

// BuildRestoreURI builds a connection URI without the database in the path. mongorestore treats a
// URI-embedded database as an implicit target rewrite that collides with --nsFrom / --nsTo, silently
// producing a zero-document restore. Restore must use a database-less URI and rely on the namespace
// flags exclusively.
func (s Settings) BuildRestoreURI() string {
	return s.buildURI(false)
}

func (s Settings) buildURI(isDatabaseInPath bool) string {
	extraParams := ""
	if s.IsTLS {
		extraParams += "&tls=true&tlsInsecure=true"
	} else {
		extraParams += "&tls=false"
	}

	if s.IsDirectConnection {
		extraParams += "&directConnection=true"
	}

	databasePath := ""
	if isDatabaseInPath {
		databasePath = url.PathEscape(s.Database)
	}

	if s.IsSrv {
		return fmt.Sprintf(
			"mongodb+srv://%s:%s@%s/%s?authSource=%s&connectTimeoutMS=15000%s",
			url.QueryEscape(s.Username),
			url.QueryEscape(s.Password),
			s.Host,
			databasePath,
			url.QueryEscape(s.getAuthDatabaseName()),
			extraParams,
		)
	}

	return fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/%s?authSource=%s&connectTimeoutMS=15000%s",
		url.QueryEscape(s.Username),
		url.QueryEscape(s.Password),
		s.Host,
		s.getPort(),
		databasePath,
		url.QueryEscape(s.getAuthDatabaseName()),
		extraParams,
	)
}

func (s Settings) getPort() int {
	if s.Port == 0 {
		return defaultPort
	}

	return s.Port
}

func (s Settings) getAuthDatabaseName() string {
	if s.AuthDatabase == "" {
		return defaultAuthDatabaseName
	}

	return s.AuthDatabase
}
