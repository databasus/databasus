package upgrade

import "errors"

//nolint:staticcheck // Operator recovery guidance must match the SQL migration verbatim.
var ErrGoogleDriveRetired = errors.New("Google Drive support ended after Databasus 3.67.0. " +
	"Roll back to Databasus 3.67.0 and remove all Google Drive storages before upgrading, " +
	"or continue using 3.67.0.")
