package upgrade

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"testing"

	"databasus-backend/e2e/upgrade/upgradecheck"
	"databasus-backend/internal/lib/testwait"
)

// This file speaks the API of the previous release. When a change breaks an operation, only
// current_release_checks_test.go follows it; this file catches up in the next release, whose
// predecessor then speaks the new API.

const (
	accountEmail        = "upgrade-owner@databasus.local"
	accountPassword     = "upgrade-owner-password"
	accountName         = "Upgrade Owner"
	workspaceName       = "upgrade workspace"
	memberEmail         = "upgrade-member@databasus.local"
	memberName          = "Upgrade Member"
	memberRole          = "WORKSPACE_MEMBER"
	workspaceAdminEmail = "upgrade-admin@databasus.local"
	workspaceAdminName  = "Upgrade Admin"
	storageName         = "upgrade local storage"
	databaseName        = "upgrade source database"
	backupEncryption    = "ENCRYPTED"
	// The backup config written before the upgrade differs from the logical default in every value.
	backupScheduleType        = "WEEKLY"
	backupScheduleTimeOfDay   = "02:30"
	backupScheduleWeekday     = 3
	backupRetentionPolicy     = "GFS"
	backupRetentionGfsHours   = 6
	backupRetentionGfsDays    = 5
	backupRetentionGfsWeeks   = 4
	backupRetentionGfsMonths  = 3
	backupRetentionGfsYears   = 2
	backupMaxFailedTriesCount = 5
	backupNotificationTrigger = "BACKUP_FAILED"
)

var (
	errBackupNotCompleted = errors.New("the backup did not complete")
	errBackupNotEncrypted = errors.New("the backup is not encrypted")
)

// writtenUpgradeData is what the previous release accepted: the items to find after the upgrade and
// the databases and the backup to check after it.
type writtenUpgradeData struct {
	items []upgradecheck.Item
	// databaseIDs are keyed by the database's name.
	databaseIDs        map[string]string
	backupID           string
	verificationID     string
	physicalDatabaseID string
	physicalItems      []upgradecheck.Item
}

type previousSignUpRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type previousTokenResponse struct {
	Token string `json:"token"`
}

type previousCreatedItem struct {
	ID string `json:"id"`
}

type previousWorkspaceRequest struct {
	Name string `json:"name"`
}

type previousMemberRequest struct {
	Email       string `json:"email"`
	Role        string `json:"role"`
	WorkspaceID string `json:"workspaceId"`
}

type previousBackupRequest struct {
	DatabaseID string `json:"databaseId"`
}

type previousBackupList struct {
	Backups []previousBackup `json:"items"`
}

type previousBackup struct {
	ID          string  `json:"id"`
	Status      string  `json:"status"`
	FailMessage *string `json:"failMessage"`
	Encryption  string  `json:"encryption"`
}

// writeUpgradeData creates through the previous release what a user has after a while: an account,
// a workspace with a member and an admin, one storage and one notifier of every
// type, a database of every logical type, the PostgreSQL one using every notifier and the local
// storage with an encrypted backup config, and one completed backup.
func writeUpgradeData(t *testing.T, api *apiClient, sourceHostPort int) (writtenUpgradeData, error) {
	t.Helper()

	ctx := t.Context()

	var signUp previousTokenResponse

	err := api.post(ctx, "/users", previousSignUpRequest{
		Email:    accountEmail,
		Password: accountPassword,
		Name:     accountName,
	}, &signUp)
	if err != nil {
		return writtenUpgradeData{}, fmt.Errorf("sign up: %w", err)
	}

	api.token = signUp.Token

	var workspace previousCreatedItem

	err = api.post(ctx, "/workspaces", previousWorkspaceRequest{Name: workspaceName}, &workspace)
	if err != nil {
		return writtenUpgradeData{}, fmt.Errorf("create the workspace: %w", err)
	}

	err = addMembers(ctx, api, workspace.ID)
	if err != nil {
		return writtenUpgradeData{}, err
	}

	storageItems, localStorageID, err := createStorages(ctx, api, workspace.ID)
	if err != nil {
		return writtenUpgradeData{}, err
	}

	notifierItems, notifierReferences, err := createNotifiers(ctx, api, workspace.ID)
	if err != nil {
		return writtenUpgradeData{}, err
	}

	databases, err := createDatabases(ctx, api, workspace.ID, notifierReferences)
	if err != nil {
		return writtenUpgradeData{}, err
	}

	sourceDatabaseID := databases.ids[databaseName]

	err = switchOnEncryptedBackups(ctx, api, sourceDatabaseID, localStorageID)
	if err != nil {
		return writtenUpgradeData{}, err
	}

	backup, err := makeBackup(t, api, sourceDatabaseID)
	if err != nil {
		return writtenUpgradeData{}, err
	}

	physicalDatabaseID, physicalItems, err := writePhysicalBackup(
		t,
		api,
		physicalSourceSpec{workspaceID: workspace.ID, storageID: localStorageID, hostPort: sourceHostPort},
	)
	if err != nil {
		return writtenUpgradeData{}, err
	}

	verificationID, verificationItems, err := writeVerificationData(ctx, api, verificationSource{
		databaseID: sourceDatabaseID, backupID: backup.ID,
	})
	if err != nil {
		return writtenUpgradeData{}, err
	}

	healthcheckItems, err := writeHealthcheckData(ctx, api, sourceDatabaseID)
	if err != nil {
		return writtenUpgradeData{}, err
	}

	auditEventItems, err := readAuditEventItems(ctx, api)
	if err != nil {
		return writtenUpgradeData{}, err
	}

	return writtenUpgradeData{
		items: slices.Concat([]upgradecheck.Item{
			{Kind: "account", Name: accountEmail, Values: map[string]string{}},
			{Kind: "workspace", Name: workspaceName, Values: map[string]string{"id": workspace.ID}},
			{Kind: "workspace member", Name: accountEmail, Values: map[string]string{"role": "WORKSPACE_OWNER"}},
			{Kind: "workspace member", Name: memberEmail, Values: map[string]string{"role": memberRole}},
			{Kind: "workspace member", Name: workspaceAdminEmail, Values: map[string]string{"role": "WORKSPACE_ADMIN"}},
			{Kind: "backup config", Name: databaseName, Values: map[string]string{
				"isBackupsEnabled":              "true",
				"storageId":                     localStorageID,
				"encryption":                    backupEncryption,
				"settings.type":                 "LOGICAL",
				"settings.schedule.type":        backupScheduleType,
				"settings.schedule.timeOfDay":   backupScheduleTimeOfDay,
				"settings.schedule.weekday":     strconv.Itoa(backupScheduleWeekday),
				"settings.retention.policy":     backupRetentionPolicy,
				"settings.retention.gfsHours":   strconv.Itoa(backupRetentionGfsHours),
				"settings.retention.gfsDays":    strconv.Itoa(backupRetentionGfsDays),
				"settings.retention.gfsWeeks":   strconv.Itoa(backupRetentionGfsWeeks),
				"settings.retention.gfsMonths":  strconv.Itoa(backupRetentionGfsMonths),
				"settings.retention.gfsYears":   strconv.Itoa(backupRetentionGfsYears),
				"settings.isRetryIfFailed":      "true",
				"settings.maxFailedTriesCount":  strconv.Itoa(backupMaxFailedTriesCount),
				"settings.notificationTriggers": backupNotificationTrigger,
			}},
			{Kind: "backup", Name: backup.ID, Values: map[string]string{
				"status":     backup.Status,
				"encryption": backupEncryption,
			}},
			newDatabaseNotifiersItem(notifierItems),
		}, databases.items, storageItems, notifierItems, auditEventItems, verificationItems, healthcheckItems),
		databaseIDs:        databases.ids,
		backupID:           backup.ID,
		verificationID:     verificationID,
		physicalDatabaseID: physicalDatabaseID,
		physicalItems:      physicalItems,
	}, nil
}

func addMembers(ctx context.Context, api *apiClient, workspaceID string) error {
	err := addMember(ctx, api, workspaceID, previousSignUpRequest{
		Email:    memberEmail,
		Password: accountPassword,
		Name:     memberName,
	}, memberRole)
	if err != nil {
		return err
	}

	return addMember(ctx, api, workspaceID, previousSignUpRequest{
		Email:    workspaceAdminEmail,
		Password: accountPassword,
		Name:     workspaceAdminName,
	}, "WORKSPACE_ADMIN")
}

func addMember(
	ctx context.Context,
	api *apiClient,
	workspaceID string,
	signUp previousSignUpRequest,
	role string,
) error {
	err := api.post(ctx, "/users", signUp, nil)
	if err != nil {
		return fmt.Errorf("sign up %s: %w", signUp.Email, err)
	}

	err = api.post(ctx, "/workspace-members",
		previousMemberRequest{Email: signUp.Email, Role: role, WorkspaceID: workspaceID}, nil)
	if err != nil {
		return fmt.Errorf("add %s to the workspace: %w", signUp.Email, err)
	}

	return nil
}

// The previous release fills a new database's backup config with its defaults on the first read. The
// test sends it back with every setting changed from the logical default, so a lost value cannot pass
// for a default after the upgrade.
func switchOnEncryptedBackups(ctx context.Context, api *apiClient, databaseID, storageID string) error {
	var backupConfig map[string]any

	err := api.get(ctx, "/backup-configs/"+databaseID, &backupConfig)
	if err != nil {
		return fmt.Errorf("read the backup config: %w", err)
	}

	backupConfig["isBackupsEnabled"] = true
	backupConfig["storageId"] = storageID
	backupConfig["encryption"] = backupEncryption
	settings, hasSettings := backupConfig["settings"].(map[string]any)
	if !hasSettings {
		return errors.New("previous backup config has no settings")
	}
	settings["schedule"] = map[string]any{
		"type":      backupScheduleType,
		"timeOfDay": backupScheduleTimeOfDay,
		"weekday":   backupScheduleWeekday,
	}
	retention, hasRetention := settings["retention"].(map[string]any)
	if !hasRetention {
		return errors.New("previous logical backup config has no retention")
	}
	retention["policy"] = backupRetentionPolicy
	retention["gfsHours"] = backupRetentionGfsHours
	retention["gfsDays"] = backupRetentionGfsDays
	retention["gfsWeeks"] = backupRetentionGfsWeeks
	retention["gfsMonths"] = backupRetentionGfsMonths
	retention["gfsYears"] = backupRetentionGfsYears
	settings["isRetryIfFailed"] = true
	settings["maxFailedTriesCount"] = backupMaxFailedTriesCount
	settings["notificationTriggers"] = []string{backupNotificationTrigger}

	err = api.send(ctx, http.MethodPut, "/backup-configs/"+databaseID, backupConfig, nil)
	if err != nil {
		return fmt.Errorf("switch on encrypted backups: %w", err)
	}

	return nil
}

func makeBackup(t *testing.T, api *apiClient, databaseID string) (previousBackup, error) {
	t.Helper()

	err := api.post(t.Context(), "/logical-backups", previousBackupRequest{DatabaseID: databaseID}, nil)
	if err != nil {
		return previousBackup{}, fmt.Errorf("start the backup: %w", err)
	}

	var finishedBackup previousBackup

	err = testwait.Poll(t, "the backup of the previous release to finish", func(ctx context.Context) error {
		var backupList previousBackupList

		listErr := api.get(ctx, "/logical-backups?databaseId="+databaseID+"&limit=1", &backupList)
		if listErr != nil {
			return listErr
		}

		if len(backupList.Backups) == 0 {
			return errors.New("no backup is listed yet")
		}

		finishedBackup = backupList.Backups[0]
		if finishedBackup.Status == "IN_PROGRESS" {
			return errors.New("the backup is in progress")
		}

		return nil
	})
	if err != nil {
		return previousBackup{}, err
	}

	if finishedBackup.Status != "COMPLETED" {
		failMessage := ""
		if finishedBackup.FailMessage != nil {
			failMessage = *finishedBackup.FailMessage
		}

		return previousBackup{}, fmt.Errorf(
			"%w: status %s, %s",
			errBackupNotCompleted,
			finishedBackup.Status,
			failMessage,
		)
	}

	if finishedBackup.Encryption != backupEncryption {
		return previousBackup{}, fmt.Errorf("%w: backup %s has encryption %s",
			errBackupNotEncrypted, finishedBackup.ID, finishedBackup.Encryption)
	}

	return finishedBackup, nil
}

// readAuditEventItems lists the global audit log the account sees as the first account of the
// instance, which makes it the admin. Every event with an author is expected to show the email and
// name the previous release showed for it, and an event without an author is expected to stay
// without one.
func readAuditEventItems(ctx context.Context, api *apiClient) ([]upgradecheck.Item, error) {
	return listAuditEventItems(ctx, api)
}

func getOptionalValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
