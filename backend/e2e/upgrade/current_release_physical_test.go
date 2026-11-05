package upgrade

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"databasus-backend/e2e/upgrade/upgradecheck"
)

type currentPhysicalBackupPage struct {
	Backups []currentPhysicalBackup `json:"items"`
}

type currentPhysicalBackup struct {
	RootFullBackupID          string `json:"rootFullBackupId"`
	ParentIncrementalBackupID string `json:"parentIncrementalBackupId"`
	ID                        string `json:"id"`
	Type                      string `json:"type"`
	Status                    string `json:"status"`
	TimelineID                int    `json:"timelineId"`
	StartLSN                  string `json:"startLsn"`
	StopLSN                   string `json:"stopLsn"`
	FromWalFilename           string `json:"fromWalFilename"`
	ToWalFilename             string `json:"toWalFilename"`
}

type currentPhysicalRestoreToken struct {
	Token string `json:"token"`
}

func checkPhysicalBackup(ctx context.Context, api *apiClient, written writtenUpgradeData) error {
	var page currentPhysicalBackupPage
	err := api.get(ctx, "/physical-backups?databaseId="+written.physicalDatabaseID+"&limit=100", &page)
	if err != nil {
		return fmt.Errorf("list physical backups after the upgrade: %w", err)
	}

	listedItems := []upgradecheck.Item{}
	for _, backup := range page.Backups {
		snapshot := physicalBackupSnapshot(backup)
		switch backup.Type {
		case "FULL":
			listedItems = append(listedItems, physicalFullBackupItem(snapshot))
		case "INCREMENTAL":
			listedItems = append(listedItems, physicalIncrementalBackupItem(snapshot))
		case "WAL":
			segments, segmentErr := physicalWALSegmentItems(snapshot)
			if segmentErr != nil {
				return segmentErr
			}
			listedItems = append(listedItems, segments...)
		}
	}

	if err = upgradecheck.Compare(written.physicalItems, listedItems); err != nil {
		return err
	}

	var token currentPhysicalRestoreToken
	err = api.post(
		ctx,
		"/physical-restore-tokens",
		map[string]any{"type": "POINT_IN_TIME", "databaseId": written.physicalDatabaseID},
		&token,
	)
	if err != nil {
		return fmt.Errorf("issue a physical restore token after the upgrade: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		api.baseURL+"/physical-restore-sets?token="+url.QueryEscape(token.Token),
		nil,
	)
	if err != nil {
		return err
	}

	response, err := api.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("physical restore stream answered %d", response.StatusCode)
	}

	archive := tar.NewReader(response.Body)
	hasFullBackup, hasIncrementalBackup, hasPreUpgradeWAL := false, false, false
	for {
		entry, readErr := archive.Next()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read the physical restore archive: %w", readErr)
		}

		if strings.HasPrefix(entry.Name, "full/") && entry.Size > 0 {
			hasFullBackup = true
		}
		if strings.HasPrefix(entry.Name, "incr-1/") && entry.Size > 0 {
			hasIncrementalBackup = true
		}
		for _, item := range written.physicalItems {
			if item.Kind == "physical WAL segment" && entry.Name == "wal/"+item.Name+".zst" && entry.Size > 0 {
				hasPreUpgradeWAL = true
			}
		}

		if _, readErr = io.Copy(io.Discard, archive); readErr != nil {
			return fmt.Errorf("read the physical restore entry %s: %w", entry.Name, readErr)
		}
	}

	if _, err = io.Copy(io.Discard, response.Body); err != nil {
		return fmt.Errorf("finish the physical restore stream: %w", err)
	}

	if !hasFullBackup || !hasIncrementalBackup || !hasPreUpgradeWAL {
		return fmt.Errorf(
			"physical restore stream: full backup present = %t, incremental backup present = %t, pre-upgrade WAL present = %t",
			hasFullBackup,
			hasIncrementalBackup,
			hasPreUpgradeWAL,
		)
	}
	return nil
}
