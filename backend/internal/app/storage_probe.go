package app

import (
	"context"

	"databasus-backend/internal/modules/storages"
)

const storagePermissionsDocumentationURL = "https://databasus.com/advanced-config/#docker-storage-permissions"

// ProbeStorage saves and deletes a file through local storage. Container startup runs it before the
// internal database exists, so it needs neither the database nor New. The error describes what the
// operator has to fix.
func ProbeStorage(ctx context.Context, environment *Environment) error {
	err := storages.ProbeLocalStorage(ctx, storages.LocalStorageProbe{
		Folders: storages.LocalFolders{
			TempFolder: environment.Config.TempFolder,
			DataFolder: environment.Config.DataFolder,
		},
		Logger: environment.Logging.GetLogger(),
	})
	if err != nil {
		return &StorageProbeError{
			TempFolder: environment.Config.TempFolder,
			DataFolder: environment.Config.DataFolder,
			Err:        err,
		}
	}

	return nil
}
