package email

import (
	"context"
	"fmt"
	"html"
	"strings"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers"
)

type Deps struct {
	Repositories   Repositories
	FieldEncryptor *crypto.FieldEncryptor
	// PublicURL is the address of the instance, from which the default SMTP greeting takes its host.
	PublicURL string
}

type Provider struct {
	repository     *settingsRepository
	fieldEncryptor *crypto.FieldEncryptor
	publicURL      string
}

func New(deps Deps) *Provider {
	return &Provider{
		repository:     deps.Repositories.settings,
		fieldEncryptor: deps.FieldEncryptor,
		publicURL:      deps.PublicURL,
	}
}

func (p *Provider) GetType() notification.NotifierType {
	return notification.NotifierTypeEmail
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	emailSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	if emailSettings.TargetEmail == "" {
		return providers.NewInvalidSettingError("targetEmail", "is required")
	}

	if _, err := mailer.ParseRecipient(emailSettings.TargetEmail); err != nil {
		return providers.NewInvalidSettingError("targetEmail", err.Error())
	}

	if emailSettings.SMTPHost == "" {
		return providers.NewInvalidSettingError("smtpHost", "is required")
	}

	if emailSettings.SMTPPort == 0 {
		return providers.NewInvalidSettingError("smtpPort", "is required")
	}

	if !emailSettings.Security.IsValid() {
		return providers.NewInvalidSettingError("security", mailer.ErrUnknownSecurity.Error())
	}

	if (emailSettings.SMTPUser == "") != (emailSettings.SMTPPassword == "") {
		return providers.NewInvalidSettingError("smtpPassword", "must be given together with the SMTP user")
	}

	if emailSettings.From != "" {
		if _, err := mailer.ParseSender(emailSettings.From); err != nil {
			return providers.NewInvalidSettingError("from", err.Error())
		}
	}

	if emailSettings.HeloName != "" {
		if err := mailer.ValidateHeloName(emailSettings.HeloName); err != nil {
			return providers.NewInvalidSettingError("heloName", err.Error())
		}
	}

	return nil
}

func (p *Provider) MergeStoredSecrets(merge providers.SecretMerge) (providers.Settings, error) {
	incomingSettings, err := providers.CastSettings[Settings](merge.Incoming)
	if err != nil {
		return nil, err
	}

	storedSettings, err := providers.CastSettings[Settings](merge.Stored)
	if err != nil {
		return nil, err
	}

	isAddressChanged := incomingSettings.SMTPHost != storedSettings.SMTPHost ||
		incomingSettings.SMTPPort != storedSettings.SMTPPort

	if incomingSettings.SMTPPassword == "" && !isAddressChanged {
		incomingSettings.SMTPPassword = storedSettings.SMTPPassword
	}

	return incomingSettings, nil
}

func (p *Provider) Send(ctx context.Context, settings providers.Settings, message notification.Notification) error {
	emailSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	smtpPassword, err := p.fieldEncryptor.Decrypt(emailSettings.SMTPPassword)
	if err != nil {
		return fmt.Errorf("decrypt the SMTP password: %w", err)
	}

	sender, err := mailer.ResolveSender(emailSettings.From, emailSettings.SMTPUser, emailSettings.SMTPHost)
	if err != nil {
		return err
	}

	heloName := emailSettings.HeloName
	if heloName == "" {
		heloName = mailer.GetDefaultHeloName(p.publicURL)
	}

	return mailer.Send(ctx, mailer.Connection{
		Host:                 emailSettings.SMTPHost,
		Port:                 emailSettings.SMTPPort,
		Security:             emailSettings.Security,
		Username:             emailSettings.SMTPUser,
		Password:             smtpPassword,
		HeloName:             heloName,
		IsInsecureSkipVerify: emailSettings.IsInsecureSkipVerify,
	}, mailer.Message{
		Sender:    sender,
		Recipient: emailSettings.TargetEmail,
		Subject:   message.Heading,
		HTMLBody:  strings.ReplaceAll(html.EscapeString(message.Message), "\n", "<br>"),
	})
}
