package physical

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrSlotNameTaken       = errors.New("replication slot name is taken")
	ErrWalSwitchDenied     = errors.New("WAL switch permission denied")
	ErrUnsupportedVersion  = errors.New("physical backup requires PostgreSQL 17 or 18")
	ErrUnreadableVersion   = errors.New("could not parse version")
	ErrSlotNameEmpty       = errors.New("replication slot name is empty; row corruption or BeforeCreate bypassed")
	ErrCannotCreateRoles   = errors.New("current user cannot create roles — connect as a platform admin")
	ErrUserNameCollision   = errors.New("failed to generate unique username")
	ErrAzureReplicationOff = errors.New(
		"replication must be enabled at the server level in the Azure portal (Server parameters → wal_level=logical → restart); after that, retry",
	)
	ErrGcpReplicationOff = errors.New(
		"GCP Cloud SQL: external replication requires console-side enablement (see cloud.google.com/sql/docs/postgres/replication/configure-external-replica); provision the user via gcloud after enabling, or use REPLICATION-capable credentials directly",
	)
	ErrReplicationGrantDeny = errors.New(
		"replication grant denied on this managed PG; consult the platform docs for how to grant REPLICATION (or membership in its replication role)",
	)
)

type serverErrorKind int

const (
	serverErrorOther serverErrorKind = iota
	serverErrorNameTaken
	serverErrorObjectMissing
	serverErrorNoHbaEntry
	serverErrorBadPassword
	serverErrorNoReplicationPrivilege
	serverErrorPermissionDenied
)

// classifyServerError reads what the server reported. SQLSTATE decides where it is unambiguous:
// 42710 duplicate_object for a taken role or slot name, 42704 undefined_object for a slot that is
// already gone. The connection failures and the refused grants share their codes with other
// causes (28000 and 42501), so those are told apart by the server's message text, here and nowhere
// else.
func classifyServerError(err error) serverErrorKind {
	if err == nil {
		return serverErrorOther
	}

	if pgErr, isPgErr := errors.AsType[*pgconn.PgError](err); isPgErr {
		switch pgErr.Code {
		case "42710":
			return serverErrorNameTaken
		case "42704":
			return serverErrorObjectMissing
		}
	}

	message := err.Error()

	switch {
	case strings.Contains(message, "no pg_hba.conf entry"):
		return serverErrorNoHbaEntry
	case strings.Contains(message, "password authentication failed"):
		return serverErrorBadPassword
	case strings.Contains(message, "must have replication privilege"),
		strings.Contains(message, "must be replication role"),
		strings.Contains(message, "permission denied to start WAL sender"):
		return serverErrorNoReplicationPrivilege
	case strings.Contains(message, "permission denied"), strings.Contains(message, "must have"):
		return serverErrorPermissionDenied
	}

	if pgErr, isPgErr := errors.AsType[*pgconn.PgError](err); isPgErr && pgErr.Code == "42501" {
		return serverErrorPermissionDenied
	}

	return serverErrorOther
}

var (
	errByteStall            = errors.New("byte-stall watcher cancelled pg_basebackup")
	errManifestWalk         = errors.New("manifest reconstruction failed")
	errManifestSystemIDZero = errors.New(
		"source system identifier is 0; cannot build a manifest pg_combinebackup would accept",
	)
)

type BaseBackupErrorKind int

const (
	BaseBackupStalled BaseBackupErrorKind = iota
	BaseBackupCodecRejected
	BaseBackupManifestBroken
	BaseBackupContextEnded
	BaseBackupToolFailed
)

type BaseBackupError struct {
	Kind   BaseBackupErrorKind
	Cause  error
	Stderr []byte
}

func (e *BaseBackupError) Error() string {
	return fmt.Sprintf("%v; stderr: %s", e.Cause, truncateStderr(e.Stderr))
}
func (e *BaseBackupError) Unwrap() error { return e.Cause }

func getClientExitError(err error) error {
	if exitErr, isExitErr := errors.AsType[*exec.ExitError](err); isExitErr {
		return exitErr
	}
	return err
}

func truncateStderr(stderr []byte) string {
	const limit = 2048
	if len(stderr) > limit {
		stderr = stderr[len(stderr)-limit:]
	}
	return string(stderr)
}
