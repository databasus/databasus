package notification

// NotifierType is the channel a notifier sends through. It is fixed when the notifier is created.
type NotifierType string

const (
	NotifierTypeEmail      NotifierType = "EMAIL"
	NotifierTypeTelegram   NotifierType = "TELEGRAM"
	NotifierTypeSlack      NotifierType = "SLACK"
	NotifierTypeDiscord    NotifierType = "DISCORD"
	NotifierTypeTeams      NotifierType = "TEAMS"
	NotifierTypeMattermost NotifierType = "MATTERMOST"
	NotifierTypeWebhook    NotifierType = "WEBHOOK"
)
