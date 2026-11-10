package main

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"

	"github.com/google/uuid"

	"databasus-backend/internal/app"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/modules/users"
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
	BaseURL     string
	EnvMode     config.EnvMode
	DatabaseDsn string

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
	if params.EnvMode != config.EnvModeDevelopment {
		return app.ErrDevSeedNotDevelopment
	}

	client := &apiClient{httpClient: httpClient, baseURL: params.BaseURL}

	if err := checkBackendHealth(ctx, client, logger); err != nil {
		return err
	}

	session, err := signInOrSignUp(ctx, client, params, logger)
	if err != nil {
		return err
	}

	client.token = session.Token

	if session.UserID == uuid.Nil || session.Token == "" {
		return fmt.Errorf("the dev account authentication response has no identity or token")
	}

	isChanged, err := app.EnsureDevSeedBootstrapAdmin(ctx, app.DevSeedAdminRequest{
		EnvMode:     params.EnvMode,
		DatabaseDsn: params.DatabaseDsn,
		Account:     users.DevSeedAccount{UserID: session.UserID, Email: params.Email},
		Clock:       clock.NewSystem(),
		AuditLogger: logger.With(slog.String("log_type", "audit")),
	})
	if err != nil {
		return fmt.Errorf("assign the dev account as bootstrap administrator: %w", err)
	}

	if isChanged {
		logger.InfoContext(
			ctx,
			"assigned the dev account as bootstrap administrator",
			slog.String("user_id", session.UserID.String()),
		)
	}

	workspaceID, err := findOrCreateWorkspace(ctx, client, params.WorkspaceName, logger)
	if err != nil {
		return err
	}

	storageID, err := findOrCreateLocalStorage(ctx, client, workspaceID, params.StorageName, logger)
	if err != nil {
		return err
	}

	existingDatabases, err := listAll(ctx, client, "/databases?limit=100&workspaceId="+workspaceID.String())
	if err != nil {
		return fmt.Errorf("list databases of workspace %q: %w", params.WorkspaceName, err)
	}

	if _, isFound := findNamedItem(existingDatabases, params.LogicalDatabaseName); !isFound {
		if err := createLogicalDatabase(ctx, client, workspaceID, storageID, params); err != nil {
			return err
		}

		logger.InfoContext(
			ctx,
			"created database with logical backups",
			slog.String("name", params.LogicalDatabaseName),
		)
	}

	if _, isFound := findNamedItem(existingDatabases, params.WalDatabaseName); !isFound {
		if err := createWalDatabase(ctx, client, workspaceID, storageID, params); err != nil {
			return err
		}

		logger.InfoContext(
			ctx,
			"created database with WAL-streaming backups",
			slog.String("name", params.WalDatabaseName),
		)
	}

	logger.InfoContext(ctx, "dev app is seeded", slog.String("sign_in_email", params.Email))

	return nil
}

func signInOrSignUp(
	ctx context.Context,
	client *apiClient,
	params seedParams,
	logger *slog.Logger,
) (sessionResponse, error) {
	var signIn signInResponse

	signInErr := client.post(ctx, "/sessions",
		signInRequest{Email: params.Email, Password: params.Password}, &signIn)
	if signInErr == nil {
		if signIn.Outcome != "COMPLETED" {
			return sessionResponse{}, fmt.Errorf(
				"cannot sign in as %s: the instance requires two-factor sign-in", params.Email,
			)
		}

		return sessionResponse{UserID: signIn.UserID, Token: signIn.Token}, nil
	}

	if !isAPIRefusal(signInErr) {
		return sessionResponse{}, fmt.Errorf("sign in as %s: %w", params.Email, signInErr)
	}

	var signUp sessionResponse

	signUpErr := client.post(ctx, "/users", signUpRequest{
		Email:    params.Email,
		Password: params.Password,
		Name:     "Dev",
	}, &signUp)
	if signUpErr != nil {
		return sessionResponse{}, fmt.Errorf("cannot sign in as %s (%w) and cannot register it (%w)",
			params.Email, signInErr, signUpErr)
	}

	logger.InfoContext(ctx, "registered the dev account")

	return signUp, nil
}

func findOrCreateWorkspace(
	ctx context.Context,
	client *apiClient,
	workspaceName string,
	logger *slog.Logger,
) (uuid.UUID, error) {
	existingWorkspaces, err := listAll(ctx, client, "/workspaces?limit=100")
	if err != nil {
		return uuid.Nil, fmt.Errorf("list workspaces: %w", err)
	}

	if workspace, isFound := findNamedItem(existingWorkspaces, workspaceName); isFound {
		return workspace.ID, nil
	}

	var createdWorkspace namedItem
	if err := client.post(ctx, "/workspaces",
		createWorkspaceRequest{Name: workspaceName}, &createdWorkspace); err != nil {
		return uuid.Nil, fmt.Errorf("cannot create workspace %q: %w", workspaceName, err)
	}

	logger.InfoContext(ctx, "created workspace", slog.String("name", workspaceName))

	return createdWorkspace.ID, nil
}

func findOrCreateLocalStorage(
	ctx context.Context,
	client *apiClient,
	workspaceID uuid.UUID,
	storageName string,
	logger *slog.Logger,
) (uuid.UUID, error) {
	existingStorages, err := listAll(ctx, client, "/storages?limit=100&workspaceId="+workspaceID.String())
	if err != nil {
		return uuid.Nil, fmt.Errorf("list storages: %w", err)
	}

	if storage, isFound := findNamedItem(existingStorages, storageName); isFound {
		return storage.ID, nil
	}

	var createdStorage namedItem
	if err := client.post(ctx, "/storages", createStorageRequest{
		WorkspaceID: workspaceID,
		Name:        storageName,
		Settings:    localStorageSettings{Type: "LOCAL"},
	}, &createdStorage); err != nil {
		return uuid.Nil, fmt.Errorf("cannot create storage %q: %w", storageName, err)
	}

	logger.InfoContext(ctx, "created storage", slog.String("name", storageName))

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
		NotifierIDs: []uuid.UUID{},
		Settings: postgresqlLogicalSettings{
			Type:               "POSTGRES_LOGICAL",
			Host:               source.Host,
			Port:               source.Port,
			Username:           source.Username,
			Password:           source.Password,
			Database:           source.Database,
			SslMode:            "disable",
			SSHTunnel:          disabledSSHTunnel{IsEnabled: false},
			IncludeSchemas:     []string{},
			ExcludeTables:      []string{},
			ExcludeTableData:   []string{},
			CPUCount:           1,
			IsSkipUserMappings: false,
		},
	})
	if err != nil {
		return err
	}

	err = switchOnBackups(ctx, client, params.LogicalDatabaseName, databaseID, storageID,
		map[string]any{
			"policy": "COUNT",
			"count":  logicalBackupsRetentionCount,
		})
	if err != nil {
		return err
	}

	if err := client.post(ctx, "/logical-backups", startLogicalBackupRequest{DatabaseID: databaseID}, nil); err != nil {
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
		NotifierIDs: []uuid.UUID{},
		Settings: postgresqlPhysicalSettings{
			Type:       "POSTGRES_PHYSICAL",
			BackupType: "FULL_INCREMENTAL_WAL_STREAM",
			Host:       source.Host,
			Port:       source.Port,
			Username:   source.Username,
			Password:   source.Password,
			SslMode:    "disable",
			SSHTunnel:  disabledSSHTunnel{IsEnabled: false},
		},
	})
	if err != nil {
		return err
	}

	err = switchOnBackups(ctx, client, params.WalDatabaseName, databaseID, storageID,
		map[string]any{
			"policy":      "CHAINS",
			"chainsCount": walChainsRetentionCount,
			"fullBackups": nil,
		})
	if err != nil {
		return err
	}

	if err := client.post(
		ctx,
		"/physical-backups",
		map[string]any{"databaseId": databaseID, "kind": "AUTO"},
		nil,
	); err != nil {
		return fmt.Errorf("cannot start the first backup of database %q: %w", params.WalDatabaseName, err)
	}

	return nil
}

func createDatabase(ctx context.Context, client *apiClient, database createDatabaseRequest) (uuid.UUID, error) {
	var createdDatabase namedItem
	if err := client.post(ctx, "/databases", database, &createdDatabase); err != nil {
		return uuid.Nil, fmt.Errorf("cannot create database %q: %w", database.Name, err)
	}

	return createdDatabase.ID, nil
}

// A database's backup config reads as its defaults until the first save. The seed changes only the
// storage and the retention fields it owns and sends the rest back as read, so it never restates
// defaults such as the schedule or the WAL lag threshold.
func switchOnBackups(
	ctx context.Context,
	client *apiClient,
	databaseName string,
	databaseID, storageID uuid.UUID,
	seededRetentionFields map[string]any,
) error {
	configPath := "/backup-configs/" + databaseID.String()

	var backupConfig struct {
		Encryption string         `json:"encryption"`
		Settings   map[string]any `json:"settings"`
	}
	if err := client.get(ctx, configPath, &backupConfig); err != nil {
		return fmt.Errorf("read backup settings of database %q: %w", databaseName, err)
	}

	retention, _ := backupConfig.Settings["retention"].(map[string]any)
	if retention == nil {
		retention = map[string]any{}
	}

	maps.Copy(retention, seededRetentionFields)
	backupConfig.Settings["retention"] = retention

	backupConfigInput := map[string]any{
		"isBackupsEnabled": true,
		"storageId":        storageID,
		"encryption":       backupConfig.Encryption,
		"settings":         backupConfig.Settings,
	}

	if err := client.put(ctx, configPath, backupConfigInput, nil); err != nil {
		return fmt.Errorf("cannot switch on backups of database %q: %w", databaseName, err)
	}

	return nil
}
