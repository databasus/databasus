package mysqlfamily

import (
	"errors"

	mysqldriver "github.com/go-sql-driver/mysql"
)

var ErrUserNameCollision = errors.New("failed to generate unique username")

// erCannotUser is ER_CANNOT_USER, which both servers return when CREATE USER names an account that
// already exists.
const erCannotUser = 1396

type serverErrorKind int

const (
	serverErrorOther serverErrorKind = iota
	serverErrorNameTaken
)

// classifyServerError reads what the server reported from its error number.
func classifyServerError(err error) serverErrorKind {
	mysqlErr, isMysqlErr := errors.AsType[*mysqldriver.MySQLError](err)
	if !isMysqlErr {
		return serverErrorOther
	}

	if mysqlErr.Number == erCannotUser {
		return serverErrorNameTaken
	}

	return serverErrorOther
}
