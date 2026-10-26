//go:build testhelpers

package healthcheck_attempt

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"databasus-backend/internal/modules/databases"
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

type MockHealthcheckedDatabases struct {
	mock.Mock
}

func (m *MockHealthcheckedDatabases) ProbeLiveness(_ context.Context, databaseID uuid.UUID) error {
	return m.Called(databaseID).Error(0)
}

func (m *MockHealthcheckedDatabases) SetHealthStatus(
	_ context.Context,
	databaseID uuid.UUID,
	healthStatus *databases.HealthStatus,
) error {
	return m.Called(databaseID, healthStatus).Error(0)
}

func (m *MockHealthcheckedDatabases) GetDatabaseByID(_ context.Context, id uuid.UUID) (databases.Database, error) {
	args := m.Called(id)

	database, ok := args.Get(0).(databases.Database)
	if !ok {
		return databases.Database{}, args.Error(1)
	}

	return database, args.Error(1)
}
