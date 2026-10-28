package physical

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/download"
	"databasus-backend/internal/modules/backups/internal/physical/chains"
)

type RestoreStream struct {
	DatabaseID uuid.UUID
	Write      func(io.Writer) error
	Close      func()
}

func (s *Service) OpenRestoreStream(ctx context.Context, token string) (RestoreStream, error) {
	restoreToken, userStream, err := s.restoreTokenService.ValidateAndConsumeRestoreToken(ctx, token)
	if err != nil {
		return RestoreStream{}, err
	}
	var restoreSet *chains.RestoreSet
	if restoreToken.BackupID != nil {
		restoreSet, err = s.chains.ResolveRestoreSetForBackup(
			ctx,
			restoreToken.DatabaseID,
			*restoreToken.BackupID,
		)
	} else {
		restoreSet, err = s.chains.ResolveRestoreSet(ctx, restoreToken.DatabaseID, restoreToken.TargetTime)
	}
	if err != nil {
		_ = userStream.Release(context.WithoutCancel(ctx))
		return RestoreStream{}, err
	}
	streamCtx, cancel := context.WithCancel(ctx)
	stopShutdown := context.AfterFunc(s.appCtx, cancel)
	s.runs.Go(streamCtx, "physical_restore_stream_heartbeat", func(ctx context.Context) error {
		ticker := time.NewTicker(download.UserStreamHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				if err := userStream.Refresh(ctx); err != nil && ctx.Err() == nil {
					s.logger.ErrorContext(ctx, "failed to refresh physical restore stream", slog.Any("error", err))
				}
			}
		}
	})
	var closeOnce sync.Once
	return RestoreStream{
		DatabaseID: restoreToken.DatabaseID,
		Write: func(writer io.Writer) error {
			return s.restoreStreamWriter.Write(streamCtx, writer, restoreSet, s.masterKey)
		},
		Close: func() {
			closeOnce.Do(func() {
				stopShutdown()
				cancel()
				if err := userStream.Release(context.WithoutCancel(ctx)); err != nil {
					s.logger.ErrorContext(ctx, "failed to release physical restore stream", slog.Any("error", err))
				}
			})
		},
	}, nil
}
