package config

import (
	"fmt"

	"databasus-backend/internal/lib/mailer"
)

func validate(loadedConfig Config, spec LoadSpec) error {
	if spec.IsDatabaseRequired && loadedConfig.DatabaseDsn == "" {
		return fmt.Errorf("%w: %s", ErrRequiredVariableNotSet, databaseDsnVariable)
	}

	if loadedConfig.EnvMode != EnvModeDevelopment && loadedConfig.EnvMode != EnvModeProduction {
		return fmt.Errorf(
			"%w: ENV_MODE %q is neither development nor production",
			ErrInvalidVariable,
			loadedConfig.EnvMode,
		)
	}

	return validateSMTPSettings(loadedConfig)
}

func validateSMTPSettings(loadedConfig Config) error {
	if loadedConfig.SMTPHost != "" && loadedConfig.SMTPPort <= 0 {
		return fmt.Errorf(
			"%w: SMTP_PORT must be a positive integer when SMTP_HOST is set, got %d",
			ErrInvalidVariable, loadedConfig.SMTPPort,
		)
	}

	if loadedConfig.SMTPSecurity != "" && !loadedConfig.SMTPSecurity.IsValid() {
		return fmt.Errorf("SMTP_SECURITY %q: %w", loadedConfig.SMTPSecurity, mailer.ErrUnknownSecurity)
	}

	if loadedConfig.SMTPHeloName != "" {
		if err := mailer.ValidateHeloName(loadedConfig.SMTPHeloName); err != nil {
			return fmt.Errorf("SMTP_HELO_NAME %q: %w", loadedConfig.SMTPHeloName, err)
		}
	}

	if loadedConfig.SMTPFrom != "" {
		if _, err := mailer.ParseSender(loadedConfig.SMTPFrom); err != nil {
			return fmt.Errorf("SMTP_FROM %q: %w", loadedConfig.SMTPFrom, err)
		}
	}

	return nil
}
