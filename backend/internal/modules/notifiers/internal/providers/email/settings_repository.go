package email

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/modules/notifiers/internal/providers"
)

// settingsRepository keeps the settings of email notifiers in notifiers_email.
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
	NotifierID           uuid.UUID       `gorm:"column:notifier_id;primaryKey"`
	TargetEmail          string          `gorm:"column:target_email"`
	SMTPHost             string          `gorm:"column:smtp_host"`
	SMTPPort             int             `gorm:"column:smtp_port"`
	SMTPUser             *string         `gorm:"column:smtp_user"`
	SMTPPassword         *string         `gorm:"column:smtp_password"`
	From                 *string         `gorm:"column:from_email"`
	IsInsecureSkipVerify bool            `gorm:"column:is_insecure_skip_verify"`
	Security             mailer.Security `gorm:"column:security"`
	HeloName             string          `gorm:"column:helo_name"`
}

func (settingsRow) TableName() string {
	return "notifiers_email"
}

func (r *settingsRepository) Upsert(ctx context.Context, notifierID uuid.UUID, settings providers.Settings) error {
	emailSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	encryptedPassword, err := r.fieldEncryptor.Encrypt(emailSettings.SMTPPassword)
	if err != nil {
		return fmt.Errorf("encrypt the SMTP password: %w", err)
	}

	row := settingsRow{
		NotifierID:           notifierID,
		TargetEmail:          emailSettings.TargetEmail,
		SMTPHost:             emailSettings.SMTPHost,
		SMTPPort:             emailSettings.SMTPPort,
		SMTPUser:             &emailSettings.SMTPUser,
		SMTPPassword:         &encryptedPassword,
		From:                 &emailSettings.From,
		IsInsecureSkipVerify: emailSettings.IsInsecureSkipVerify,
		Security:             emailSettings.Security,
		HeloName:             emailSettings.HeloName,
	}

	err = database.Conn(ctx, r.db).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("upsert the email settings: %w", err)
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
		return nil, fmt.Errorf("list the email settings: %w", err)
	}

	settingsByNotifierID := make(map[uuid.UUID]providers.Settings, len(rows))
	for _, row := range rows {
		settingsByNotifierID[row.NotifierID] = Settings{
			TargetEmail:          row.TargetEmail,
			SMTPHost:             row.SMTPHost,
			SMTPPort:             row.SMTPPort,
			SMTPUser:             valueOrEmpty(row.SMTPUser),
			SMTPPassword:         valueOrEmpty(row.SMTPPassword),
			From:                 valueOrEmpty(row.From),
			IsInsecureSkipVerify: row.IsInsecureSkipVerify,
			Security:             row.Security,
			HeloName:             row.HeloName,
		}
	}

	return settingsByNotifierID, nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
