package telemetry

import (
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/modules/telemetry/internal/report"
)

type Deps = report.Deps

type Module interface{ GetJobs() []background.Job }

func New(deps Deps) Module { return report.New(deps) }
