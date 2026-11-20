// Package internaldb dumps and restores Databasus's own internal database. It is separate from the
// logical engine on purpose: user databases keep refusing the embedded target, and only this
// narrowly scoped operation reads it, joined to an exported snapshot and without ownership or
// privileges, so the archive restores into a database of another installation.
package internaldb

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// Connection reaches the internal PostgreSQL server. Tools receive it through libpq variables, so
// the password never appears in their arguments.
type Connection struct {
	Host     string
	Port     uint16
	User     string
	Password string
	Database string
}

// ParseConnection reads the application's DSN.
func ParseConnection(dsn string) (Connection, error) {
	parsed, err := pgconn.ParseConfig(dsn)
	if err != nil {
		return Connection{}, fmt.Errorf("parse the internal database DSN: %w", err)
	}

	return Connection{
		Host:     parsed.Host,
		Port:     parsed.Port,
		User:     parsed.User,
		Password: parsed.Password,
		Database: parsed.Database,
	}, nil
}

// WithDatabase returns the same server and credentials for another database, such as the staging
// database a recovery imports into.
func (c Connection) WithDatabase(database string) Connection {
	c.Database = database

	return c
}

// ConnString is the libpq connection string of the connection, for a pgx or GORM pool.
func (c Connection) ConnString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		quoteConnValue(c.Host), c.Port, quoteConnValue(c.User), quoteConnValue(c.Password), quoteConnValue(c.Database))
}

func (c Connection) toEnv() []string {
	return []string{
		"PGHOST=" + c.Host,
		"PGPORT=" + strconv.Itoa(int(c.Port)),
		"PGUSER=" + c.User,
		"PGPASSWORD=" + c.Password,
		"PGDATABASE=" + c.Database,
		"PGSSLMODE=disable",
		"PGCLIENTENCODING=UTF8",
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	}
}

func quoteConnValue(value string) string {
	var escaped strings.Builder

	escaped.WriteByte('\'')

	for _, character := range value {
		if character == '\\' || character == '\'' {
			escaped.WriteByte('\\')
		}

		escaped.WriteRune(character)
	}

	escaped.WriteByte('\'')

	return escaped.String()
}
