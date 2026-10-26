package httpapi

import "databasus-backend/internal/modules/notifiers/internal/providers/teams"

func toTeamsSettings(body TeamsNotifierSettings) teams.Settings {
	return teams.Settings{WebhookURL: getStringOrEmpty(body.PowerAutomateUrl)}
}

func fromTeamsSettings(teams.Settings) TeamsNotifierSettings {
	return TeamsNotifierSettings{Type: TeamsNotifierSettingsTypeTEAMS, PowerAutomateUrl: nil}
}
