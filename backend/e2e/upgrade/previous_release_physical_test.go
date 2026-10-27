package upgrade

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"

	"databasus-backend/e2e/upgrade/upgradecheck"
	"databasus-backend/internal/lib/testwait"
)

type physicalSourceSpec struct {
	workspaceID string
	storageID   string
	hostPort    int
}

type previousPhysicalBackupPage struct {
	Backups []previousPhysicalBackup `json:"backups"`
}

type previousPhysicalBackup struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"`
	Status          string  `json:"status"`
	TimelineID      int     `json:"timelineId"`
	StartLSN        string  `json:"startLsn"`
	StopLSN         string  `json:"stopLsn"`
	FromWalFilename string  `json:"fromWalFilename"`
	ToWalFilename   string  `json:"toWalFilename"`
	FailMessage     *string `json:"failMessage"`
}

func writePhysicalBackup(t *testing.T, api *apiClient, source physicalSourceSpec) (string, []upgradecheck.Item, error) {
	t.Helper()
	var createdDatabase previousCreatedItem
	err := api.post(t.Context(), "/databases/create", map[string]any{
		"workspaceId": source.workspaceID, "name": "upgrade physical database", "type": "POSTGRES_PHYSICAL",
		"postgresqlPhysical": map[string]any{
			"version": sourceVersion, "host": sourceNetworkAlias, "port": sourcePort, "username": sourceUsername,
			"password": sourcePassword, "sslMode": "disable", "backupType": "FULL_INCREMENTAL_WAL_STREAM",
		}, "notifiers": []any{},
	}, &createdDatabase)
	if err != nil {
		return "", nil, fmt.Errorf("create the physical database: %w", err)
	}
	databaseID := createdDatabase.ID
	var config map[string]any
	err = api.get(t.Context(), "/backup-configs/physical/database/"+databaseID, &config)
	if err != nil {
		return "", nil, err
	}
	config["isBackupsEnabled"] = true
	config["storage"] = map[string]any{"id": source.storageID}
	err = api.post(t.Context(), "/backup-configs/physical/save", config, nil)
	if err != nil {
		return "", nil, fmt.Errorf("enable physical backups: %w", err)
	}
	err = api.post(
		t.Context(),
		"/backups/physical/database/"+databaseID+"/trigger",
		map[string]string{"type": "full"},
		nil,
	)
	if err != nil {
		return "", nil, err
	}
	var fullBackup previousPhysicalBackup
	err = testwait.Poll(t, "the previous release's full backup to finish", func(ctx context.Context) error {
		var page previousPhysicalBackupPage
		if readErr := api.get(
			ctx,
			"/backups/physical/database/"+databaseID+"/backups?type=FULL&limit=100",
			&page,
		); readErr != nil {
			return readErr
		}
		for _, backup := range page.Backups {
			if backup.Type == "FULL" && backup.Status != "IN_PROGRESS" {
				fullBackup = backup
				return nil
			}
		}
		return errors.New("no finished physical full backup is listed")
	})
	if err != nil {
		return "", nil, err
	}
	if fullBackup.Status != "COMPLETED" {
		return "", nil, fmt.Errorf(
			"physical full backup %s ended with %s: %s",
			fullBackup.ID,
			fullBackup.Status,
			getOptionalValue(fullBackup.FailMessage),
		)
	}
	items := []upgradecheck.Item{physicalFullBackupItem(fullBackup.GetSnapshot())}
	var forcedWalFilename string
	err = executeOnSourceDatabase(t.Context(), source.hostPort, sourceDatabaseName, func(connection *pgx.Conn) error {
		_, writeErr := connection.Exec(
			t.Context(),
			"INSERT INTO upgrade_rows (label) VALUES ('physical WAL before the upgrade')",
		)
		if writeErr != nil {
			return writeErr
		}
		if writeErr = connection.QueryRow(t.Context(), "SELECT pg_walfile_name(pg_current_wal_lsn())").
			Scan(&forcedWalFilename); writeErr != nil {
			return writeErr
		}
		_, writeErr = connection.Exec(t.Context(), "SELECT pg_switch_wal()")
		return writeErr
	})
	if err != nil {
		return "", nil, err
	}
	err = testwait.Poll(
		t,
		"the previous release to archive WAL after its full backup",
		func(ctx context.Context) error {
			var page previousPhysicalBackupPage
			if readErr := api.get(
				ctx,
				"/backups/physical/database/"+databaseID+"/backups?type=WAL&limit=100",
				&page,
			); readErr != nil {
				return readErr
			}
			for _, backup := range page.Backups {
				if backup.Type != "WAL" || backup.FromWalFilename == "" || forcedWalFilename < backup.FromWalFilename ||
					forcedWalFilename > backup.ToWalFilename {
					continue
				}
				segments, segmentErr := physicalWALSegmentItems(backup.GetSnapshot())
				if segmentErr != nil {
					return segmentErr
				}
				items = append(items, segments...)
				return nil
			}
			return errors.New("no archived WAL range is listed")
		},
	)
	return databaseID, items, err
}

func (backup previousPhysicalBackup) GetSnapshot() physicalBackupSnapshot {
	return physicalBackupSnapshot{
		ID: backup.ID, Type: backup.Type, Status: backup.Status, TimelineID: backup.TimelineID,
		StartLSN: backup.StartLSN, StopLSN: backup.StopLSN,
		FromWalFilename: backup.FromWalFilename, ToWalFilename: backup.ToWalFilename,
	}
}
