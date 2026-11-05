package config

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

func (s *Service) withCleanup(
	ctx context.Context,
	actor users.Actor,
	configs []AnyBackupConfig,
) ([]AnyBackupConfig, error) {
	databaseIDs := make([]uuid.UUID, 0, len(configs))
	for _, backupConfig := range configs {
		databaseIDs = append(databaseIDs, backupConfig.GetBackupConfig().DatabaseID)
	}

	configDatabases, err := s.databases.GetDatabases(ctx, databaseIDs)
	if err != nil {
		return nil, err
	}

	scopes := make([]storages.CleanupScope, 0, len(configDatabases))
	databaseScopes := make(map[uuid.UUID]storages.CleanupScope, len(configDatabases))
	authorizedWorkspaces := make(map[uuid.UUID]bool)
	for _, configDatabase := range configDatabases {
		if !authorizedWorkspaces[configDatabase.WorkspaceID] {
			if err := s.authorizeDatabase(ctx, actor, configDatabase, workspaces.ActionView); err != nil {
				return nil, err
			}

			authorizedWorkspaces[configDatabase.WorkspaceID] = true
		}

		scope := storages.CleanupScope{DatabaseID: configDatabase.ID, WorkspaceID: configDatabase.WorkspaceID}
		scopes = append(scopes, scope)
		databaseScopes[configDatabase.ID] = scope
	}

	summaries, err := s.storages.GetCleanupSummaries(ctx, scopes)
	if err != nil {
		return nil, err
	}

	for index, backupConfig := range configs {
		summary := summaries[databaseScopes[backupConfig.GetBackupConfig().DatabaseID]]
		switch typedConfig := backupConfig.(type) {
		case LogicalBackupConfig:
			typedConfig.Cleanup = summary
			configs[index] = typedConfig
		case PhysicalBackupConfig:
			typedConfig.Cleanup = summary
			configs[index] = typedConfig
		}
	}

	return configs, nil
}
