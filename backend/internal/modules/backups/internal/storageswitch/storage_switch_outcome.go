package storageswitch

import "databasus-backend/internal/modules/storages"

type Outcome struct {
	Operation StorageSwitch
	Cleanup   storages.DeletionBatch
}
