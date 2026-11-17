package providers

import (
	"errors"

	"databasus-backend/internal/engines/mysqlfamily"
)

// ValidateMysqlFamilyConnectionValues refuses a MySQL or MariaDB value that the client tools or the
// driver could read as more than a value, naming the setting the engine reports.
func ValidateMysqlFamilyConnectionValues(values mysqlfamily.ConnectionValues) error {
	err := values.Validate()

	invalidSettingError, isInvalidSetting := errors.AsType[*mysqlfamily.InvalidSettingError](err)
	if isInvalidSetting {
		return NewInvalidSettingError(invalidSettingError.Setting, invalidSettingError.Reason)
	}

	return err
}
