package physicaltest

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/app/apptest"
)

type backupRow struct {
	ID               uuid.UUID `json:"id"`
	Type             string    `json:"type"`
	Status           string    `json:"status"`
	TimelineID       int       `json:"timelineId"`
	StartLSN         string    `json:"startLsn"`
	StopLSN          string    `json:"stopLsn"`
	WalSegmentsCount *int      `json:"walSegmentsCount"`
	HasGapBefore     bool      `json:"hasGapBefore"`
	FailMessage      *string   `json:"failMessage"`
	CreatedAt        time.Time `json:"createdAt"`
}

func (team Team) listBackups(t *testing.T, backupType string) []backupRow {
	t.Helper()

	var response struct {
		Backups []backupRow `json:"items"`
	}
	team.call(
		t,
		physicalResponseSpec{
			Call:        apptest.PhysicalListRequest(team.Database.Id, "?limit=1000&type="+backupType),
			Status:      http.StatusOK,
			Destination: &response,
		},
	)
	return response.Backups
}
