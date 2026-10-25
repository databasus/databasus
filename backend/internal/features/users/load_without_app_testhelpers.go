//go:build testhelpers

package users_bridge

import (
	"databasus-backend/internal/features/caching"
	"databasus-backend/internal/features/email"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

func loadWithoutApp() users.Module {
	return NewModuleWithoutApp(email.GetInstanceSender())
}

// NewModuleWithoutApp builds the module from the old access points with the given mailer, for a test
// that serves the module's routes itself and must control what mail delivery answers.
func NewModuleWithoutApp(mailSender mailer.Sender) users.Module {
	systemClock := clock.NewSystem()

	return users.New(users.Deps{
		DB:         storage.GetDb(),
		Transactor: database.NewTransactor(storage.GetDb()),
		Clock:      systemClock,
		Logger:     logger.GetLogger(),
		MasterKey:  encryption.GetMasterKey(),
		Mailer:     mailSender,
		AuditLog: audit.New(
			audit.Deps{DB: storage.GetDb(), Clock: systemClock, AuditLogger: logger.GetAuditLogger()},
		),
		RateLimiter: caching.GetRateLimiter(),
		// The tests of the old code sign in through the module only with a password, so they need no
		// challenge and no identity provider.
		Turnstile: users.TurnstileSettings{SecretKey: "", VerifyURL: ""},
		GitHub: users.GitHubSettings{
			ClientID:     "",
			ClientSecret: "",
			AuthURL:      "",
			TokenURL:     "",
			UserURL:      "",
			EmailsURL:    "",
		},
		Google: users.GoogleSettings{ClientID: "", ClientSecret: "", AuthURL: "", TokenURL: "", UserInfoURL: ""},
	})
}
