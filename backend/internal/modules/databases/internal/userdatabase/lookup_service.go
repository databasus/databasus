package userdatabase

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/databases/internal/providers"
)

// GetDatabases leaves out the IDs that match no database.
func (s *Service) GetDatabases(ctx context.Context, databaseIDs []uuid.UUID) ([]Database, error) {
	if len(databaseIDs) == 0 {
		return []Database{}, nil
	}

	foundDatabases, err := s.repository.listByIDs(ctx, databaseIDs)
	if err != nil {
		return nil, err
	}

	return s.loadNotifierIDs(ctx, foundDatabases)
}

func (s *Service) ListAllDatabases(ctx context.Context) ([]Database, error) {
	allDatabases, err := s.repository.listAll(ctx)
	if err != nil {
		return nil, err
	}

	return s.loadNotifierIDs(ctx, allDatabases)
}

// GetTelemetry skips a database without its settings row, which the statistics cannot describe.
func (s *Service) GetTelemetry(ctx context.Context) ([]TelemetryEntry, error) {
	allDatabases, err := s.repository.listAll(ctx)
	if err != nil {
		return nil, err
	}

	settingsByDatabaseID, err := s.loadSettingsByDatabaseID(ctx, allDatabases)
	if err != nil {
		return nil, err
	}

	telemetryEntries := make([]TelemetryEntry, 0, len(allDatabases))

	for _, describedDatabase := range allDatabases {
		settings, isFound := settingsByDatabaseID[describedDatabase.ID]
		if !isFound {
			continue
		}

		provider, err := s.registry.GetProvider(describedDatabase.Type)
		if err != nil {
			return nil, err
		}

		telemetryFacts, err := provider.GetTelemetryFacts(settings)
		if err != nil {
			return nil, err
		}

		telemetryEntries = append(telemetryEntries, TelemetryEntry{
			DatabaseID:     describedDatabase.ID,
			Type:           describedDatabase.Type,
			HealthStatus:   describedDatabase.HealthStatus,
			LastBackupTime: describedDatabase.LastBackupTime,
			TelemetryFacts: telemetryFacts,
		})
	}

	return telemetryEntries, nil
}

// GetWithSettings answers ErrDatabaseNotFound for a database that does not exist. It checks no
// access: only the trusted paths of the module call it.
func (s *Service) GetWithSettings(ctx context.Context, databaseID uuid.UUID) (WithSettings, error) {
	foundDatabase, isFound, err := s.repository.get(ctx, databaseID, noRowLock)
	if err != nil {
		return WithSettings{}, err
	}

	if !isFound {
		return WithSettings{}, ErrDatabaseNotFound
	}

	return s.loadOneWithSettings(ctx, foundDatabase)
}

// GetConnectionSettings decrypts the settings of one database for the code that connects on its own.
func (s *Service) GetConnectionSettings(
	ctx context.Context,
	databaseID uuid.UUID,
) (providers.ConnectionSettings, error) {
	foundDatabase, err := s.GetWithSettings(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	provider, err := s.registry.GetProvider(foundDatabase.Type)
	if err != nil {
		return nil, err
	}

	return provider.GetConnectionSettings(foundDatabase.ID, foundDatabase.Settings)
}
