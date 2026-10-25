//go:build testhelpers

package email

import (
	"databasus-backend/internal/config"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/util/logger"
)

func loadWithoutApp() *mailer.InstanceSender {
	return mailer.NewInstanceSender(mailer.Deps{
		Settings: config.GetEnv().GetInstanceMailSettings(),
		Logger:   logger.GetLogger(),
	})
}
