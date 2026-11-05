package systemhttpapi

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"databasus-backend/internal/engines/clienttools"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/diskusage"
	"databasus-backend/internal/modules/verification"
)

type Deps struct {
	Version         string
	Clock           clock.Source
	Logger          *slog.Logger
	DB              *gorm.DB
	Verification    verification.Module
	ClientTools     *clienttools.Checker
	DataDirectory   string
	BinaryDirectory string
	GetJobs         func(time.Time) []JobHealth
	InspectDisk     func(string) (diskusage.Usage, error)
}

type Service struct {
	version         string
	clock           clock.Source
	logger          *slog.Logger
	verification    verification.Module
	clientTools     *clienttools.Checker
	dataDirectory   string
	binaryDirectory string
	getJobs         func(time.Time) []JobHealth
	inspectDisk     func(string) (diskusage.Usage, error)
	connectivity    *connectivityRepository
}

func NewService(deps Deps) *Service {
	return &Service{
		version: deps.Version, clock: deps.Clock, logger: deps.Logger,
		verification: deps.Verification, clientTools: deps.ClientTools,
		dataDirectory: deps.DataDirectory, binaryDirectory: deps.BinaryDirectory,
		getJobs: deps.GetJobs, inspectDisk: deps.InspectDisk,
		connectivity: &connectivityRepository{db: deps.DB},
	}
}

func (s *Service) GetVersion(context.Context) (GetVersionResponseObject, error) {
	return GetVersion200JSONResponse{Version: s.version}, nil
}
