package health

import (
	"context"
	"log/slog"
	"net/mail"
	"time"

	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/modules/instancebackups/internal/configs"
	"databasus-backend/internal/modules/instancebackups/internal/keyexport"
	"databasus-backend/internal/modules/instancebackups/internal/recovery"
	"databasus-backend/internal/modules/users"
)

// overdueGrace is how long a scheduled occurrence may stay uncovered before it is overdue.
const overdueGrace = 15 * time.Minute

type Deps struct {
	Repositories  Repositories
	Transactor    *database.Transactor
	Clock         clock.Source
	Logger        *slog.Logger
	Mailer        mailer.Sender
	MasterKeyPath string
	Users         users.Module
	Configs       *configs.Service
	KeyExport     *keyexport.Service
	Recoveries    *recovery.Service
}

type Service struct {
	incidents     *incidentRepository
	transactor    *database.Transactor
	clock         clock.Source
	logger        *slog.Logger
	mailer        mailer.Sender
	masterKeyPath string
	users         users.Module
	configs       *configs.Service
	keyExport     *keyexport.Service
	recoveries    *recovery.Service
}

func NewService(deps Deps) *Service {
	return &Service{
		incidents:     deps.Repositories.incidents,
		transactor:    deps.Transactor,
		clock:         deps.Clock,
		logger:        deps.Logger,
		mailer:        deps.Mailer,
		masterKeyPath: deps.MasterKeyPath,
		users:         deps.Users,
		configs:       deps.Configs,
		keyExport:     deps.KeyExport,
		recoveries:    deps.Recoveries,
	}
}

// assessment is the protection at one moment: the settings, every destination and their sum.
type assessment struct {
	Config       configs.Config
	Health       Health
	Destinations []DestinationHealth
}

func (s *Service) assess(ctx context.Context, now time.Time) (assessment, error) {
	config, err := s.configs.GetConfig(ctx)
	if err != nil {
		return assessment{}, err
	}

	destinations, err := s.describeDestinations(ctx, config, now)
	if err != nil {
		return assessment{}, err
	}

	return assessment{
		Config:       config,
		Health:       combineHealth(config.ProtectionState, destinations),
		Destinations: destinations,
	}, nil
}

func (s *Service) describeDestinations(
	ctx context.Context,
	config configs.Config,
	now time.Time,
) ([]DestinationHealth, error) {
	summaries, err := s.configs.GetDestinationSummaries(ctx)
	if err != nil {
		return nil, err
	}

	destinations, err := s.configs.GetDestinations(ctx)
	if err != nil {
		return nil, err
	}

	described := make([]DestinationHealth, 0, len(summaries))

	for _, summary := range summaries {
		for _, destination := range destinations {
			if destination.StorageID != summary.StorageID {
				continue
			}

			overdueAt := getOverdueAt(destination.GetOutstandingDueAt(config.GetSchedule(), now))

			described = append(described, DestinationHealth{
				StorageID:                summary.StorageID,
				StorageName:              summary.StorageName,
				StorageType:              summary.StorageType,
				WorkspaceID:              summary.WorkspaceID,
				WorkspaceName:            summary.WorkspaceName,
				Health:                   assessDestination(destination, overdueAt, now),
				LastSuccessfulSnapshotAt: destination.LastSuccessfulSnapshotAt,
				OverdueAt:                overdueAt,
				LastError:                destination.LastError,
			})
		}
	}

	return described, nil
}

// isMailEligible tells whether incident mail can reach root admin now.
func (s *Service) isMailEligible(rootAdmin users.User) bool {
	return s.mailer.IsConfigured() && isEmailAddress(rootAdmin.Email)
}

func assessDestination(destination configs.Destination, overdueAt *time.Time, now time.Time) Health {
	switch {
	case overdueAt != nil && !now.Before(*overdueAt):
		return HealthOverdue
	case destination.LastError != nil:
		return HealthFailed
	case destination.LastSuccessfulSnapshotAt == nil:
		return HealthInitializing
	default:
		return HealthHealthy
	}
}

func getOverdueAt(outstandingDueAt *time.Time) *time.Time {
	if outstandingDueAt == nil {
		return nil
	}

	overdueAt := outstandingDueAt.Add(overdueGrace)

	return &overdueAt
}

// combineHealth reports the lifecycle while no copy has succeeded yet, and the worst destination
// afterwards, where some healthy destinations among failing ones make protection partial.
func combineHealth(state configs.ProtectionState, destinations []DestinationHealth) Health {
	switch state {
	case configs.ProtectionStateNotConfigured:
		return HealthNotConfigured
	case configs.ProtectionStateStopped:
		return HealthStopped
	case configs.ProtectionStateInitializing, configs.ProtectionStateResuming, configs.ProtectionStateProtected:
	}

	healthyCount, overdueCount, failedCount := 0, 0, 0

	for _, destination := range destinations {
		switch destination.Health {
		case HealthHealthy:
			healthyCount++
		case HealthOverdue:
			overdueCount++
		case HealthFailed:
			failedCount++
		case HealthNotConfigured, HealthInitializing, HealthPartial, HealthStopped, HealthResuming:
		}
	}

	switch {
	case state == configs.ProtectionStateInitializing && healthyCount == 0 && overdueCount == 0 && failedCount == 0:
		return HealthInitializing
	case state == configs.ProtectionStateResuming && healthyCount == 0 && overdueCount == 0 && failedCount == 0:
		return HealthResuming
	case healthyCount == len(destinations):
		return HealthHealthy
	case healthyCount > 0:
		return HealthPartial
	case overdueCount > 0 && failedCount == 0:
		return HealthOverdue
	default:
		return HealthFailed
	}
}

func isEmailAddress(value string) bool {
	parsedAddress, err := mail.ParseAddress(value)

	return err == nil && parsedAddress.Address == value
}
