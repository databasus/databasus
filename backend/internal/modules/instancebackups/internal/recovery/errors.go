package recovery

import (
	"errors"

	"databasus-backend/internal/lib/apierror"
)

var (
	ErrJournalUnreadable = errors.New("the recovery journal is not readable")
	// ErrPublicationUnfinished keeps a process away from a database whose replacement another start
	// must finish first.
	ErrPublicationUnfinished = errors.New("a recovery publication is unfinished")
	// ErrPublishedKeyMismatch refuses a start whose key file is not the one published with the
	// database, because the database's secrets would not decrypt.
	ErrPublishedKeyMismatch = errors.New("the master key file is not the one published with the recovered database")
	ErrLifecycleBusy        = errors.New("another process holds the lifecycle lock exclusively")
	ErrArchiveChanged       = errors.New("the Databasus backup file changed while it was copied")
	ErrStagedStateInvalid   = errors.New("the recovered database does not match its Databasus backup")
	// ErrPublicationNotStarted is a publication that failed before it changed the active database
	// or key.
	ErrPublicationNotStarted = errors.New("the recovery publication could not start")
	// errStagingWrite is a failure to write the staged copy of an upload, apart from a failure to
	// read the upload, which is the client's.
	errStagingWrite = errors.New("write the staged Databasus backup file")
)

var (
	ErrRecoveryNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the recovery is not in the local journal",
		Errors: nil,
	}
	ErrRecoveryNotPaused = &apierror.Problem{
		Code:   apierror.CodeRecoveryNotReady,
		Detail: "the recovery does not hold the instance paused",
		Errors: nil,
	}
	// ErrRecoveryNotReady is a recovery that is not prepared, expired or already ended, so it cannot be
	// confirmed or applied.
	ErrRecoveryNotReady = &apierror.Problem{
		Code:   apierror.CodeRecoveryNotReady,
		Detail: "the recovery is not prepared for apply",
		Errors: nil,
	}
	ErrRecoveryAlreadyRunning = &apierror.Problem{
		Code:   apierror.CodeAlreadyRunning,
		Detail: "another recovery is being prepared or applied",
		Errors: nil,
	}
	// ErrRecoveryNotCancellable is a recovery whose apply was accepted or that already ended.
	ErrRecoveryNotCancellable = &apierror.Problem{
		Code:   apierror.CodeRecoveryNotReady,
		Detail: "the recovery can no longer be cancelled",
		Errors: nil,
	}
	// ErrRecoveryEndedDuringUpload is an upload whose recovery was cancelled while its file arrived.
	ErrRecoveryEndedDuringUpload = &apierror.Problem{
		Code:   apierror.CodeRecoveryNotReady,
		Detail: "the recovery was cancelled while its file was uploaded",
		Errors: nil,
	}
	ErrArchiveTooLarge = &apierror.Problem{
		Code:   apierror.CodeArchiveTooLarge,
		Detail: "the Databasus backup file is larger than RECOVERY_MAX_ARCHIVE_BYTES allows",
		Errors: nil,
	}
	ErrInsufficientDiskSpace = &apierror.Problem{
		Code:   apierror.CodeInsufficientDiskSpace,
		Detail: "the data volume has too little free space to stage the Databasus backup file",
		Errors: nil,
	}
)

// RefusalReason is why a configured startup recovery did not run.
type RefusalReason string

const (
	RefusalExistingAccounts    RefusalReason = "EXISTING_ACCOUNTS"
	RefusalChangedSettings     RefusalReason = "CHANGED_SETTINGS"
	RefusalUserData            RefusalReason = "USER_DATA"
	RefusalUnknownState        RefusalReason = "UNKNOWN_STATE"
	RefusalNewerSchema         RefusalReason = "NEWER_SCHEMA"
	RefusalInputUnreadable     RefusalReason = "INPUT_UNREADABLE"
	RefusalInvalidMasterKey    RefusalReason = "INVALID_MASTER_KEY"
	RefusalInvalidArchive      RefusalReason = "INVALID_ARCHIVE"
	RefusalIncompatibleRelease RefusalReason = "INCOMPATIBLE_RELEASE"
	RefusalIncompatibleSchema  RefusalReason = "INCOMPATIBLE_SCHEMA"
	RefusalNoExclusiveAccess   RefusalReason = "NO_EXCLUSIVE_ACCESS"
	RefusalInsufficientDisk    RefusalReason = "INSUFFICIENT_DISK_SPACE"
	RefusalImportFailed        RefusalReason = "IMPORT_FAILED"
)

// RefusalError is a configured startup recovery that did not run. Explanation and NextStep are
// written for the operator reading the log; neither carries a key, a credential or a user record.
type RefusalError struct {
	Reason      RefusalReason
	Explanation string
	NextStep    string
	Cause       error
}

func (r *RefusalError) Error() string {
	if r.Cause == nil {
		return "startup recovery did not run: " + r.Explanation
	}

	return "startup recovery did not run: " + r.Explanation + ": " + r.Cause.Error()
}

func (r *RefusalError) Unwrap() error {
	return r.Cause
}
