//go:build testhelpers

package healthcheck_attempt

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	notifier_models "databasus-backend/internal/features/notifiers/models"
)

type MockHealthcheckAttemptSender struct {
	mock.Mock
}

func (m *MockHealthcheckAttemptSender) SendNotification(
	_ context.Context,
	notifier *notifiers.Notifier,
	notification notifier_models.Notification,
) {
	m.Called(notifier, notification)
}

type MockDatabaseService struct {
	mock.Mock
}

func (m *MockDatabaseService) ProbeDatabaseLiveness(
	ctx context.Context,
	database *databases.Database,
) error {
	return m.Called(database).Error(0)
}

func (m *MockDatabaseService) SetHealthStatus(
	databaseID uuid.UUID,
	healthStatus *databases.HealthStatus,
) error {
	return m.Called(databaseID, healthStatus).Error(0)
}

func (m *MockDatabaseService) GetDatabaseByID(
	id uuid.UUID,
) (*databases.Database, error) {
	args := m.Called(id)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	database, ok := args.Get(0).(*databases.Database)
	if !ok {
		return nil, args.Error(1)
	}

	return database, args.Error(1)
}
