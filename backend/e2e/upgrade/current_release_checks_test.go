package upgrade

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"testing"
	"time"

	"databasus-backend/e2e/upgrade/upgradecheck"
	"databasus-backend/internal/lib/testwait"
)

// This file speaks the API of the current code, so it changes together with any operation the
// current code breaks.

var (
	errSignInIncomplete     = errors.New("the sign-in did not complete")
	errWorkspaceNotListed   = errors.New("the workspace is not listed")
	errRestoreNotCompleted  = errors.New("the restore did not complete")
	errDatabaseNotConnected = errors.New("the database does not connect")
)

type currentSignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type currentSignInResponse struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type currentWorkspacePage struct {
	Items []currentWorkspace `json:"items"`
}

type currentWorkspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type currentMemberPage struct {
	Items []currentMember `json:"items"`
}

type currentMember struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type currentAuditEventPage struct {
	Items      []currentAuditEvent `json:"items"`
	NextCursor *string             `json:"nextCursor"`
}

type currentAuditEvent struct {
	ID        string                      `json:"id"`
	Message   string                      `json:"message"`
	CreatedAt string                      `json:"createdAt"`
	Author    *currentAuditEventAuthor    `json:"author"`
	Workspace *currentAuditEventWorkspace `json:"workspace"`
}

type currentAuditEventAuthor struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type currentAuditEventWorkspace struct {
	ID string `json:"id"`
}

type currentRestoreStart struct {
	BackupID string               `json:"backupId"`
	Target   currentRestoreTarget `json:"target"`
}

type currentRestoreTarget struct {
	Type                string                `json:"type"`
	Host                string                `json:"host"`
	Port                int                   `json:"port"`
	Username            string                `json:"username"`
	Password            string                `json:"password"`
	Database            string                `json:"database"`
	SslMode             string                `json:"sslMode"`
	SSHTunnel           currentDisabledTunnel `json:"sshTunnel"`
	CPUCount            int                   `json:"cpuCount"`
	IsExcludeExtensions bool                  `json:"isExcludeExtensions"`
	IsRestoreOwnership  bool                  `json:"isRestoreOwnership"`
	IsRestorePrivileges bool                  `json:"isRestorePrivileges"`
}

type currentDisabledTunnel struct {
	IsEnabled bool `json:"isEnabled"`
}

type currentDatabaseTestResult struct {
	IsConnected bool    `json:"isConnected"`
	Error       *string `json:"error"`
}

type currentRestore struct {
	ID          string  `json:"id"`
	Status      string  `json:"status"`
	FailMessage *string `json:"failMessage"`
}

// readUpgradeData signs in with the password set before the upgrade and lists every item the way
// the current code returns it.
func readUpgradeData(ctx context.Context, api *apiClient) ([]upgradecheck.Item, error) {
	var signIn currentSignInResponse

	err := api.post(ctx, "/sessions", currentSignInRequest{Email: accountEmail, Password: accountPassword}, &signIn)
	if err != nil {
		return nil, fmt.Errorf("sign in with the password set before the upgrade: %w", err)
	}

	if signIn.Token == "" {
		return nil, fmt.Errorf("%w for %s", errSignInIncomplete, accountEmail)
	}

	api.token = signIn.Token
	listedItems := []upgradecheck.Item{{Kind: "account", Name: signIn.Email, Values: map[string]string{}}}

	var workspacePage currentWorkspacePage

	err = api.get(ctx, "/workspaces?limit=100", &workspacePage)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}

	workspaceID := ""

	for _, workspace := range workspacePage.Items {
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

	auditEventItems, err := listAuditEventItems(ctx, api)
	if err != nil {
		return nil, err
	}

	return slices.Concat(listedItems, workspaceItems, auditEventItems), nil
}

// listAuditEventItems follows the pages of the global audit log to the last, so every event is
// checked and not only the first page.
func listAuditEventItems(ctx context.Context, api *apiClient) ([]upgradecheck.Item, error) {
	auditEventItems := []upgradecheck.Item{}
	query := url.Values{"limit": {"100"}}

	for {
		var auditEventPage currentAuditEventPage

		err := api.get(ctx, "/audit-events?"+query.Encode(), &auditEventPage)
		if err != nil {
			return nil, fmt.Errorf("list the global audit log: %w", err)
		}

		for _, auditEvent := range auditEventPage.Items {
			auditEventItem, err := toAuditEventItem(auditEvent)
			if err != nil {
				return nil, err
			}

			auditEventItems = append(auditEventItems, auditEventItem)
		}

		if auditEventPage.NextCursor == nil {
			return auditEventItems, nil
		}

		query.Set("after", *auditEventPage.NextCursor)
	}
}

func toAuditEventItem(auditEvent currentAuditEvent) (upgradecheck.Item, error) {
	createdAt, err := normalizeAuditTime(auditEvent.CreatedAt)
	if err != nil {
		return upgradecheck.Item{}, err
	}

	authorEmail, authorName := "", ""
	if auditEvent.Author != nil {
		authorEmail, authorName = auditEvent.Author.Email, auditEvent.Author.Name
	}

	workspaceID := ""
	if auditEvent.Workspace != nil {
		workspaceID = auditEvent.Workspace.ID
	}

	return upgradecheck.Item{
		Kind: "audit event",
		Name: auditEvent.ID,
		Values: map[string]string{
			"message":     auditEvent.Message,
			"createdAt":   createdAt,
			"workspaceId": workspaceID,
			"authorEmail": authorEmail,
			"authorName":  authorName,
		},
	}, nil
}

// normalizeAuditTime writes a time in UTC, because the release before the upgrade stored a time
// without a zone and the current code stores one with it, so the two serialize the same instant
// differently.
func normalizeAuditTime(serializedTime string) (string, error) {
	parsedTime, err := time.Parse(time.RFC3339Nano, serializedTime)
	if err != nil {
		return "", fmt.Errorf("parse the audit event time %q: %w", serializedTime, err)
	}

	return parsedTime.UTC().Format(time.RFC3339Nano), nil
}

func readWorkspaceItems(ctx context.Context, api *apiClient, workspaceID string) ([]upgradecheck.Item, error) {
	var listedItems []upgradecheck.Item

	var memberPage currentMemberPage

	err := api.get(ctx, "/workspace-members?limit=100&workspaceId="+workspaceID, &memberPage)
	if err != nil {
		return nil, fmt.Errorf("list workspace members: %w", err)
	}

	for _, member := range memberPage.Items {
		listedItems = append(listedItems, upgradecheck.Item{
			Kind: "workspace member", Name: member.Email, Values: map[string]string{"role": member.Role},
		})
	}

	storageItems, err := listStorageItems(ctx, api, workspaceID)
	if err != nil {
		return nil, err
	}

	listedItems = append(listedItems, storageItems...)

	notifierItems, err := listNotifierItems(ctx, api, workspaceID)
	if err != nil {
		return nil, err
	}

	listedItems = append(listedItems, notifierItems...)

	databaseItems, err := listDatabaseItems(ctx, api, workspaceID, notifierItems)
	if err != nil {
		return nil, err
	}

	return append(listedItems, databaseItems...), nil
}

// checkStoredCredentials connects to every source with the password the previous release encrypted
// and stored, because the restore below sends a password of its own.
func checkStoredCredentials(ctx context.Context, api *apiClient, databaseIDs map[string]string) error {
	for _, storedDatabaseName := range slices.Sorted(maps.Keys(databaseIDs)) {
		var testResult currentDatabaseTestResult

		err := api.post(ctx, "/databases/"+databaseIDs[storedDatabaseName]+"/test", nil, &testResult)
		if err != nil {
			return fmt.Errorf("test the connection of database %q: %w", storedDatabaseName, err)
		}

		if !testResult.IsConnected {
			return fmt.Errorf("%w with the password stored before the upgrade: database %q, %s",
				errDatabaseNotConnected, storedDatabaseName, getOptionalValue(testResult.Error))
		}
	}

	return nil
}

// restoreBackup restores the backup of the previous release into a new database on the source
// server. The restore decrypts the backup itself, so it proves that the master key survived the
// upgrade; checkStoredCredentials covers the stored database password.
func restoreBackup(t *testing.T, api *apiClient, backupID string) error {
	t.Helper()

	var startedRestore currentRestore

	err := api.post(t.Context(), "/restores", currentRestoreStart{
		BackupID: backupID,
		Target: currentRestoreTarget{
			Type:                "POSTGRES_LOGICAL",
			Host:                sourceNetworkAlias,
			Port:                sourcePort,
			Username:            sourceUsername,
			Password:            sourcePassword,
			Database:            restoredDatabase,
			SslMode:             "disable",
			SSHTunnel:           currentDisabledTunnel{IsEnabled: false},
			CPUCount:            1,
			IsExcludeExtensions: false,
			IsRestoreOwnership:  false,
			IsRestorePrivileges: false,
		},
	}, &startedRestore)
	if err != nil {
		return fmt.Errorf("start the restore: %w", err)
	}

	var finishedRestore currentRestore

	err = testwait.Poll(t, "the restore of the pre-upgrade backup to finish", func(ctx context.Context) error {
		readErr := api.get(ctx, "/restores/"+startedRestore.ID, &finishedRestore)
		if readErr != nil {
			return readErr
		}

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
