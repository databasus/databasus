package walstreaming

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

const pendingUploadDirName = "pending-upload"

func movePendingUploadsOutOfResumePath(
	watchDir string,
	resumeFloor walmath.WalSegmentNo,
	segmentSizeBytes int64,
) (movedCount int, err error) {
	entries, err := os.ReadDir(watchDir)
	if err != nil {
		return 0, fmt.Errorf("read wal watch dir: %w", err)
	}

	pendingUploadDir := filepath.Join(watchDir, pendingUploadDirName)
	if err := os.MkdirAll(pendingUploadDir, 0o700); err != nil {
		return 0, fmt.Errorf("create pending upload dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		_, segmentNo, err := walmath.ParseWALFilename(entry.Name(), uint64(segmentSizeBytes))
		if err != nil || walmath.WalSegmentNo(segmentNo) >= resumeFloor {
			continue
		}

		segmentPath := filepath.Join(watchDir, entry.Name())

		if !pgphysical.IsCompleteSegmentFile(segmentPath, segmentSizeBytes) {
			continue
		}

		moveErr := os.Rename(segmentPath, filepath.Join(pendingUploadDir, entry.Name()))
		if moveErr != nil && !os.IsNotExist(moveErr) {
			return movedCount, fmt.Errorf("move %s out of the resume path: %w", entry.Name(), moveErr)
		}

		movedCount++
	}

	return movedCount, nil
}

func (s *Streamer) realignResumePath(ctx context.Context, logger *slog.Logger) {
	probe := s.probeSlot(ctx, logger)
	if !probe.IsReachableWithSlot() {
		return
	}

	state := probe.State

	segmentSizeBytes := walSegmentSizeBytes(s.spec.SourceDB)

	resumeSegmentNo, isResumeSegmentFound := pgphysical.GetResumeSegmentNo(s.watchDir, segmentSizeBytes)
	if !isResumeSegmentFound {
		return
	}

	slotSegmentNo := walmath.NewWalSegmentNo(state.RestartLSN, uint64(segmentSizeBytes))
	if !shouldRealignResumePath(resumePathState{ResumeSegmentNo: resumeSegmentNo, SlotSegmentNo: &slotSegmentNo}) {
		return
	}

	movedCount, err := movePendingUploadsOutOfResumePath(s.watchDir, slotSegmentNo, segmentSizeBytes)
	if err != nil {
		logger.ErrorContext(ctx, "failed to realign wal resume point", slog.Any("error", err))

		return
	}

	logger.WarnContext(
		ctx,
		"realigned wal resume point",
		slog.Any("moved_segment_count", movedCount),
		slog.Any("slot_restart_lsn", state.RestartLSN.String()),
		slog.Any("resume_segment_no", uint64(resumeSegmentNo)),
	)
}

type resumePathState struct {
	ResumeSegmentNo walmath.WalSegmentNo
	SlotSegmentNo   *walmath.WalSegmentNo
}

func shouldRealignResumePath(state resumePathState) bool {
	return state.SlotSegmentNo != nil && state.ResumeSegmentNo < *state.SlotSegmentNo
}
