package database

import "errors"

var ErrSchemaNewerThanBinary = errors.New("the internal database was migrated by a newer version")
