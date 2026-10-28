package physical

import "errors"

var (
	ErrNoExtendableChain              = errors.New("no extendable chain; run a full backup first")
	ErrInvalidBackupsCursor           = errors.New("invalid backups cursor")
	ErrBackupNotFound                 = errors.New("backup not found")
	ErrBackupNotInProgress            = errors.New("backup is not in progress")
	ErrWalStreamStatisticsNotPhysical = errors.New(
		"WAL stream statistics are only available for physical PostgreSQL databases",
	)
	ErrUnknownWalStreamStatisticsPeriod = errors.New("period must be one of HOUR, DAY or WEEK")
)
