package usecases_physical_postgresql

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

// pg_receivewal reads only the top level of --directory, so a pending segment
// below the slot's restart_lsn drags the resume point into WAL the server has
// already recycled. Moving rather than deleting keeps those segments archivable:
// they are valid WAL of the older chain.
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

func (s *WalStreamSupervisor) realignResumePath(ctx context.Context, logger *slog.Logger) {
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
		logger.ErrorContext(ctx, "failed to realign wal resume point", "error", err)

		return
	}

	logger.WarnContext(
		ctx,
		fmt.Sprintf("realigned wal resume point, moved %d segments out of the resume path", movedCount),
		"slot_restart_lsn", state.RestartLSN.String(),
		"resume_segment_no", uint64(resumeSegmentNo),
	)
}

type resumePathState struct {
	ResumeSegmentNo walmath.WalSegmentNo
	SlotSegmentNo   *walmath.WalSegmentNo
}

func shouldRealignResumePath(state resumePathState) bool {
	return state.SlotSegmentNo != nil && state.ResumeSegmentNo < *state.SlotSegmentNo
}
