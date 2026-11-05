// Package rclone stores files through a remote of rclone, configured by the content of one section
// of an rclone configuration file.
package rclone

import "databasus-backend/internal/modules/storages/internal/providers"

type Settings struct {
	// ConfigContent is a secret: one remote section of an rclone configuration file.
	ConfigContent string
	// RemotePath is the folder inside the remote, empty for its root.
	RemotePath string
}

func (Settings) GetType() providers.Type {
	return providers.TypeRclone
}
