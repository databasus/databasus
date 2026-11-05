package discord

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

// settingsRepository keeps the settings of Discord notifiers in notifiers_discord.
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
	NotifierID        uuid.UUID `gorm:"column:notifier_id;primaryKey"`
	ChannelWebhookURL string    `gorm:"column:channel_webhook_url"`
}

func (settingsRow) TableName() string {
	return "notifiers_discord"
}

func (r *settingsRepository) Upsert(ctx context.Context, notifierID uuid.UUID, settings providers.Settings) error {
	discordSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	encryptedChannelWebhookURL, err := r.fieldEncryptor.Encrypt(discordSettings.ChannelWebhookURL)
	if err != nil {
		return fmt.Errorf("encrypt the channel webhook URL: %w", err)
	}

	row := settingsRow{NotifierID: notifierID, ChannelWebhookURL: encryptedChannelWebhookURL}

	err = database.Conn(ctx, r.db).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("upsert the discord settings: %w", err)
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
		return nil, fmt.Errorf("list the discord settings: %w", err)
	}

	settingsByNotifierID := make(map[uuid.UUID]providers.Settings, len(rows))
	for _, row := range rows {
		settingsByNotifierID[row.NotifierID] = Settings{ChannelWebhookURL: row.ChannelWebhookURL}
	}

	return settingsByNotifierID, nil
}
