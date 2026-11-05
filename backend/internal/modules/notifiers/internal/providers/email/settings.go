// Package email sends notifications as email through the SMTP server of the notifier.
package email

import (
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/modules/notifiers/internal/notification"
)

type Settings struct {
	TargetEmail string
	SMTPHost    string
	SMTPPort    int
	// SMTPUser and SMTPPassword are both empty when the server needs no authentication.
	SMTPUser string
	// SMTPPassword is a secret.
	SMTPPassword string
	// From is empty when the sender is derived from the user and the host.
	From                 string
	IsInsecureSkipVerify bool
	Security             mailer.Security
	// HeloName is empty when the greeting uses the host of the instance's public address.
	HeloName string
}

func (Settings) GetType() notification.NotifierType {
	return notification.NotifierTypeEmail
}
