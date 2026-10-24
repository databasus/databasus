package backups_download

import (
	"sync"

	"databasus-backend/internal/features/backups/backups/download/download_token"
	"databasus-backend/internal/features/backups/backups/download/restore_stream"
	"databasus-backend/internal/features/backups/backups/download/restore_token"
	"databasus-backend/internal/features/backups/backups/download/stream_guard"
	"databasus-backend/internal/features/storages"
	"databasus-backend/internal/util/cache"
	util_encryption "databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

// This package is the composition root for the download/restore stream feature.
// The single stream_guard.Guard built here is injected into BOTH token services
// so the per-user single-stream lock is shared — a user can never run a logical
// download and a physical restore stream at the same time.
type streamServices struct {
	downloadTokenService           *download_token.Service
	restoreTokenService            *restore_token.Service
	downloadTokenBackgroundService *download_token.BackgroundService
	restoreStreamWriter            *restore_stream.Writer
}

var getStreamServices = sync.OnceValue(func() streamServices {
	cacheStore := cache.GetStore()

	guard := stream_guard.NewGuard(
		stream_guard.NewTracker(cacheStore),
		logger.GetLogger(),
	)

	downloadTokenService := download_token.NewService(guard, logger.GetLogger())

	return streamServices{
		downloadTokenService: downloadTokenService,
		restoreTokenService:  restore_token.NewService(guard, cacheStore, logger.GetLogger()),
		downloadTokenBackgroundService: download_token.NewBackgroundService(
			downloadTokenService,
			logger.GetLogger(),
		),
		restoreStreamWriter: restore_stream.NewWriter(
			storages.GetStorageService(),
			util_encryption.GetFieldEncryptor(),
			logger.GetLogger(),
		),
	}
})

func GetDownloadTokenService() *download_token.Service {
	return getStreamServices().downloadTokenService
}

func GetRestoreTokenService() *restore_token.Service {
	return getStreamServices().restoreTokenService
}

func GetDownloadTokenBackgroundService() *download_token.BackgroundService {
	return getStreamServices().downloadTokenBackgroundService
}

func GetRestoreStreamWriter() *restore_stream.Writer {
	return getStreamServices().restoreStreamWriter
}
