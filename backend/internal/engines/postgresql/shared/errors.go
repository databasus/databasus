package shared

import "errors"

var ErrUnreadableClientVersion = errors.New("could not read a PostgreSQL version")

// ConnectionErrorCode is part of the API: renaming a value requires updating any client that
// switches on it.
type ConnectionErrorCode string

const (
	ConnErrConnectionFailed         ConnectionErrorCode = "connection_failed"
	ConnErrPgHbaNoEntry             ConnectionErrorCode = "pg_hba_no_entry"
	ConnErrBadCredentials           ConnectionErrorCode = "bad_credentials"
	ConnErrNoReplicationPrivilege   ConnectionErrorCode = "no_replication_privilege"
	ConnErrWalLevelInvalid          ConnectionErrorCode = "wal_level_invalid"
	ConnErrNoWalSenders             ConnectionErrorCode = "no_wal_senders"
	ConnErrNoReplicationSlots       ConnectionErrorCode = "no_replication_slots"
	ConnErrWalSummaryDisabled       ConnectionErrorCode = "wal_summary_disabled"
	ConnErrCustomTablespaces        ConnectionErrorCode = "custom_tablespaces"
	ConnErrSystemIdentifierMismatch ConnectionErrorCode = "system_identifier_mismatch"
	ConnErrNoWalSwitchPrivilege     ConnectionErrorCode = "no_wal_switch_privilege"
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
