package mysqlfamily

import (
	"errors"

	mysqldriver "github.com/go-sql-driver/mysql"
)

var (
	ErrUserNameCollision              = errors.New("failed to generate unique username")
	ErrControlCharacterInClientConfig = errors.New(
		"the host, user name and password must not contain a line break or another control character",
	)
)

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

// InvalidSettingError names a connection setting that the client tools or the driver could read as
// more than a value. Setting is the setting's name in the API.
type InvalidSettingError struct {
	Setting string
	Reason  string
}

func (e *InvalidSettingError) Error() string {
	return e.Setting + " " + e.Reason
}
