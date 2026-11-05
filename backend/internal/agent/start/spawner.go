package start

import (
	"context"

	"databasus-backend/internal/agent/container"
	"databasus-backend/internal/agent/runner"
)

type containerManagerSpawner struct {
	containerManager *container.Manager
}

func (s containerManagerSpawner) Spawn(
	ctx context.Context, req runner.SpawnRequest,
) (runner.JobContainer, error) {
	jobContainer, err := s.containerManager.Spawn(ctx, container.SpawnRequest{
		PgMajor:            req.PgMajor,
		CPUPerJob:          req.CPUPerJob,
		RAMMbPerJob:        req.RAMMbPerJob,
		VerificationID:     req.VerificationID,
		TimescaledbVersion: req.TimescaledbVersion,
	})
	if err != nil {
		return nil, err
	}

	return jobContainer, nil
}
