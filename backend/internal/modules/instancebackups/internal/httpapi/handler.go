package httpapi

import (
	"databasus-backend/internal/modules/instancebackups/internal/catalog"
	"databasus-backend/internal/modules/instancebackups/internal/configs"
	"databasus-backend/internal/modules/instancebackups/internal/health"
	"databasus-backend/internal/modules/instancebackups/internal/keyexport"
	"databasus-backend/internal/modules/instancebackups/internal/recovery"
)

var _ StrictServerInterface = (*Handler)(nil)

type (
	Deps struct {
		Configs         *configs.Service
		KeyExport       *keyexport.Service
		Status          *health.Service
		Recoveries      *recovery.Service
		RecoveryUploads *recovery.UploadService
		Backups         *catalog.Service
	}
	Handler struct {
		configs         *configs.Service
		keyExport       *keyexport.Service
		status          *health.Service
		recoveries      *recovery.Service
		recoveryUploads *recovery.UploadService
		backups         *catalog.Service
	}
)

func NewHandler(deps Deps) *Handler {
	return &Handler{
		configs:         deps.Configs,
		keyExport:       deps.KeyExport,
		status:          deps.Status,
		recoveries:      deps.Recoveries,
		recoveryUploads: deps.RecoveryUploads,
		backups:         deps.Backups,
	}
}
