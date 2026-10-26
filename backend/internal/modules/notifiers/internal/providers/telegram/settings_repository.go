package telegram

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

// settingsRepository keeps the settings of Telegram notifiers in notifiers_telegram.
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
	NotifierID     uuid.UUID `gorm:"column:notifier_id;primaryKey"`
	BotToken       string    `gorm:"column:bot_token"`
	TargetChatID   string    `gorm:"column:target_chat_id"`
	ThreadID       *int64    `gorm:"column:thread_id"`
	IsProxyEnabled bool      `gorm:"column:is_proxy_enabled"`
	ProxyURL       *string   `gorm:"column:proxy_url"`
	BotAPIURL      *string   `gorm:"column:bot_api_url"`
}

func (settingsRow) TableName() string {
	return "notifiers_telegram"
}

func (r *settingsRepository) Upsert(ctx context.Context, notifierID uuid.UUID, settings providers.Settings) error {
	telegramSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	encryptedBotToken, err := r.fieldEncryptor.Encrypt(telegramSettings.BotToken)
	if err != nil {
		return fmt.Errorf("encrypt the bot token: %w", err)
	}

	encryptedProxyURL, err := r.fieldEncryptor.Encrypt(telegramSettings.ProxyURL)
	if err != nil {
		return fmt.Errorf("encrypt the proxy URL: %w", err)
	}

	row := settingsRow{
		NotifierID:     notifierID,
		BotToken:       encryptedBotToken,
		TargetChatID:   telegramSettings.TargetChatID,
		ThreadID:       telegramSettings.ThreadID,
		IsProxyEnabled: telegramSettings.IsProxyEnabled,
		ProxyURL:       &encryptedProxyURL,
		BotAPIURL:      &telegramSettings.BotAPIURL,
	}

	err = database.Conn(ctx, r.db).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("upsert the telegram settings: %w", err)
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
		return nil, fmt.Errorf("list the telegram settings: %w", err)
	}

	settingsByNotifierID := make(map[uuid.UUID]providers.Settings, len(rows))
	for _, row := range rows {
		settings := Settings{
			BotToken:       row.BotToken,
			TargetChatID:   row.TargetChatID,
			ThreadID:       row.ThreadID,
			IsProxyEnabled: row.IsProxyEnabled,
			ProxyURL:       "",
			BotAPIURL:      "",
		}

		if row.ProxyURL != nil {
			settings.ProxyURL = *row.ProxyURL
		}

		if row.BotAPIURL != nil {
			settings.BotAPIURL = *row.BotAPIURL
		}

		settingsByNotifierID[row.NotifierID] = settings
	}

	return settingsByNotifierID, nil
}
