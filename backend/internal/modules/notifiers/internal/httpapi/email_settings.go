package httpapi

import (
	"strings"

	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/modules/notifiers/internal/providers/email"
)

func toEmailSettings(body EmailNotifierSettings) email.Settings {
	return email.Settings{
		TargetEmail:          body.TargetEmail,
		SMTPHost:             body.SmtpHost,
		SMTPPort:             body.SmtpPort,
		SMTPUser:             getStringOrEmpty(body.SmtpUser),
		SMTPPassword:         getStringOrEmpty(body.SmtpPassword),
		From:                 getStringOrEmpty(body.From),
		IsInsecureSkipVerify: body.IsInsecureSkipVerify,
		Security:             mailer.Security(strings.ToLower(string(body.Security))),
		HeloName:             getStringOrEmpty(body.HeloName),
	}
}

func fromEmailSettings(settings email.Settings) EmailNotifierSettings {
	return EmailNotifierSettings{
		Type:                 EmailNotifierSettingsTypeEMAIL,
		TargetEmail:          settings.TargetEmail,
		SmtpHost:             settings.SMTPHost,
		SmtpPort:             settings.SMTPPort,
		SmtpUser:             toOptionalString(settings.SMTPUser),
		SmtpPassword:         nil,
		From:                 toOptionalString(settings.From),
		IsInsecureSkipVerify: settings.IsInsecureSkipVerify,
		Security:             SmtpSecurity(strings.ToUpper(string(settings.Security))),
		HeloName:             toOptionalString(settings.HeloName),
	}
}
