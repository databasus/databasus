package walstreaming

import (
	"context"
	"log/slog"
	"time"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
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

func (s *Streamer) spawnAndSupervise(ctx context.Context, logger *slog.Logger) receiverRunResult {
	procCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.restartLatch.SetReceiver(cancel)
	defer func() {
		s.restartLatch.ClearReceiver()
		s.receiverRunStartedAtUnixNano.Store(0)
	}()

	source := pgphysical.NewSource(s.spec.SourceDB.Engine)
	receiver := source.ReceiveWal(procCtx, pgphysical.WalReceiverSpec{
		ClientTools: s.spec.ClientTools, Version: s.spec.SourceDB.Version,
		ApplicationName: pgphysical.WalReceiverApplicationNamePrefix + s.spec.SourceDB.DatabaseID.String(),
		WatchDir:        s.watchDir, SlotName: s.slotName, Clock: s.spec.Clock, Logger: logger,
		OnStarted: func() { s.receiverRunStartedAtUnixNano.Store(s.spec.Clock.Now().UnixNano()) },
	})

	if ctx.Err() != nil {
		return receiverRunResult{Exit: receiverCtxCancelled, RanFor: receiver.RanFor}
	}
	if s.restartLatch.IsRequested() {
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
