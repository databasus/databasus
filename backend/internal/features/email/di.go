package email

import (
	"sync"

	"databasus-backend/internal/config"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/smtp_transport"
)

var getEmailSMTPSender = sync.OnceValue(func() *EmailSMTPSender {
	return newEmailSMTPSenderFromEnv(config.GetEnv())
})

func GetEmailSMTPSender() *EmailSMTPSender {
	return getEmailSMTPSender()
}

func newEmailSMTPSenderFromEnv(env *config.EnvVariables) *EmailSMTPSender {
	if env.SMTPHost == "" {
		return &EmailSMTPSender{logger.GetLogger(), smtp_transport.Connection{}, "", false}
	}

	security := env.SMTPSecurity
	if security == "" {
		security = smtp_transport.GetDefaultSecurityForPort(env.SMTPPort)
	}

	heloName := env.SMTPHeloName
	if heloName == "" {
		heloName = smtp_transport.GetDefaultHeloName(env.DatabasusURL)
	}

	connection := smtp_transport.Connection{
		Host:                 env.SMTPHost,
		Port:                 env.SMTPPort,
		Security:             security,
		Username:             env.SMTPUser,
		Password:             env.SMTPPassword,
		HeloName:             heloName,
		IsInsecureSkipVerify: env.SMTPInsecureSkipVerify,
	}

	return &EmailSMTPSender{logger.GetLogger(), connection, env.SMTPFrom, true}
}
