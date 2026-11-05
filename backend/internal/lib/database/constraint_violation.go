package database

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// foreignKeyViolationCode is the SQLSTATE of a row that another table still references.
const foreignKeyViolationCode = "23503"

// IsForeignKeyViolation reports whether err is PostgreSQL refusing a write because of a foreign
// key, such as deleting a row that another table still references.
func IsForeignKeyViolation(err error) bool {
	var pgError *pgconn.PgError

	return errors.As(err, &pgError) && pgError.Code == foreignKeyViolationCode
}

// uniqueViolationCode is the SQLSTATE of a row that a unique constraint or index refuses.
const uniqueViolationCode = "23505"

// IsUniqueViolation reports whether err is PostgreSQL refusing a write because of the named unique
// constraint or index, such as a partial unique index that allows one running backup per database.
func IsUniqueViolation(err error, constraintName string) bool {
	var pgError *pgconn.PgError

	return errors.As(err, &pgError) && pgError.Code == uniqueViolationCode && pgError.ConstraintName == constraintName
}
