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

// This file speaks the API of the current code, so it changes together with any operation the
// current code breaks.

var (
	errSignInIncomplete     = errors.New("the sign-in did not complete")
	errWorkspaceNotListed   = errors.New("the workspace is not listed")
	errRestoreNotCompleted  = errors.New("the restore did not complete")
	errBackupConfigIsAbsent = errors.New("the backup config names no storage")
)

type currentSignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type currentSignInResponse struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type currentWorkspaceList struct {
	Workspaces []currentWorkspace `json:"workspaces"`
}

type currentWorkspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type currentStorage struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type currentNotifier struct {
	ID              string                  `json:"id"`
	Name            string                  `json:"name"`
	NotifierType    string                  `json:"notifierType"`
	WebhookNotifier *currentWebhookNotifier `json:"webhookNotifier"`
}

type currentWebhookNotifier struct {
	WebhookURL    string `json:"webhookUrl"`
	WebhookMethod string `json:"webhookMethod"`
}

type currentDatabase struct {
	ID                string                       `json:"id"`
	Name              string                       `json:"name"`
	Type              string                       `json:"type"`
	PostgresqlLogical *currentPostgresqlConnection `json:"postgresqlLogical"`
}

type currentPostgresqlConnection struct {
	Version  string  `json:"version"`
	Host     string  `json:"host"`
	Port     int     `json:"port"`
	Username string  `json:"username"`
	Password string  `json:"password,omitempty"`
	Database *string `json:"database"`
	CPUCount int     `json:"cpuCount,omitempty"`
}

type currentBackupConfig struct {
	IsBackupsEnabled bool    `json:"isBackupsEnabled"`
	StorageID        *string `json:"storageId"`
	Encryption       string  `json:"encryption"`
}

type currentBackupList struct {
	Backups []currentBackup `json:"backups"`
}

type currentBackup struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Encryption string `json:"encryption"`
}

type currentRestoreRequest struct {
	PostgresqlDatabase currentPostgresqlConnection `json:"postgresqlDatabase"`
}

type currentRestore struct {
	Status      string  `json:"status"`
	FailMessage *string `json:"failMessage"`
}

// readUpgradeData signs in with the password set before the upgrade and lists every item the way
// the current code returns it.
func readUpgradeData(ctx context.Context, api *apiClient) ([]upgradecheck.Item, error) {
	var signIn currentSignInResponse

	err := api.post(ctx, "/users/signin", currentSignInRequest{Email: accountEmail, Password: accountPassword}, &signIn)
	if err != nil {
		return nil, fmt.Errorf("sign in with the password set before the upgrade: %w", err)
	}

	if signIn.Token == "" {
		return nil, fmt.Errorf("%w for %s", errSignInIncomplete, accountEmail)
	}

	api.token = signIn.Token
	listedItems := []upgradecheck.Item{{Kind: "account", Name: signIn.Email, Values: map[string]string{}}}

	var workspaceList currentWorkspaceList

	err = api.get(ctx, "/workspaces", &workspaceList)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}

	workspaceID := ""

	for _, workspace := range workspaceList.Workspaces {
		listedItems = append(listedItems, upgradecheck.Item{
			Kind: "workspace", Name: workspace.Name, Values: map[string]string{"id": workspace.ID},
		})

		if workspace.Name == workspaceName {
			workspaceID = workspace.ID
		}
	}

	if workspaceID == "" {
		return nil, fmt.Errorf("%w: %q", errWorkspaceNotListed, workspaceName)
	}

	workspaceItems, err := readWorkspaceItems(ctx, api, workspaceID)
	if err != nil {
		return nil, err
	}

	return append(listedItems, workspaceItems...), nil
}

func readWorkspaceItems(ctx context.Context, api *apiClient, workspaceID string) ([]upgradecheck.Item, error) {
	var listedItems []upgradecheck.Item

	var storages []currentStorage

	err := api.get(ctx, "/storages?workspace_id="+workspaceID, &storages)
	if err != nil {
		return nil, fmt.Errorf("list storages: %w", err)
	}

	for _, storage := range storages {
		listedItems = append(listedItems, upgradecheck.Item{
			Kind: "storage", Name: storage.Name, Values: map[string]string{"id": storage.ID, "type": storage.Type},
		})
	}

	var notifiers []currentNotifier

	err = api.get(ctx, "/notifiers?workspace_id="+workspaceID, &notifiers)
	if err != nil {
		return nil, fmt.Errorf("list notifiers: %w", err)
	}

	for _, notifier := range notifiers {
		listedItems = append(listedItems, newNotifierItem(notifier))
	}

	var databases []currentDatabase

	err = api.get(ctx, "/databases?workspace_id="+workspaceID, &databases)
	if err != nil {
		return nil, fmt.Errorf("list databases: %w", err)
	}

	for _, database := range databases {
		databaseItems, readErr := readDatabaseItems(ctx, api, database)
		if readErr != nil {
			return nil, readErr
		}

		listedItems = append(listedItems, databaseItems...)
	}

	return listedItems, nil
}

func newNotifierItem(notifier currentNotifier) upgradecheck.Item {
	values := map[string]string{"id": notifier.ID, "notifierType": notifier.NotifierType}

	if notifier.WebhookNotifier != nil {
		values["webhookUrl"] = notifier.WebhookNotifier.WebhookURL
		values["webhookMethod"] = notifier.WebhookNotifier.WebhookMethod
	}

	return upgradecheck.Item{Kind: "notifier", Name: notifier.Name, Values: values}
}

func readDatabaseItems(ctx context.Context, api *apiClient, database currentDatabase) ([]upgradecheck.Item, error) {
	databaseValues := map[string]string{"id": database.ID, "type": database.Type}

	if connection := database.PostgresqlLogical; connection != nil {
		databaseValues["version"] = connection.Version
		databaseValues["host"] = connection.Host
		databaseValues["port"] = strconv.Itoa(connection.Port)
		databaseValues["username"] = connection.Username

		if connection.Database != nil {
			databaseValues["database"] = *connection.Database
		}
	}

	var backupConfig currentBackupConfig

	err := api.get(ctx, "/backup-configs/database/"+database.ID, &backupConfig)
	if err != nil {
		return nil, fmt.Errorf("read the backup config of database %q: %w", database.Name, err)
	}

	if backupConfig.StorageID == nil {
		return nil, fmt.Errorf("%w: database %q", errBackupConfigIsAbsent, database.Name)
	}

	var backupList currentBackupList

	err = api.get(ctx, "/backups?database_id="+database.ID, &backupList)
	if err != nil {
		return nil, fmt.Errorf("list backups of database %q: %w", database.Name, err)
	}

	listedItems := []upgradecheck.Item{
		{Kind: "database", Name: database.Name, Values: databaseValues},
		{Kind: "backup config", Name: database.Name, Values: map[string]string{
			"isBackupsEnabled": strconv.FormatBool(backupConfig.IsBackupsEnabled),
			"storageId":        *backupConfig.StorageID,
			"encryption":       backupConfig.Encryption,
		}},
	}

	for _, backup := range backupList.Backups {
		listedItems = append(listedItems, upgradecheck.Item{
			Kind:   "backup",
			Name:   backup.ID,
			Values: map[string]string{"status": backup.Status, "encryption": backup.Encryption},
		})
	}

	return listedItems, nil
}

// checkStoredCredentials connects to the source with the password the previous release encrypted
// and stored, because the restore below sends a password of its own.
func checkStoredCredentials(ctx context.Context, api *apiClient, databaseID string) error {
	err := api.post(ctx, "/databases/"+databaseID+"/test-connection", nil, nil)
	if err != nil {
		return fmt.Errorf("database %q cannot connect with the password stored before the upgrade: %w",
			databaseName, err)
	}

	return nil
}

// restoreBackup restores the backup of the previous release into a new database on the source
// server. The restore decrypts the backup itself, so it proves that the master key survived the
// upgrade; checkStoredCredentials covers the stored database password.
func restoreBackup(t *testing.T, api *apiClient, backupID string) error {
	t.Helper()

	restoredDatabaseName := restoredDatabase

	err := api.post(t.Context(), "/restores/"+backupID+"/restore", currentRestoreRequest{
		PostgresqlDatabase: currentPostgresqlConnection{
			Version:  sourceVersion,
			Host:     sourceNetworkAlias,
			Port:     sourcePort,
			Username: sourceUsername,
			Password: sourcePassword,
			Database: &restoredDatabaseName,
			CPUCount: 1,
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("start the restore: %w", err)
	}

	var finishedRestore currentRestore

	err = testwait.Poll(t, "the restore of the pre-upgrade backup to finish", func(ctx context.Context) error {
		var restores []currentRestore

		listErr := api.get(ctx, "/restores/"+backupID, &restores)
		if listErr != nil {
			return listErr
		}

		if len(restores) == 0 {
			return errors.New("no restore is listed yet")
		}

		finishedRestore = restores[0]
		if finishedRestore.Status == "IN_PROGRESS" {
			return errors.New("the restore is in progress")
		}

		return nil
	})
	if err != nil {
		return err
	}

	if finishedRestore.Status != "COMPLETED" {
		failMessage := ""
		if finishedRestore.FailMessage != nil {
			failMessage = *finishedRestore.FailMessage
		}

		return fmt.Errorf("%w: status %s, %s", errRestoreNotCompleted, finishedRestore.Status, failMessage)
	}

	return nil
}
