package physical

import (
	"os"
	"path/filepath"
	"strings"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

// pg_receivewal finalizes a segment only when it contains exactly wal_segment_size bytes.
func IsCompleteSegmentFile(localPath string, segmentSizeBytes int64) bool {
	info, err := os.Stat(localPath)
	return err == nil && !info.IsDir() && info.Size() == segmentSizeBytes
}

func IsPartialWalFile(name string) bool      { return strings.HasSuffix(name, ".partial") }
func IsTimelineHistoryFile(name string) bool { return strings.HasSuffix(name, ".history") }

// Mirrors pg_receivewal's FindStreamingStart: it starts at the segment after the
// highest complete file in --directory, and falls back to the slot's restart_lsn
// only when it finds none (PG 15+).
func GetResumeSegmentNo(
	watchDir string,
	segmentSizeBytes int64,
) (resumeSegmentNo walmath.WalSegmentNo, isResumeSegmentFound bool) {
	entries, err := os.ReadDir(watchDir)
	if err != nil {
		return 0, false
	}

	var highestSegmentNo uint64

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		_, segmentNo, err := walmath.ParseWALFilename(entry.Name(), uint64(segmentSizeBytes))
		if err != nil {
			continue
		}

		if !IsCompleteSegmentFile(filepath.Join(watchDir, entry.Name()), segmentSizeBytes) {
			continue
		}

		if !isResumeSegmentFound || segmentNo > highestSegmentNo {
			highestSegmentNo = segmentNo
			isResumeSegmentFound = true
		}
	}

	if !isResumeSegmentFound {
		return 0, false
	}

	return walmath.WalSegmentNo(highestSegmentNo).Next(), true
}

type WalSegmentFile struct {
	Filename   string
	LocalPath  string
	TimelineID int
	StartLSN   walmath.LSN
	EndLSN     walmath.LSN
	IsComplete bool
}

func GetWalSegmentFiles(directory string, segmentSizeBytes int64) []WalSegmentFile {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil
	}

	segments := make([]WalSegmentFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		timeline, segmentNo, err := walmath.ParseWALFilename(entry.Name(), uint64(segmentSizeBytes))
		if err != nil {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		startLSN := walmath.WalSegmentNo(segmentNo).FirstLSN(uint64(segmentSizeBytes))
		segments = append(segments, WalSegmentFile{
			Filename:   entry.Name(),
			LocalPath:  filepath.Join(directory, entry.Name()),
			TimelineID: int(timeline),
			StartLSN:   startLSN,
			EndLSN:     startLSN + walmath.LSN(segmentSizeBytes),
			IsComplete: info.Size() == segmentSizeBytes,
		})
	}

	return segments
}
