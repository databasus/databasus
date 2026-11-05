package notifiers

import (
	"errors"
	"log/slog"

	"github.com/google/uuid"

	notifier_models "databasus-backend/internal/features/notifiers/models"
	discord_notifier "databasus-backend/internal/features/notifiers/models/discord"
	"databasus-backend/internal/features/notifiers/models/email_notifier"
	mattermost_notifier "databasus-backend/internal/features/notifiers/models/mattermost"
	"databasus-backend/internal/features/notifiers/models/ses"
	slack_notifier "databasus-backend/internal/features/notifiers/models/slack"
	"databasus-backend/internal/features/notifiers/models/sns"
	teams_notifier "databasus-backend/internal/features/notifiers/models/teams"
	telegram_notifier "databasus-backend/internal/features/notifiers/models/telegram"
	webhook_notifier "databasus-backend/internal/features/notifiers/models/webhook"
	"databasus-backend/internal/util/encryption"
)

type Notifier struct {
	ID            uuid.UUID    `json:"id"            gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	WorkspaceID   uuid.UUID    `json:"workspaceId"   gorm:"column:workspace_id;not null;type:uuid;index"`
	Name          string       `json:"name"          gorm:"column:name;not null;type:varchar(255)"`
	NotifierType  NotifierType `json:"notifierType"  gorm:"column:notifier_type;not null;type:varchar(50)"`
	LastSendError *string      `json:"lastSendError" gorm:"column:last_send_error;type:text"`

	// specific notifier
	TelegramNotifier   *telegram_notifier.TelegramNotifier     `json:"telegramNotifier"       gorm:"foreignKey:NotifierID"`
	EmailNotifier      *email_notifier.EmailNotifier           `json:"emailNotifier"          gorm:"foreignKey:NotifierID"`
	WebhookNotifier    *webhook_notifier.WebhookNotifier       `json:"webhookNotifier"        gorm:"foreignKey:NotifierID"`
	SlackNotifier      *slack_notifier.SlackNotifier           `json:"slackNotifier"          gorm:"foreignKey:NotifierID"`
	DiscordNotifier    *discord_notifier.DiscordNotifier       `json:"discordNotifier"        gorm:"foreignKey:NotifierID"`
	TeamsNotifier      *teams_notifier.TeamsNotifier           `json:"teamsNotifier,omitzero" gorm:"foreignKey:NotifierID;constraint:OnDelete:CASCADE"`
	MattermostNotifier *mattermost_notifier.MattermostNotifier `json:"mattermostNotifier"     gorm:"foreignKey:NotifierID"`
	SESNotifier        *ses_notifier.SESNotifier               `json:"sesNotifier"            gorm:"foreignKey:NotifierID"`
	SNSNotifier        *sns_notifier.SNSNotifier               `json:"snsNotifier"            gorm:"foreignKey:NotifierID"`
}

func (n *Notifier) TableName() string {
	return "notifiers"
}

func (n *Notifier) FillDefaults() {
	if n.NotifierType == NotifierTypeEmail && n.EmailNotifier != nil {
		n.EmailNotifier.FillDefaults()
	}

	if n.NotifierType == NotifierTypeSNS && n.SNSNotifier != nil {
		n.SNSNotifier.FillDefaults()
	}
}

func (n *Notifier) Validate(encryptor encryption.FieldEncryptor) error {
	if n.Name == "" {
		return errors.New("name is required")
	}

	return n.getSpecificNotifier().Validate(encryptor)
}

func (n *Notifier) Send(
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	notification notifier_models.Notification,
) error {
	err := n.getSpecificNotifier().Send(encryptor, logger, notification)

	if err != nil {
		lastSendError := err.Error()
		n.LastSendError = &lastSendError
	} else {
		n.LastSendError = nil
	}

	return err
}

func (n *Notifier) HideSensitiveData() {
	n.getSpecificNotifier().HideSensitiveData()
}

func (n *Notifier) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	return n.getSpecificNotifier().EncryptSensitiveData(encryptor)
}

func (n *Notifier) Update(incoming *Notifier) {
	n.Name = incoming.Name
	n.NotifierType = incoming.NotifierType

	switch n.NotifierType {
	case NotifierTypeTelegram:
		if n.TelegramNotifier != nil && incoming.TelegramNotifier != nil {
			n.TelegramNotifier.Update(incoming.TelegramNotifier)
		}
	case NotifierTypeEmail:
		if n.EmailNotifier != nil && incoming.EmailNotifier != nil {
			n.EmailNotifier.Update(incoming.EmailNotifier)
		}
	case NotifierTypeWebhook:
		if n.WebhookNotifier != nil && incoming.WebhookNotifier != nil {
			n.WebhookNotifier.Update(incoming.WebhookNotifier)
		}
	case NotifierTypeSlack:
		if n.SlackNotifier != nil && incoming.SlackNotifier != nil {
			n.SlackNotifier.Update(incoming.SlackNotifier)
		}
	case NotifierTypeDiscord:
		if n.DiscordNotifier != nil && incoming.DiscordNotifier != nil {
			n.DiscordNotifier.Update(incoming.DiscordNotifier)
		}
	case NotifierTypeTeams:
		if n.TeamsNotifier != nil && incoming.TeamsNotifier != nil {
			n.TeamsNotifier.Update(incoming.TeamsNotifier)
		}
	case NotifierTypeMattermost:
		if n.MattermostNotifier != nil && incoming.MattermostNotifier != nil {
			n.MattermostNotifier.Update(incoming.MattermostNotifier)
		}
	case NotifierTypeSES:
		if n.SESNotifier != nil && incoming.SESNotifier != nil {
			n.SESNotifier.Update(incoming.SESNotifier)
		}
	case NotifierTypeSNS:
		if n.SNSNotifier != nil && incoming.SNSNotifier != nil {
			n.SNSNotifier.Update(incoming.SNSNotifier)
		}
	}
}

func (n *Notifier) getSpecificNotifier() NotificationSender {
	switch n.NotifierType {
	case NotifierTypeTelegram:
		return n.TelegramNotifier
	case NotifierTypeEmail:
		return n.EmailNotifier
	case NotifierTypeWebhook:
		return n.WebhookNotifier
	case NotifierTypeSlack:
		return n.SlackNotifier
	case NotifierTypeDiscord:
		return n.DiscordNotifier
	case NotifierTypeTeams:
		return n.TeamsNotifier
	case NotifierTypeMattermost:
		return n.MattermostNotifier
	case NotifierTypeSES:
		return n.SESNotifier
	case NotifierTypeSNS:
		return n.SNSNotifier
	default:
		panic("unknown notifier type: " + string(n.NotifierType))
	}
}
