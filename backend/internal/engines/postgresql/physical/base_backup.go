package physical

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/process"
)

const clientToolKillAfter = 10 * time.Second

type BaseBackupSpec struct {
	ClientTools             pgshared.ClientTools
	Version                 pgshared.Version
	Credentials             *pgshared.CredentialTempFiles
	Label                   string
	Codec                   Codec
	IncrementalManifestPath string
	SystemID                uint64
	Clock                   clock.Source
	FirstByteTimeout        time.Duration
	StallTimeout            time.Duration
	StallPollInterval       time.Duration
	ProgressInterval        time.Duration
	ProgressListener        func(completedMb float64, elapsedMs int64)
	// ReleaseOutput makes a write blocked in the output return when the stream ends early, such as
	// on a stall. Only the output's owner can do that, and the tool's run waits for that write.
	ReleaseOutput func(cause error)
	// FlushOutput writes what the output still buffers after the tool succeeded. It runs inside the
	// stream, so the stall limit and ReleaseOutput also cover that last write.
	FlushOutput func() error
	Logger      *slog.Logger
}

type BaseBackupResult struct {
	StartLSN     walmath.LSN
	StopLSN      walmath.LSN
	TimelineID   int
	Manifest     []byte
	StreamedSize int64
}

func (s *Source) StreamBaseBackup(
	ctx context.Context,
	spec BaseBackupSpec,
	output io.Writer,
) (BaseBackupResult, error) {
	toolPath := spec.ClientTools.GetExecutable(spec.Version, pgshared.ExecutablePgBasebackup)
	if _, err := exec.LookPath(toolPath); err != nil {
		return BaseBackupResult{}, fmt.Errorf("pg_basebackup binary not found at %s: %w", toolPath, err)
	}

	streamCtx, cancelStream := context.WithCancelCause(ctx)
	defer cancelStream(nil)
	group := background.New(background.Deps{Logger: spec.Logger})
	manifestReader, manifestWriter := io.Pipe()
	counter := &byteCounter{writer: output}
	startedAt := spec.Clock.Now()
	var files []manifestFileEntry
	var manifestErr error
	manifestDone := make(chan struct{})

	group.Go(streamCtx, "base backup manifest", func(ctx context.Context) error {
		defer close(manifestDone)
		defer func() { _ = manifestReader.Close() }()
		manifestErr = background.RunRecovering(ctx, func(context.Context) error {
			var walkErr error
			files, walkErr = walkTarForManifest(manifestReader, spec.Codec)
			return walkErr
		})
		_ = manifestReader.CloseWithError(manifestErr)
		if manifestErr != nil && !isTruncatedManifest(manifestErr) {
			cancelStream(errManifestWalk)
		}
		return nil
	})

	group.Go(streamCtx, "base backup stall watcher", func(ctx context.Context) error {
		watchByteStall(ctx, byteStallSpec{Counter: counter, Backup: spec, StartedAt: startedAt, OnStall: func() {
			spec.Logger.WarnContext(ctx, "byte-stall timeout tripped; terminating pg_basebackup")
			cancelStream(errByteStall)
		}})
		return nil
	})

	if spec.ProgressListener != nil {
		group.Go(streamCtx, "base backup progress", func(ctx context.Context) error {
			reportProgress(ctx, progressSpec{Counter: counter, Backup: spec, StartedAt: startedAt})
			return nil
		})
	}

	stopReleasingOutput := releaseOutputWhenStreamEnds(streamCtx, spec)

	var stderr bytes.Buffer
	running, startErr := process.Start(streamCtx, process.Spec{
		Path: toolPath, Args: s.getBaseBackupArguments(spec), Env: s.getClientEnvironment(spec.Credentials),
		Stdout: io.MultiWriter(counter, manifestWriter), Stderr: &stderr,
		StopSignal: syscall.SIGINT, KillAfter: clientToolKillAfter,
	})

	var runErr error
	if startErr != nil {
		runErr = startErr
	} else {
		runErr = running.Wait()
	}

	if runErr == nil && spec.FlushOutput != nil {
		runErr = spec.FlushOutput()
	}

	stopReleasingOutput()
	_ = manifestWriter.Close()
	<-manifestDone
	cause := context.Cause(streamCtx)
	cancelStream(nil)
	_ = group.Wait(context.WithoutCancel(ctx))
	stderrBytes := stderr.Bytes()

	failure := &BaseBackupError{Kind: BaseBackupToolFailed, Cause: getClientExitError(runErr), Stderr: stderrBytes}
	switch {
	case errors.Is(cause, errByteStall):
		failure.Kind = BaseBackupStalled
		failure.Cause = errByteStall
	case isCompressionUnsupportedError(stderrBytes):
		failure.Kind = BaseBackupCodecRejected
	case errors.Is(cause, errManifestWalk):
		failure.Kind = BaseBackupManifestBroken
		failure.Cause = fmt.Errorf("walk tar for manifest: %w", manifestErr)
	case cause != nil:
		failure.Kind = BaseBackupContextEnded
		failure.Cause = cause
	case runErr != nil:
		if startErr != nil {
			return BaseBackupResult{}, fmt.Errorf("start pg_basebackup: %w", startErr)
		}
	case manifestErr != nil:
		failure.Kind = BaseBackupManifestBroken
		failure.Cause = fmt.Errorf("walk tar for manifest: %w", manifestErr)
	default:
		backup, err := buildBaseBackupResult(
			baseBackupManifestSpec{
				Files:        files,
				SystemID:     spec.SystemID,
				Stderr:       stderrBytes,
				StreamedSize: counter.GetBytesWritten(),
			},
		)
		if err == nil {
			return backup, nil
		}
		failure.Kind = BaseBackupManifestBroken
		failure.Cause = err
	}

	return BaseBackupResult{}, failure
}

func releaseOutputWhenStreamEnds(streamCtx context.Context, spec BaseBackupSpec) (stop func()) {
	if spec.ReleaseOutput == nil {
		return func() {}
	}

	stopAfterFunc := context.AfterFunc(streamCtx, func() { spec.ReleaseOutput(context.Cause(streamCtx)) })

	return func() { stopAfterFunc() }
}

func (s *Source) getBaseBackupArguments(spec BaseBackupSpec) []string {
	args := []string{
		"--pgdata=-",
		"--format=tar",
		"--compress=" + spec.Codec.GetCompressArgument(),
		"--wal-method=fetch",
		"--checkpoint=fast",
	}

	if spec.IncrementalManifestPath != "" {
		args = append(args, "--incremental="+spec.IncrementalManifestPath)
	}

	return append(
		args,
		"--no-manifest",
		"--label="+spec.Label,
		"--no-password",
		"--verbose",
		"-h",
		s.settings.Host,
		"-p",
		strconv.Itoa(s.settings.Port),
		"-U",
		s.settings.Username,
	)
}

type baseBackupManifestSpec struct {
	Files        []manifestFileEntry
	SystemID     uint64
	Stderr       []byte
	StreamedSize int64
}

func buildBaseBackupResult(spec baseBackupManifestSpec) (BaseBackupResult, error) {
	startLSN, stopLSN, timelineID, err := parseLsnsFromStderr(spec.Stderr)
	if err != nil {
		return BaseBackupResult{}, fmt.Errorf("parse pg_basebackup LSNs: %w", err)
	}

	if spec.SystemID == 0 {
		return BaseBackupResult{}, errManifestSystemIDZero
	}
	manifest, err := serializeManifest(
		serializeManifestInput{
			Files:      spec.Files,
			SystemID:   spec.SystemID,
			TimelineID: timelineID,
			StartLSN:   startLSN,
			StopLSN:    stopLSN,
		},
	)
	if err != nil {
		return BaseBackupResult{}, fmt.Errorf("serialize manifest: %w", err)
	}

	return BaseBackupResult{
		StartLSN:     startLSN,
		StopLSN:      stopLSN,
		TimelineID:   timelineID,
		Manifest:     manifest,
		StreamedSize: spec.StreamedSize,
	}, nil
}

func isTruncatedManifest(err error) bool {
	return errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF)
}
