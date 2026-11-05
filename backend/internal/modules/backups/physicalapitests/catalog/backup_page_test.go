package catalog_test

import (
	"time"

	"github.com/google/uuid"
)

type backupPage struct {
	Backups      []backupRow `json:"items"`
	TotalUsageMb float64     `json:"totalUsageMb"`
	NextCursor   string      `json:"nextCursor"`
}
type backupRow struct {
	ID                        uuid.UUID  `json:"id"`
	Type                      string     `json:"type"`
	Status                    string     `json:"status"`
	TimelineID                int        `json:"timelineId"`
	StartLSN                  string     `json:"startLsn"`
	StopLSN                   string     `json:"stopLsn"`
	RootFullBackupID          *uuid.UUID `json:"rootFullBackupId"`
	ParentIncrementalBackupID *uuid.UUID `json:"parentIncrementalBackupId"`
	FromWalFilename           *string    `json:"fromWalFilename"`
	ToWalFilename             *string    `json:"toWalFilename"`
	WalSegmentsCount          *int       `json:"walSegmentsCount"`
	HasGapBefore              bool       `json:"hasGapBefore"`
	SizeMb                    float64    `json:"sizeMb"`
	FailMessage               *string    `json:"failMessage"`
	CreatedAt                 time.Time  `json:"createdAt"`
	CompletedAt               *time.Time `json:"completedAt"`
}
