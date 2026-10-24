package main

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"

	"github.com/google/uuid"
)

const (
	logicalBackupsRetentionCount = 3
	walChainsRetentionCount      = 2
)

type postgresSource struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

type seedParams struct {
	BaseURL string

	Email    string
	Password string

	WorkspaceName string
	StorageName   string

	LogicalDatabaseName string
	LogicalSource       postgresSource

	WalDatabaseName string
	WalSource       postgresSource
}

// Sign-in and sign-up stay one attempt each: sign-in is rate-limited per address, and the seed
// must never take the password reset path that could change an existing account.
func seedDevApp(ctx context.Context, httpClient *http.Client, params seedParams, logger *slog.Logger) error {
	client := &apiClient{httpClient: httpClient, baseURL: params.BaseURL}

	if err := checkBackendHealth(ctx, client, params.BaseURL); err != nil {
		return err
	}

	token, err := signInOrSignUp(ctx, client, params, logger)
	if err != nil {
		return err
	}

	client.token = token

	workspaceID, err := findOrCreateWorkspace(ctx, client, params.WorkspaceName, logger)
	if err != nil {
		return err
	}

	storageID, err := findOrCreateLocalStorage(ctx, client, workspaceID, params.StorageName, logger)
	if err != nil {
		return err
	}

	var existingDatabases []namedItem
	if err := client.get(ctx, "/databases?workspace_id="+workspaceID.String(), &existingDatabases); err != nil {
		return fmt.Errorf("list databases of workspace %q: %w", params.WorkspaceName, err)
	}

	if !hasDatabaseNamed(existingDatabases, params.LogicalDatabaseName) {
		if err := createLogicalDatabase(ctx, client, workspaceID, storageID, params); err != nil {
			return err
		}

		logger.Info("created database with logical backups", "name", params.LogicalDatabaseName)
	}

	if !hasDatabaseNamed(existingDatabases, params.WalDatabaseName) {
		if err := createWalDatabase(ctx, client, workspaceID, storageID, params); err != nil {
			return err
		}

		logger.Info("created database with WAL-streaming backups", "name", params.WalDatabaseName)
	}

	logger.Info(fmt.Sprintf("dev app is seeded: sign in as %s", params.Email))

	return nil
}

func checkBackendHealth(ctx context.Context, client *apiClient, baseURL string) error {
	err := client.get(ctx, "/system/health", nil)
	if err == nil {
		return nil
	}

	if isAPIRefusal(err) {
		return fmt.Errorf("backend at %s reports itself unhealthy: %w", baseURL, err)
	}

	return fmt.Errorf("backend at %s does not answer, start the app with `mise run dev`: %w", baseURL, err)
}

func signInOrSignUp(
	ctx context.Context,
	client *apiClient,
	params seedParams,
	logger *slog.Logger,
) (string, error) {
	var signIn signInResponse

	signInErr := client.post(ctx, "/users/signin",
		signInRequest{Email: params.Email, Password: params.Password}, &signIn)
	if signInErr == nil {
		if signIn.Token == "" {
			return "", fmt.Errorf(
				"cannot sign in as %s: the instance requires two-factor sign-in", params.Email)
		}

		return signIn.Token, nil
	}

	if !isAPIRefusal(signInErr) {
		return "", fmt.Errorf("sign in as %s: %w", params.Email, signInErr)
	}

	var signUp signInResponse

	signUpErr := client.post(ctx, "/users/signup", signUpRequest{
		Email:    params.Email,
		Password: params.Password,
		Name:     "Dev",
	}, &signUp)
	if signUpErr != nil {
		return "", fmt.Errorf("cannot sign in as %s (%w) and cannot register it (%w)",
			params.Email, signInErr, signUpErr)
	}

	logger.Info("registered the dev account")

	return signUp.Token, nil
}

func findOrCreateWorkspace(
	ctx context.Context,
	client *apiClient,
	workspaceName string,
	logger *slog.Logger,
) (uuid.UUID, error) {
	var listedWorkspaces listWorkspacesResponse
	if err := client.get(ctx, "/workspaces", &listedWorkspaces); err != nil {
		return uuid.Nil, fmt.Errorf("list workspaces: %w", err)
	}

	for _, workspace := range listedWorkspaces.Workspaces {
		if workspace.Name == workspaceName {
			return workspace.ID, nil
		}
	}

	var createdWorkspace namedItem
	if err := client.post(ctx, "/workspaces",
		createWorkspaceRequest{Name: workspaceName}, &createdWorkspace); err != nil {
		return uuid.Nil, fmt.Errorf("cannot create workspace %q: %w", workspaceName, err)
	}

	logger.Info("created workspace", "name", workspaceName)

	return createdWorkspace.ID, nil
}

func findOrCreateLocalStorage(
	ctx context.Context,
	client *apiClient,
	workspaceID uuid.UUID,
	storageName string,
	logger *slog.Logger,
) (uuid.UUID, error) {
	var listedStorages []namedItem
	if err := client.get(ctx, "/storages?workspace_id="+workspaceID.String(), &listedStorages); err != nil {
		return uuid.Nil, fmt.Errorf("list storages: %w", err)
	}

	for _, storage := range listedStorages {
		if storage.Name == storageName {
			return storage.ID, nil
		}
	}

	var createdStorage namedItem
	if err := client.post(ctx, "/storages", createLocalStorageRequest{
		WorkspaceID: workspaceID,
		Type:        "LOCAL",
		Name:        storageName,
	}, &createdStorage); err != nil {
		return uuid.Nil, fmt.Errorf("cannot create storage %q: %w", storageName, err)
	}

	logger.Info("created storage", "name", storageName)

	return createdStorage.ID, nil
}

func createLogicalDatabase(
	ctx context.Context,
	client *apiClient,
	workspaceID, storageID uuid.UUID,
	params seedParams,
) error {
	source := params.LogicalSource

	databaseID, err := createDatabase(ctx, client, createDatabaseRequest{
		WorkspaceID: workspaceID,
		Name:        params.LogicalDatabaseName,
		Type:        "POSTGRES_LOGICAL",
		PostgresqlLogical: &postgresqlLogicalSource{
			Version:  "16",
			Host:     source.Host,
			Port:     source.Port,
			Username: source.Username,
			Password: source.Password,
			Database: source.Database,
			CpuCount: 1,
		},
	})
	if err != nil {
		return err
	}

	err = switchOnBackups(ctx, client, params.LogicalDatabaseName,
		"/backup-configs/database/"+databaseID.String(), "/backup-configs/save",
		map[string]any{
			"storage":             map[string]any{"id": storageID},
			"storageId":           storageID,
			"retentionPolicyType": "COUNT",
			"retentionCount":      logicalBackupsRetentionCount,
		})
	if err != nil {
		return err
	}

	if err := client.post(ctx, "/backups", makeLogicalBackupRequest{DatabaseID: databaseID}, nil); err != nil {
		return fmt.Errorf("cannot start the first backup of database %q: %w", params.LogicalDatabaseName, err)
	}

	return nil
}

func createWalDatabase(
	ctx context.Context,
	client *apiClient,
	workspaceID, storageID uuid.UUID,
	params seedParams,
) error {
	source := params.WalSource

	databaseID, err := createDatabase(ctx, client, createDatabaseRequest{
		WorkspaceID: workspaceID,
		Name:        params.WalDatabaseName,
		Type:        "POSTGRES_PHYSICAL",
		PostgresqlPhysical: &postgresqlPhysicalSource{
			Version:    "18",
			BackupType: "FULL_INCREMENTAL_WAL_STREAM",
			Host:       source.Host,
			Port:       source.Port,
			Username:   source.Username,
			Password:   source.Password,
		},
	})
	if err != nil {
		return err
	}

	err = switchOnBackups(ctx, client, params.WalDatabaseName,
		"/backup-configs/physical/database/"+databaseID.String(), "/backup-configs/physical/save",
		map[string]any{
			"storage":              map[string]any{"id": storageID},
			"storageId":            storageID,
			"retention":            "CHAINS",
			"chainsRetention":      map[string]any{"count": walChainsRetentionCount},
			"fullBackupsRetention": map[string]any{},
		})
	if err != nil {
		return err
	}

	if err := client.post(ctx, "/backups/physical/database/"+databaseID.String()+"/trigger",
		triggerPhysicalBackupRequest{Type: "auto"}, nil); err != nil {
		return fmt.Errorf("cannot start the first backup of database %q: %w", params.WalDatabaseName, err)
	}

	return nil
}

func createDatabase(ctx context.Context, client *apiClient, database createDatabaseRequest) (uuid.UUID, error) {
	var createdDatabase namedItem
	if err := client.post(ctx, "/databases/create", database, &createdDatabase); err != nil {
		return uuid.Nil, fmt.Errorf("cannot create database %q: %w", database.Name, err)
	}

	return createdDatabase.ID, nil
}

// The backend fills a new database's backup settings with its defaults on the first read. The seed
// changes only the fields it owns and sends the rest back as read, so it never restates defaults
// such as the schedule or the WAL lag threshold.
func switchOnBackups(
	ctx context.Context,
	client *apiClient,
	databaseName, readPath, savePath string,
	seededFields map[string]any,
) error {
	var backupConfig map[string]any
	if err := client.get(ctx, readPath, &backupConfig); err != nil {
		return fmt.Errorf("read backup settings of database %q: %w", databaseName, err)
	}

	backupConfig["isBackupsEnabled"] = true
	maps.Copy(backupConfig, seededFields)

	if err := client.post(ctx, savePath, backupConfig, nil); err != nil {
		return fmt.Errorf("cannot switch on backups of database %q: %w", databaseName, err)
	}

	return nil
}

func hasDatabaseNamed(existingDatabases []namedItem, name string) bool {
	return slices.ContainsFunc(existingDatabases, func(database namedItem) bool {
		return database.Name == name
	})
}
