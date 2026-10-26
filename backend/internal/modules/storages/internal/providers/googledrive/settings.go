// Package googledrive stores files in the databasus_backups folder of a Google Drive, signing in
// with the OAuth token of the user who authorized the storage.
package googledrive

import "databasus-backend/internal/modules/storages/internal/providers"

type Settings struct {
	ClientID string
	// ClientSecret is a secret.
	ClientSecret string
	// TokenJSON is a secret: the OAuth token with its refresh token, as JSON.
	TokenJSON string
}

func (Settings) GetType() providers.Type {
	return providers.TypeGoogleDrive
}
