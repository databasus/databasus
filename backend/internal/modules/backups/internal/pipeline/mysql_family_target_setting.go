package pipeline

import (
	"errors"

	"databasus-backend/internal/engines/mysqlfamily"
)

// toInvalidTargetSettingError names the setting a MySQL or MariaDB engine refused, so the restore
// answers with the target's field.
func toInvalidTargetSettingError(err error) error {
	invalidSettingError, isInvalidSetting := errors.AsType[*mysqlfamily.InvalidSettingError](err)
	if !isInvalidSetting {
		return err
	}

	return &InvalidTargetSettingError{Setting: invalidSettingError.Setting, Err: err}
}
