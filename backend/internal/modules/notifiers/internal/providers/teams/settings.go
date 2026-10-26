// Package teams sends notifications to Microsoft Teams through a Power Automate workflow URL.
package teams

import "databasus-backend/internal/modules/notifiers/internal/notification"

type Settings struct {
	// WebhookURL is the Power Automate URL, a secret because the URL itself authorizes the post.
	WebhookURL string
}

func (Settings) GetType() notification.NotifierType {
	return notification.NotifierTypeTeams
}
