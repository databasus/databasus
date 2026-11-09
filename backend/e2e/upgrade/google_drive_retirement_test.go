package upgrade

import "testing"

const (
	googleDriveReleaseImage = "databasus/databasus:v3.67.0@sha256:356f0d75d8e7d3e964fe7be6e353f8a0370f3b661d2dc77a0afb6f7d067b44c6"
	retirementGuidance      = "Google Drive support ended after Databasus 3.67.0. " +
		"Roll back to Databasus 3.67.0 and remove all Google Drive storages before upgrading, " +
		"or continue using 3.67.0."
	retirementClientSecret = "retirement-synthetic-client-secret"
	retirementRefreshToken = "retirement-synthetic-refresh-token"
)

type retiredGoogleDriveSettings struct {
	Type         string `json:"type"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	TokenJSON    string `json:"tokenJson"`
}

func TestGoogleDriveRetirement(t *testing.T) {
	t.Parallel()
	t.Run("preserves 3.67.0 on refusal and upgrades after removal through its API", func(t *testing.T) {
		t.Parallel()
		runRetirementScenario(t, retirementScenario{
			image: googleDriveReleaseImage, guidance: retirementGuidance,
			settings: []any{newRetiredGoogleDriveSettings()},
		})
	})
}

func newRetiredGoogleDriveSettings() retiredGoogleDriveSettings {
	return retiredGoogleDriveSettings{
		Type:         "GOOGLE_DRIVE",
		ClientID:     "retirement-client.apps.googleusercontent.com",
		ClientSecret: retirementClientSecret,
		TokenJSON:    `{"access_token":"synthetic-access","refresh_token":"` + retirementRefreshToken + `","token_type":"Bearer","expiry":"2030-01-01T00:00:00Z"}`,
	}
}
