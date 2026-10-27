package backups_download

import (
	"sync"

	"databasus-backend/internal/features/backups/backups/download/restore_stream"
	"databasus-backend/internal/features/backups/backups/download/restore_token"
	backups_bridge "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/caching"
	storages_bridge "databasus-backend/internal/features/storages"
	"databasus-backend/internal/util/logger"
)

// The physical restore stream takes the user stream of the backups module, so a user never runs a
// logical download and a physical restore stream at the same time.
type streamServices struct {
	restoreTokenService *restore_token.Service
	restoreStreamWriter *restore_stream.Writer
}

var getStreamServices = sync.OnceValue(func() streamServices {
	return streamServices{
		restoreTokenService: restore_token.NewService(backups_bridge.GetModule, caching.GetStore(), logger.GetLogger()),
		restoreStreamWriter: restore_stream.NewWriter(storages_bridge.GetModule),
	}
})

func GetRestoreTokenService() *restore_token.Service {
	return getStreamServices().restoreTokenService
}

func GetRestoreStreamWriter() *restore_stream.Writer {
	return getStreamServices().restoreStreamWriter
}
