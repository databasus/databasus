package internaldb

import "errors"

var (
	ErrDumpFailed          = errors.New("pg_dump of the internal database failed")
	ErrRestoreFailed       = errors.New("pg_restore into the staging database failed")
	ErrStagingNameInvalid  = errors.New("a staging database name must start with databasus_staging_")
	ErrSnapshotIDRequired  = errors.New("a dump of the internal database needs an exported snapshot")
	ErrPreviousNameInvalid = errors.New(
		"a replaced database name must start with databasus_previous_ and be renamed from another database",
	)
)
