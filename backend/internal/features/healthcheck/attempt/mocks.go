//go:build testhelpers

package healthcheck_attempt

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/modules/notifiers"
)

type MockHealthcheckAttemptSender struct {
	mock.Mock
}

func (m *MockHealthcheckAttemptSender) Notify(
	_ context.Context,
	notifierIDs []uuid.UUID,
	notification notifiers.Notification,
) {
	m.Called(notifierIDs, notification)
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
