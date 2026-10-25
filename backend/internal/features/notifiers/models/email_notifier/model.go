package email_notifier

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/util/encryption"
)

type EmailNotifier struct {
	NotifierID           uuid.UUID       `json:"notifierId"           gorm:"primaryKey;type:uuid;column:notifier_id"`
	TargetEmail          string          `json:"targetEmail"          gorm:"not null;type:varchar(255);column:target_email"`
	SMTPHost             string          `json:"smtpHost"             gorm:"not null;type:varchar(255);column:smtp_host"`
	SMTPPort             int             `json:"smtpPort"             gorm:"not null;column:smtp_port"`
	SMTPUser             string          `json:"smtpUser"             gorm:"type:varchar(255);column:smtp_user"`
	SMTPPassword         string          `json:"smtpPassword"         gorm:"type:varchar(255);column:smtp_password"`
	From                 string          `json:"from"                 gorm:"type:varchar(255);column:from_email"`
	IsInsecureSkipVerify bool            `json:"isInsecureSkipVerify" gorm:"default:false;column:is_insecure_skip_verify"`
	Security             mailer.Security `json:"security"             gorm:"not null;type:varchar(16);column:security"`
	HeloName             string          `json:"heloName"             gorm:"not null;type:varchar(255);column:helo_name"`
}

func (e *EmailNotifier) TableName() string {
	return "email_notifiers"
}

func (e *EmailNotifier) FillDefaults() {
	if e.Security == "" {
		e.Security = mailer.GetDefaultSecurityForPort(e.SMTPPort)
	}
}

func (e *EmailNotifier) Validate(encryptor encryption.FieldEncryptor) error {
	if e.TargetEmail == "" {
		return errors.New("target email is required")
	}

	if _, err := mailer.ParseRecipient(e.TargetEmail); err != nil {
		return err
	}

	if e.SMTPHost == "" {
		return errors.New("SMTP host is required")
	}

	if e.SMTPPort == 0 {
		return errors.New("SMTP port is required")
	}

	if !e.Security.IsValid() {
		return mailer.ErrUnknownSecurity
	}

	// Authentication is optional - both user and password must be provided together or both empty
	if (e.SMTPUser == "") != (e.SMTPPassword == "") {
		return errors.New("SMTP user and password must both be provided or both be empty")
	}

	if e.From != "" {
		if _, err := mailer.ParseSender(e.From); err != nil {
			return err
		}
	}

	if e.HeloName != "" {
		if err := mailer.ValidateHeloName(e.HeloName); err != nil {
			return err
		}
	}

	return nil
}

func (e *EmailNotifier) Send(
	encryptor encryption.FieldEncryptor,
	_ *slog.Logger,
	notification notifier_models.Notification,
) error {
	var smtpPassword string
	if e.SMTPPassword != "" {
		decrypted, err := encryptor.Decrypt(e.SMTPPassword)
		if err != nil {
			return fmt.Errorf("failed to decrypt SMTP password: %w", err)
		}
		smtpPassword = decrypted
	}

	sender, err := mailer.ResolveSender(e.From, e.SMTPUser, e.SMTPHost)
	if err != nil {
		return err
	}

	heloName := e.HeloName
	if heloName == "" {
		heloName = mailer.GetDefaultHeloName(config.GetEnv().DatabasusURL)
	}

	connection := mailer.Connection{
		Host:                 e.SMTPHost,
		Port:                 e.SMTPPort,
		Security:             e.Security,
		Username:             e.SMTPUser,
		Password:             smtpPassword,
		HeloName:             heloName,
		IsInsecureSkipVerify: e.IsInsecureSkipVerify,
	}

	return mailer.Send(context.Background(), connection, mailer.Message{
		Sender:    sender,
		Recipient: e.TargetEmail,
		Subject:   notification.Heading,
		HTMLBody:  notification.Message,
	})
}

func (e *EmailNotifier) HideSensitiveData() {
	e.SMTPPassword = ""
}

func (e *EmailNotifier) Update(incoming *EmailNotifier) {
	e.TargetEmail = incoming.TargetEmail
	e.SMTPHost = incoming.SMTPHost
	e.SMTPPort = incoming.SMTPPort
	e.SMTPUser = incoming.SMTPUser
	e.From = incoming.From
	e.IsInsecureSkipVerify = incoming.IsInsecureSkipVerify
	e.HeloName = incoming.HeloName

	if incoming.Security != "" {
		e.Security = incoming.Security
	}

	if incoming.SMTPPassword != "" {
		e.SMTPPassword = incoming.SMTPPassword
	}
}

func (e *EmailNotifier) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	if e.SMTPPassword != "" {
		encrypted, err := encryptor.Encrypt(e.SMTPPassword)
		if err != nil {
			return fmt.Errorf("failed to encrypt SMTP password: %w", err)
		}
		e.SMTPPassword = encrypted
	}
	return nil
}
