package runner

import (
	"errors"
	"fmt"
	"strings"
)

var errAbortNoReport = errors.New("verification aborted; no report")

type unverifiedPostgresMajorError struct {
	major          string
	verifiedMajors []string
}

func (e *unverifiedPostgresMajorError) Error() string {
	return fmt.Sprintf(
		"this agent release cannot verify PostgreSQL %s backups; it verifies PostgreSQL %s. "+
			"A newer agent release may add PostgreSQL %s",
		e.major, strings.Join(e.verifiedMajors, ", "), e.major,
	)
}
