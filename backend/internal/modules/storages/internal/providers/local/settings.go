// Package local stores files in the data folder of the instance, staging each one in the temp folder
// until it is complete.
package local

import "databasus-backend/internal/modules/storages/internal/providers"

// Settings are empty: every local storage writes to the folders of the instance.
type Settings struct{}

func (Settings) GetType() providers.Type {
	return providers.TypeLocal
}
