package usecases_physical_postgresql

import (
	"context"
	"log/slog"
	"time"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/lib/clock"
)

type receiverExit int

const (
	receiverCtxCancelled    receiverExit = iota // ctx cancelled — stop the loop, no error
	receiverInternalRestart                     // back pressure / slot stall — respawn promptly
	receiverRetryable                           // non-zero exit that a local respawn may fix (network)
	receiverFatal                               // non-retryable exit — escalate so the supervisor reclaims it on a later tick
	receiverResumeMismatch                      // asked for WAL the server no longer has — realign the resume point and retry
)

type receiverRunResult struct {
	Exit     receiverExit
	RanFor   time.Duration
	FatalErr error
}

func (s *WalStreamSupervisor) spawnAndSupervise(ctx context.Context, logger *slog.Logger) receiverRunResult {
	procCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.restartMu.Lock()
	s.cancelReceiver = cancel
	if s.isRestartRequested {
		cancel()
	}
	s.restartMu.Unlock()
	defer func() {
		s.restartMu.Lock()
		s.cancelReceiver = nil
		s.restartMu.Unlock()
		s.receiverRunStartedAtUnixNano.Store(0)
	}()
	source := pgphysical.NewSource(s.spec.SourceDB.Engine)
	receiver := source.ReceiveWal(procCtx, pgphysical.WalReceiverSpec{
		ClientTools: s.spec.ClientTools, Version: s.spec.SourceDB.Version,
		ApplicationName: pgphysical.WalReceiverApplicationNamePrefix + s.spec.SourceDB.DatabaseID.String(),
		WatchDir:        s.watchDir, SlotName: s.slotName, Clock: clock.NewSystem(), Logger: logger,
		OnStarted: func() { s.receiverRunStartedAtUnixNano.Store(time.Now().UTC().UnixNano()) },
	})
	if ctx.Err() != nil {
		return receiverRunResult{Exit: receiverCtxCancelled, RanFor: receiver.RanFor}
	}
	s.restartMu.Lock()
	isRestartRequested := s.isRestartRequested
	s.restartMu.Unlock()
	if isRestartRequested {
		return receiverRunResult{Exit: receiverInternalRestart, RanFor: receiver.RanFor}
	}

	switch receiver.Disposition {
	case pgphysical.WalReceiverStopped, pgphysical.WalReceiverRetryable:
		return receiverRunResult{Exit: receiverRetryable, RanFor: receiver.RanFor}
	case pgphysical.WalReceiverFatal:
		return receiverRunResult{Exit: receiverFatal, RanFor: receiver.RanFor, FatalErr: receiver.FatalErr}
	case pgphysical.WalReceiverResumeMismatch:
		return receiverRunResult{Exit: receiverResumeMismatch, RanFor: receiver.RanFor}
	}
	panic("unknown WAL receiver disposition")
}
