package walstreaming_test

import (
	"net/http"
	"testing"
	"time"

	"databasus-backend/internal/app/apptest"
)

type streamStatistics struct {
	Streamer struct {
		State            string     `json:"state"`
		SessionStartedAt *time.Time `json:"sessionStartedAt"`
		FailingSince     *time.Time `json:"failingSince"`
		RestartsLastHour int        `json:"restartsLastHour"`
	} `json:"streamer"`
	Current *struct {
		SampledAt                time.Time  `json:"sampledAt"`
		SessionStartedAt         time.Time  `json:"sessionStartedAt"`
		SourceState              string     `json:"sourceState"`
		SourceUnreachableSince   *time.Time `json:"sourceUnreachableSince"`
		SourceCurrentLSN         *string    `json:"sourceCurrentLsn"`
		SourceWrittenBytes       *int64     `json:"sourceWrittenBytes"`
		ArchivedSegments         int        `json:"archivedSegments"`
		ArchivedWalBytes         int64      `json:"archivedWalBytes"`
		ArchivedStoredBytes      int64      `json:"archivedStoredBytes"`
		WalSegmentSizeBytes      int64      `json:"walSegmentSizeBytes"`
		HighestConcurrentUploads int        `json:"highestConcurrentUploads"`
		SegmentWriteAvgMillis    *int64     `json:"segmentWriteAvgMillis"`
		SegmentWriteMaxMillis    *int64     `json:"segmentWriteMaxMillis"`
		SlotWalStatus            *string    `json:"slotWalStatus"`
		SlotLagBytes             *int64     `json:"slotLagBytes"`
		SlotExtraWalSince        *time.Time `json:"slotExtraWalSince"`
		ArchiveStalledSince      *time.Time `json:"archiveStalledSince"`
		LocalQueueBytes          int64      `json:"localQueueBytes"`
		LocalQueuePauseBytes     int64      `json:"localQueuePauseBytes"`
		ReceiverRestarts         int        `json:"receiverRestarts"`
		ReceiverPauses           int        `json:"receiverPauses"`
		IsReceiverRunning        bool       `json:"isReceiverRunning"`
		IsForcedRotationDenied   bool       `json:"isForcedRotationDenied"`
	} `json:"current"`
}

func (team *walTeam) statistics(t *testing.T) streamStatistics {
	t.Helper()
	var statistics streamStatistics
	team.call(
		t,
		apptest.PhysicalStatisticsRequest(team.Database.Id, "HOUR"),
		http.StatusOK,
		&statistics,
	)
	return statistics
}
