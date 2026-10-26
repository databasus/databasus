//go:build testhelpers

package notifiers_bridge

import (
	"time"

	"databasus-backend/internal/config"
	users_bridge "databasus-backend/internal/features/users"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

const deliveryTimeoutWithoutApp = 30 * time.Second

// loadWithoutApp leaves the endpoints of Slack and the official Telegram Bot API empty: the old
// tests attach only webhook notifiers.
func loadWithoutApp() notifiers.Module {
	return notifiers.New(notifiers.Deps{
		DB:         storage.GetDb(),
		Transactor: database.NewTransactor(storage.GetDb()),
		Logger:     logger.GetLogger(),
		AuditLog: audit.New(
			audit.Deps{DB: storage.GetDb(), Clock: clock.NewSystem(), AuditLogger: logger.GetAuditLogger()},
		),
		FieldEncryptor:  crypto.NewFieldEncryptor(encryption.GetMasterKey()),
		Users:           users_bridge.GetModule(),
		Workspaces:      workspaces_bridge.GetModule(),
		DeliveryTimeout: deliveryTimeoutWithoutApp,
		Endpoints:       notifiers.Endpoints{SlackPostMessageURL: "", TelegramBotAPIURL: ""},
		PublicURL:       config.GetEnv().DatabasusURL,
	})
}
