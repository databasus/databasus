package report

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/clock"
)

type Instance struct {
	InstanceID  string `json:"instanceID"`
	InstalledAt string `json:"installedAt"`
}

type IdentityLoader struct {
	path   string
	logger *slog.Logger
	clock  clock.Source
}

func NewIdentityLoader(deps Deps) *IdentityLoader {
	return &IdentityLoader{path: deps.IdentityPath, logger: deps.Logger, clock: deps.Clock}
}

func (l *IdentityLoader) LoadOrCreate(ctx context.Context) (*Instance, bool) {
	data, err := os.ReadFile(l.path)
	if err == nil {
		var instance Instance

		if err := json.Unmarshal(data, &instance); err != nil {
			l.logger.WarnContext(ctx, "telemetry instance file is corrupt; skipping telemetry",
				slog.String("path", l.path), slog.Any("error", err))
			return nil, false
		}

		if instance.InstanceID == "" {
			l.logger.WarnContext(ctx, "telemetry instance file is missing instanceID; skipping telemetry",
				slog.String("path", l.path))
			return nil, false
		}

		return &instance, true
	}

	if !os.IsNotExist(err) {
		l.logger.WarnContext(ctx, "failed to read telemetry instance file; skipping telemetry",
			slog.String("path", l.path), slog.Any("error", err))
		return nil, false
	}

	instance := &Instance{
		InstanceID:  uuid.New().String(),
		InstalledAt: l.clock.Now().UTC().Format("2006-01-02"),
	}

	encoded, err := json.Marshal(instance)
	if err != nil {
		l.logger.WarnContext(ctx, "failed to encode telemetry instance file; skipping telemetry",
			slog.String("path", l.path), slog.Any("error", err))
		return nil, false
	}

	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		l.logger.WarnContext(ctx, "failed to create telemetry instance file directory; skipping telemetry",
			slog.String("path", l.path), slog.Any("error", err))
		return nil, false
	}

	file, err := os.OpenFile(l.path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		l.logger.WarnContext(ctx, "failed to create telemetry identity", slog.Any("error", err))
		return nil, false
	}
	_, writeError := file.Write(encoded)
	closeError := file.Close()
	if writeError != nil || closeError != nil {
		l.logger.WarnContext(
			ctx,
			"failed to persist telemetry identity",
			slog.Any("write_error", writeError),
			slog.Any("close_error", closeError),
		)
		return nil, false
	}

	return instance, true
}
