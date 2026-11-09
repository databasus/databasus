package upgrade

import "errors"

//nolint:staticcheck // Operator recovery guidance must match the SQL migration verbatim.
var ErrGoogleDriveRetired = errors.New("Google Drive support ended after Databasus 3.67.0. " +
	"Roll back to Databasus 3.67.0 and remove all Google Drive storages before upgrading, " +
	"or continue using 3.67.0.")

//nolint:staticcheck // Operator recovery guidance must match the SQL migration verbatim.
var ErrRcloneRetired = errors.New("Databasus 3.67.1 supports rclone storages. " +
	"Roll back to Databasus 3.67.1 and remove all rclone storages before upgrading, " +
	"or continue using 3.67.1.")

//nolint:staticcheck // Operator recovery guidance names the release that supports both types.
var ErrGoogleDriveAndRcloneRetired = errors.New("Google Drive and rclone storages prevent this upgrade. " +
	"Roll back to Databasus 3.67.0 and remove all Google Drive and rclone storages before upgrading, " +
	"or continue using 3.67.0.")
