package shared

import "errors"

var ErrUnreadableClientVersion = errors.New("could not read a PostgreSQL version")

// ConnectionErrorCode is part of the API: renaming a value requires updating any client that
// switches on it.
type ConnectionErrorCode string

const (
	ConnErrConnectionFailed         ConnectionErrorCode = "CONNECTION_FAILED"
	ConnErrPgHbaNoEntry             ConnectionErrorCode = "PG_HBA_NO_ENTRY"
	ConnErrBadCredentials           ConnectionErrorCode = "BAD_CREDENTIALS"
	ConnErrNoReplicationPrivilege   ConnectionErrorCode = "NO_REPLICATION_PRIVILEGE"
	ConnErrWalLevelInvalid          ConnectionErrorCode = "WAL_LEVEL_INVALID"
	ConnErrNoWalSenders             ConnectionErrorCode = "NO_WAL_SENDERS"
	ConnErrNoReplicationSlots       ConnectionErrorCode = "NO_REPLICATION_SLOTS"
	ConnErrWalSummaryDisabled       ConnectionErrorCode = "WAL_SUMMARY_DISABLED"
	ConnErrCustomTablespaces        ConnectionErrorCode = "CUSTOM_TABLESPACES"
	ConnErrSystemIdentifierMismatch ConnectionErrorCode = "SYSTEM_IDENTIFIER_MISMATCH"
	ConnErrNoWalSwitchPrivilege     ConnectionErrorCode = "NO_WAL_SWITCH_PRIVILEGE"
)

// ConnectionTestError carries a machine-readable Code for a classified connection failure. The
// client keys its error card off Code; Message is a human-readable fallback for paths that surface
// only Error(), so they do not show a bare code.
type ConnectionTestError struct {
	Code    ConnectionErrorCode `json:"code"`
	Message string              `json:"message,omitempty"`
}

func (e *ConnectionTestError) Error() string {
	if e.Message != "" {
		return e.Message
	}

	return string(e.Code)
}
