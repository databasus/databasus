package mattermost

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/notifiers/internal/providers"
)

// settingsRepository keeps the settings of Mattermost notifiers in notifiers_mattermost.
type settingsRepository struct {
	db             *gorm.DB
	fieldEncryptor *crypto.FieldEncryptor
}

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// provider never holds it.
type Repositories struct {
	settings *settingsRepository
}

func NewRepositories(db *gorm.DB, fieldEncryptor *crypto.FieldEncryptor) Repositories {
	return Repositories{settings: &settingsRepository{db: db, fieldEncryptor: fieldEncryptor}}
}

type settingsRow struct {
	NotifierID           uuid.UUID    `gorm:"column:notifier_id;primaryKey"`
	DeliveryMode         DeliveryMode `gorm:"column:delivery_mode"`
	WebhookURL           string       `gorm:"column:webhook_url"`
	ServerURL            string       `gorm:"column:server_url"`
	BotToken             string       `gorm:"column:bot_token"`
	TargetChannelName    string       `gorm:"column:target_channel_name"`
	TargetChannelID      string       `gorm:"column:target_channel_id"`
	OverrideUsername     string       `gorm:"column:override_username"`
	OverrideIconURL      string       `gorm:"column:override_icon_url"`
	IsInsecureSkipVerify bool         `gorm:"column:is_insecure_skip_verify"`
}

func (settingsRow) TableName() string {
	return "notifiers_mattermost"
}

func (r *settingsRepository) Upsert(ctx context.Context, notifierID uuid.UUID, settings providers.Settings) error {
	mattermostSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	encryptedWebhookURL, err := r.fieldEncryptor.Encrypt(mattermostSettings.WebhookURL)
	if err != nil {
		return fmt.Errorf("encrypt the incoming webhook URL: %w", err)
	}

	encryptedBotToken, err := r.fieldEncryptor.Encrypt(mattermostSettings.BotToken)
	if err != nil {
		return fmt.Errorf("encrypt the bot token: %w", err)
	}

	row := settingsRow{
		NotifierID:           notifierID,
		DeliveryMode:         mattermostSettings.DeliveryMode,
		WebhookURL:           encryptedWebhookURL,
		ServerURL:            mattermostSettings.ServerURL,
		BotToken:             encryptedBotToken,
		TargetChannelName:    mattermostSettings.TargetChannelName,
		TargetChannelID:      mattermostSettings.TargetChannelID,
		OverrideUsername:     mattermostSettings.OverrideUsername,
		OverrideIconURL:      mattermostSettings.OverrideIconURL,
		IsInsecureSkipVerify: mattermostSettings.IsInsecureSkipVerify,
	}

	err = database.Conn(ctx, r.db).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("upsert the mattermost settings: %w", err)
	}

	return nil
}

func (r *settingsRepository) ListByNotifierIDs(
	ctx context.Context,
	notifierIDs []uuid.UUID,
) (map[uuid.UUID]providers.Settings, error) {
	var rows []settingsRow

	err := database.Conn(ctx, r.db).Where("notifier_id IN ?", notifierIDs).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list the mattermost settings: %w", err)
	}

	settingsByNotifierID := make(map[uuid.UUID]providers.Settings, len(rows))
	for _, row := range rows {
		settingsByNotifierID[row.NotifierID] = Settings{
			DeliveryMode:         row.DeliveryMode,
			WebhookURL:           row.WebhookURL,
			ServerURL:            row.ServerURL,
			BotToken:             row.BotToken,
			TargetChannelName:    row.TargetChannelName,
			TargetChannelID:      row.TargetChannelID,
			OverrideUsername:     row.OverrideUsername,
			OverrideIconURL:      row.OverrideIconURL,
			IsInsecureSkipVerify: row.IsInsecureSkipVerify,
		}
	}

	return settingsByNotifierID, nil
}
