package physical

import (
	"errors"
	"fmt"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/modules/backups/internal/physical/chains"
	"databasus-backend/internal/modules/backups/internal/physical/restorestream"
)

var (
	ErrDatabaseNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the database does not exist",
		Errors: nil,
	}
	ErrNotPhysicalDatabase = apierror.NewInvalidFieldsProblem(
		[]apierror.FieldError{{Field: "databaseId", Detail: "must name a physical PostgreSQL database"}},
	)
	errInvalidPageLimit = apierror.NewInvalidFieldsProblem(
		[]apierror.FieldError{{Field: "limit", Detail: "must be between 1 and 1000"}},
	)
	errNoRestorableBackup = &apierror.Problem{
		Code:   apierror.CodeNoRestorableBackup,
		Detail: "no completed backup can be restored",
		Errors: nil,
	}
	errTargetTimeUnreachable = &apierror.Problem{
		Code:   apierror.CodeTargetTimeUnreachable,
		Detail: "the target time precedes the earliest full backup",
		Errors: nil,
	}
	errInvalidRestoreToken = &apierror.Problem{
		Code:   apierror.CodeUnauthenticated,
		Detail: "the restore token is unknown, expired or already used",
		Errors: nil,
	}
	errInvalidBackupKind = apierror.NewInvalidFieldsProblem(
		[]apierror.FieldError{{Field: "kind", Detail: "must be AUTO, FULL or INCREMENTAL"}},
	)
	errBackupsTurnedOff = apierror.NewInvalidFieldsProblem(
		[]apierror.FieldError{{Field: "databaseId", Detail: "must name a database whose backups are turned on"}},
	)
	errNoStorage = apierror.NewInvalidFieldsProblem(
		[]apierror.FieldError{
			{Field: "databaseId", Detail: "must name a database whose backup config names a storage"},
		},
	)
	ErrNoExtendableChain = &apierror.Problem{
		Code:   apierror.CodeNoExtendableChain,
		Detail: "no extendable chain; run a full backup first",
		Errors: nil,
	}
	ErrInvalidBackupsCursor = apierror.NewInvalidFieldsProblem(
		[]apierror.FieldError{{Field: "after", Detail: "must be a cursor issued by this list"}},
	)
	ErrBackupNotFound = &apierror.Problem{
		Code:   apierror.CodeNotFound,
		Detail: "the backup does not exist",
		Errors: nil,
	}
	ErrBackupNotInProgress = &apierror.Problem{
		Code:   apierror.CodeNotRunning,
		Detail: "the backup is no longer running",
		Errors: nil,
	}
	ErrUnknownWalStreamStatisticsPeriod = apierror.NewInvalidFieldsProblem(
		[]apierror.FieldError{{Field: "period", Detail: "must be HOUR, DAY or WEEK"}},
	)
)

func toRestoreProblem(err error) error {
	var gap chains.WalGapBeforeTargetError
	switch {
	case errors.Is(err, chains.ErrNoChainForRestore):
		return errNoRestorableBackup
	case errors.Is(err, chains.ErrTargetBeforeEarliest):
		return errTargetTimeUnreachable
	case errors.As(err, &gap):
		return &apierror.Problem{
			Code: apierror.CodeTargetTimeUnreachable,
			Detail: fmt.Sprintf(
				"WAL gap before target; latest restorable position is %s",
				gap.LatestRestorableLSN.String(),
			),
			Errors: nil,
		}
	case errors.Is(err, restorestream.ErrInvalidToken):
		return errInvalidRestoreToken
	default:
		return err
	}
}
