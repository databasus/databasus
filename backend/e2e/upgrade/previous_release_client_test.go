package upgrade

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"databasus-backend/e2e/upgrade/upgradecheck"
	"databasus-backend/internal/lib/testwait"
)

// This file speaks the API of the previous release. When a change breaks an operation, only
// current_release_checks_test.go follows it; this file catches up in the next release, whose
// predecessor then speaks the new API.

const (
	accountEmail     = "upgrade-owner@databasus.local"
	accountPassword  = "upgrade-owner-password"
	accountName      = "Upgrade Owner"
	workspaceName    = "upgrade workspace"
	storageName      = "upgrade local storage"
	notifierName     = "upgrade webhook"
	notifierURL      = "http://upgrade-webhook.invalid/hook"
	databaseName     = "upgrade source database"
	backupEncryption = "ENCRYPTED"
)

var (
	errBackupNotCompleted = errors.New("the backup did not complete")
	errBackupNotEncrypted = errors.New("the backup is not encrypted")
)

// writtenUpgradeData is what the previous release accepted: the items to find after the upgrade and
// the database and the backup to check after it.
type writtenUpgradeData struct {
	items      []upgradecheck.Item
	databaseID string
	backupID   string
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

type previousStorageRequest struct {
	WorkspaceID  string   `json:"workspaceId"`
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	LocalStorage struct{} `json:"localStorage"`
}

type previousNotifierRequest struct {
	WorkspaceID     string                 `json:"workspaceId"`
	Name            string                 `json:"name"`
	NotifierType    string                 `json:"notifierType"`
	WebhookNotifier previousWebhookRequest `json:"webhookNotifier"`
}

type previousWebhookRequest struct {
	WebhookURL    string `json:"webhookUrl"`
	WebhookMethod string `json:"webhookMethod"`
}

type previousDatabaseRequest struct {
	WorkspaceID       string                       `json:"workspaceId"`
	Name              string                       `json:"name"`
	Type              string                       `json:"type"`
	PostgresqlLogical previousPostgresqlConnection `json:"postgresqlLogical"`
}

type previousPostgresqlConnection struct {
	Version  string `json:"version"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	CPUCount int    `json:"cpuCount"`
}

type previousBackupRequest struct {
	DatabaseID string `json:"database_id"`
}

type previousBackupList struct {
	Backups []previousBackup `json:"backups"`
}

type previousAuditLogList struct {
	AuditLogs []previousAuditLog `json:"auditLogs"`
}

type previousAuditLog struct {
	ID          string  `json:"id"`
	UserID      *string `json:"userId"`
	WorkspaceID *string `json:"workspaceId"`
	Message     string  `json:"message"`
	CreatedAt   string  `json:"createdAt"`
}

type previousBackup struct {
	ID          string  `json:"id"`
	Status      string  `json:"status"`
	FailMessage *string `json:"failMessage"`
	Encryption  string  `json:"encryption"`
}

// writeUpgradeData creates through the previous release what a user has after a while: an account,
// a workspace, a storage, a notifier, a database with an encrypted backup config and one completed
// backup.
func writeUpgradeData(t *testing.T, api *apiClient) (writtenUpgradeData, error) {
	t.Helper()

	ctx := t.Context()

	var signUp previousTokenResponse

	err := api.post(ctx, "/users/signup", previousSignUpRequest{
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

	var storage previousCreatedItem

	err = api.post(ctx, "/storages", previousStorageRequest{
		WorkspaceID:  workspace.ID,
		Type:         "LOCAL",
		Name:         storageName,
		LocalStorage: struct{}{},
	}, &storage)
	if err != nil {
		return writtenUpgradeData{}, fmt.Errorf("create the storage: %w", err)
	}

	var notifier previousCreatedItem

	err = api.post(ctx, "/notifiers", previousNotifierRequest{
		WorkspaceID:     workspace.ID,
		Name:            notifierName,
		NotifierType:    "WEBHOOK",
		WebhookNotifier: previousWebhookRequest{WebhookURL: notifierURL, WebhookMethod: "POST"},
	}, &notifier)
	if err != nil {
		return writtenUpgradeData{}, fmt.Errorf("create the notifier: %w", err)
	}

	sourceConnection := previousPostgresqlConnection{
		Version:  sourceVersion,
		Host:     sourceNetworkAlias,
		Port:     sourcePort,
		Username: sourceUsername,
		Password: sourcePassword,
		Database: sourceDatabaseName,
		CPUCount: 1,
	}

	var database previousCreatedItem

	err = api.post(ctx, "/databases/create", previousDatabaseRequest{
		WorkspaceID:       workspace.ID,
		Name:              databaseName,
		Type:              "POSTGRES_LOGICAL",
		PostgresqlLogical: sourceConnection,
	}, &database)
	if err != nil {
		return writtenUpgradeData{}, fmt.Errorf("create the database: %w", err)
	}

	err = switchOnEncryptedBackups(ctx, api, database.ID, storage.ID)
	if err != nil {
		return writtenUpgradeData{}, err
	}

	backup, err := makeBackup(t, api, database.ID)
	if err != nil {
		return writtenUpgradeData{}, err
	}

	auditEventItems, err := readAuditEventItems(ctx, api)
	if err != nil {
		return writtenUpgradeData{}, err
	}

	return writtenUpgradeData{
		items: append([]upgradecheck.Item{
			{Kind: "account", Name: accountEmail, Values: map[string]string{}},
			{Kind: "workspace", Name: workspaceName, Values: map[string]string{"id": workspace.ID}},
			{Kind: "storage", Name: storageName, Values: map[string]string{"id": storage.ID, "type": "LOCAL"}},
			{Kind: "notifier", Name: notifierName, Values: map[string]string{
				"id":            notifier.ID,
				"notifierType":  "WEBHOOK",
				"webhookUrl":    notifierURL,
				"webhookMethod": "POST",
			}},
			{Kind: "database", Name: databaseName, Values: map[string]string{
				"id":       database.ID,
				"type":     "POSTGRES_LOGICAL",
				"version":  sourceConnection.Version,
				"host":     sourceConnection.Host,
				"port":     strconv.Itoa(sourceConnection.Port),
				"username": sourceConnection.Username,
				"database": sourceConnection.Database,
			}},
			{Kind: "backup config", Name: databaseName, Values: map[string]string{
				"isBackupsEnabled": "true",
				"storageId":        storage.ID,
				"encryption":       backupEncryption,
			}},
			{Kind: "backup", Name: backup.ID, Values: map[string]string{
				"status":     backup.Status,
				"encryption": backupEncryption,
			}},
		}, auditEventItems...),
		databaseID: database.ID,
		backupID:   backup.ID,
	}, nil
}

// The previous release fills a new database's backup config with its defaults on the first read,
// so the test changes only the fields it owns and sends the rest back as read.
func switchOnEncryptedBackups(ctx context.Context, api *apiClient, databaseID, storageID string) error {
	var backupConfig map[string]any

	err := api.get(ctx, "/backup-configs/database/"+databaseID, &backupConfig)
	if err != nil {
		return fmt.Errorf("read the backup config: %w", err)
	}

	backupConfig["isBackupsEnabled"] = true
	backupConfig["storage"] = map[string]any{"id": storageID}
	backupConfig["storageId"] = storageID
	backupConfig["encryption"] = backupEncryption

	err = api.post(ctx, "/backup-configs/save", backupConfig, nil)
	if err != nil {
		return fmt.Errorf("switch on encrypted backups: %w", err)
	}

	return nil
}

func makeBackup(t *testing.T, api *apiClient, databaseID string) (previousBackup, error) {
	t.Helper()

	err := api.post(t.Context(), "/backups", previousBackupRequest{DatabaseID: databaseID}, nil)
	if err != nil {
		return previousBackup{}, fmt.Errorf("start the backup: %w", err)
	}

	var finishedBackup previousBackup

	err = testwait.Poll(t, "the backup of the previous release to finish", func(ctx context.Context) error {
		var backupList previousBackupList

		listErr := api.get(ctx, "/backups?database_id="+databaseID+"&limit=1", &backupList)
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
// instance, which makes it the admin. Every event the account wrote is expected to show its email
// and name after the upgrade, and an event without an author is expected to stay without one.
func readAuditEventItems(ctx context.Context, api *apiClient) ([]upgradecheck.Item, error) {
	var auditLogList previousAuditLogList

	err := api.get(ctx, "/audit-logs/global?limit=1000", &auditLogList)
	if err != nil {
		return nil, fmt.Errorf("list the global audit log: %w", err)
	}

	auditEventItems := make([]upgradecheck.Item, 0, len(auditLogList.AuditLogs))

	for _, auditLog := range auditLogList.AuditLogs {
		createdAt, err := normalizeAuditTime(auditLog.CreatedAt)
		if err != nil {
			return nil, err
		}

		authorEmail, authorName := "", ""
		if auditLog.UserID != nil {
			authorEmail, authorName = accountEmail, accountName
		}

		auditEventItems = append(auditEventItems, upgradecheck.Item{
			Kind: "audit event",
			Name: auditLog.ID,
			Values: map[string]string{
				"message":     auditLog.Message,
				"createdAt":   createdAt,
				"workspaceId": getOptionalValue(auditLog.WorkspaceID),
				"authorEmail": authorEmail,
				"authorName":  authorName,
			},
		})
	}

	return auditEventItems, nil
}
