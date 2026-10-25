package healthcheck_attempt

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"databasus-backend/internal/features/databases"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	"databasus-backend/internal/features/notifiers"
	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/logger"
)

func unavailableNotification(databaseName string) notifier_models.Notification {
	return notifier_models.Notification{
		Type:    notifier_models.NotificationTypeHealthcheckFailed,
		Heading: fmt.Sprintf("❌ [%s] DB is unavailable", databaseName),
		Message: fmt.Sprintf("❌ [%s] DB is currently unavailable", databaseName),
	}
}

func onlineNotification(databaseName string) notifier_models.Notification {
	return notifier_models.Notification{
		Type:    notifier_models.NotificationTypeHealthcheckSuccess,
		Heading: fmt.Sprintf("✅ [%s] DB is online", databaseName),
		Message: fmt.Sprintf("✅ [%s] DB is back online", databaseName),
	}
}

const anyPhysicalVersionTag = "17"

func newTestUseCase(
	healthcheckAttemptSender HealthcheckAttemptSender,
	databaseService DatabaseService,
) *CheckDatabaseHealthUseCase {
	return &CheckDatabaseHealthUseCase{
		&HealthcheckAttemptRepository{},
		healthcheckAttemptSender,
		databaseService,
	}
}

func newTestHealthcheckConfig(databaseID uuid.UUID) *healthcheck_config.HealthcheckConfig {
	return &healthcheck_config.HealthcheckConfig{
		DatabaseID:                        databaseID,
		IsHealthcheckEnabled:              true,
		IsSentNotificationWhenUnavailable: true,
		IntervalMinutes:                   1,
		AttemptsBeforeConcideredAsDown:    1,
		StoreAttemptsDays:                 7,
	}
}

func Test_CheckDatabaseHealthUseCase(t *testing.T) {
	user := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)

	// Create workspace directly via service
	workspace, err := workspaces_testing.CreateTestWorkspaceDirect("Test Workspace", user.UserID)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)

	defer func() {
		storages.RemoveTestStorage(t.Context(), storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspaceDirect(workspace.ID)
	}()

	t.Run("Test_DbAttemptFailed_DbMarkedAsUnavailable", func(t *testing.T) {
		database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
		defer databases.RemoveTestDatabase(t.Context(), database)

		// Setup mock notifier sender
		mockSender := &MockHealthcheckAttemptSender{}
		mockSender.On("SendNotification", mock.Anything, mock.Anything).Return()

		// Setup mock database service
		mockDatabaseService := &MockDatabaseService{}
		mockDatabaseService.On("ProbeDatabaseLiveness", database).
			Return(errors.New("test error"))
		unavailableStatus := databases.HealthStatusUnavailable
		mockDatabaseService.On("SetHealthStatus", database.ID, &unavailableStatus).
			Return(nil)
		mockDatabaseService.On("GetDatabaseByID", database.ID).
			Return(database, nil)

		// Setup healthcheck config
		healthcheckConfig := &healthcheck_config.HealthcheckConfig{
			DatabaseID:                        database.ID,
			IsHealthcheckEnabled:              true,
			IsSentNotificationWhenUnavailable: true,
			IntervalMinutes:                   1,
			AttemptsBeforeConcideredAsDown:    1,
			StoreAttemptsDays:                 7,
		}

		// Create use case with mock sender
		useCase := newTestUseCase(mockSender, mockDatabaseService)

		// Execute healthcheck
		err := useCase.Execute(t.Context(), logger.GetLogger(), time.Now().UTC(), healthcheckConfig)
		assert.NoError(t, err)

		// Verify attempt was created and marked as unavailable
		attempts, err := useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
			database.ID,
			1,
		)
		assert.NoError(t, err)
		assert.Len(t, attempts, 1)
		assert.Equal(t, databases.HealthStatusUnavailable, attempts[0].Status)

		// Verify database health status was updated (via mock)
		mockDatabaseService.AssertCalled(
			t,
			"SetHealthStatus",
			database.ID,
			&unavailableStatus,
		)

		// Verify notification was sent
		mockSender.AssertCalled(
			t,
			"SendNotification",
			mock.Anything,
			unavailableNotification(database.Name),
		)
	})

	t.Run(
		"Test_DbShouldBeConsideredAsDownOnThirdFailedAttempt_DbNotMarkerdAsDownAfterFirstAttempt",
		func(t *testing.T) {
			database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
			defer databases.RemoveTestDatabase(t.Context(), database)

			// Setup mock notifier sender
			mockSender := &MockHealthcheckAttemptSender{}

			// Setup mock database service - connection fails but SetHealthStatus should not be called
			mockDatabaseService := &MockDatabaseService{}
			mockDatabaseService.On("ProbeDatabaseLiveness", database).
				Return(errors.New("test error"))
			mockDatabaseService.On("GetDatabaseByID", database.ID).
				Return(database, nil)

			// Setup healthcheck config requiring 3 attempts before marking as down
			healthcheckConfig := &healthcheck_config.HealthcheckConfig{
				DatabaseID:                        database.ID,
				IsHealthcheckEnabled:              true,
				IsSentNotificationWhenUnavailable: true,
				IntervalMinutes:                   1,
				AttemptsBeforeConcideredAsDown:    3,
				StoreAttemptsDays:                 7,
			}

			// Create use case with mock sender
			useCase := newTestUseCase(mockSender, mockDatabaseService)

			// Execute first healthcheck
			err := useCase.Execute(t.Context(), logger.GetLogger(), time.Now().UTC(), healthcheckConfig)
			assert.NoError(t, err)

			// Verify attempt was created and marked as unavailable
			attempts, err := useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
				database.ID,
				1,
			)
			assert.NoError(t, err)
			assert.Len(t, attempts, 1)
			assert.Equal(t, databases.HealthStatusUnavailable, attempts[0].Status)

			// Verify database health status was NOT updated (SetHealthStatus should not be called)
			unavailableStatus := databases.HealthStatusUnavailable
			mockDatabaseService.AssertNotCalled(
				t,
				"SetHealthStatus",
				database.ID,
				&unavailableStatus,
			)

			// Verify no notification was sent (not marked as down yet)
			mockSender.AssertNotCalled(
				t,
				"SendNotification",
				mock.Anything,
				unavailableNotification(database.Name),
			)
		},
	)

	t.Run(
		"Test_DbShouldBeConsideredAsDownOnThirdFailedAttempt_DbMarkerdAsDownAfterThirdFailedAttempt",
		func(t *testing.T) {
			database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
			defer databases.RemoveTestDatabase(t.Context(), database)

			// Make sure DB is available
			available := databases.HealthStatusAvailable
			database.HealthStatus = &available
			err := databases.GetDatabaseService().
				SetHealthStatus(database.ID, &available)
			assert.NoError(t, err)

			// Setup mock notifier sender
			mockSender := &MockHealthcheckAttemptSender{}
			mockSender.On("SendNotification", mock.Anything, mock.Anything).Return()

			// Setup mock database service
			mockDatabaseService := &MockDatabaseService{}
			mockDatabaseService.On("ProbeDatabaseLiveness", database).
				Return(errors.New("test error"))
			unavailableStatus := databases.HealthStatusUnavailable
			mockDatabaseService.On("SetHealthStatus", database.ID, &unavailableStatus).
				Return(nil)
			mockDatabaseService.On("GetDatabaseByID", database.ID).
				Return(database, nil)

			// Setup healthcheck config requiring 3 attempts before marking as down
			healthcheckConfig := &healthcheck_config.HealthcheckConfig{
				DatabaseID:                        database.ID,
				IsHealthcheckEnabled:              true,
				IsSentNotificationWhenUnavailable: true,
				IntervalMinutes:                   1,
				AttemptsBeforeConcideredAsDown:    3,
				StoreAttemptsDays:                 7,
			}

			// Create use case with mock sender
			useCase := newTestUseCase(mockSender, mockDatabaseService)

			// Execute three failed healthchecks
			now := time.Now().UTC()
			for i := range 3 {
				err := useCase.Execute(
					t.Context(),
					logger.GetLogger(),
					now.Add(time.Duration(i)*time.Minute),
					healthcheckConfig,
				)
				assert.NoError(t, err)

				// Verify attempt was created
				attempts, err := useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
					database.ID,
					1,
				)
				assert.NoError(t, err)
				assert.Len(t, attempts, 1)
				assert.Equal(t, databases.HealthStatusUnavailable, attempts[0].Status)
			}

			// Verify database health status was updated to unavailable after 3rd attempt
			mockDatabaseService.AssertCalled(
				t,
				"SetHealthStatus",
				database.ID,
				&unavailableStatus,
			)

			// Verify notification was sent
			mockSender.AssertCalled(
				t,
				"SendNotification",
				mock.Anything,
				unavailableNotification(database.Name),
			)
		},
	)

	t.Run("Test_UnavailableDbAttemptSucceed_DbMarkedAsAvailable", func(t *testing.T) {
		database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
		defer databases.RemoveTestDatabase(t.Context(), database)

		// Make sure DB is unavailable
		unavailable := databases.HealthStatusUnavailable
		database.HealthStatus = &unavailable
		err := databases.GetDatabaseService().
			SetHealthStatus(database.ID, &unavailable)
		assert.NoError(t, err)

		// Setup mock notifier sender
		mockSender := &MockHealthcheckAttemptSender{}
		mockSender.On("SendNotification", mock.Anything, mock.Anything).Return()

		// Setup mock database service - connection succeeds
		mockDatabaseService := &MockDatabaseService{}
		mockDatabaseService.On("ProbeDatabaseLiveness", database).Return(nil)
		availableStatus := databases.HealthStatusAvailable
		mockDatabaseService.On("SetHealthStatus", database.ID, &availableStatus).
			Return(nil)
		mockDatabaseService.On("GetDatabaseByID", database.ID).
			Return(database, nil)

		// Setup healthcheck config
		healthcheckConfig := &healthcheck_config.HealthcheckConfig{
			DatabaseID:                        database.ID,
			IsHealthcheckEnabled:              true,
			IsSentNotificationWhenUnavailable: true,
			IntervalMinutes:                   1,
			AttemptsBeforeConcideredAsDown:    1,
			StoreAttemptsDays:                 7,
		}

		// Create use case with mock sender
		useCase := newTestUseCase(mockSender, mockDatabaseService)

		// Execute healthcheck (should succeed)
		err = useCase.Execute(t.Context(), logger.GetLogger(), time.Now().UTC(), healthcheckConfig)
		assert.NoError(t, err)

		// Verify attempt was created and marked as available
		attempts, err := useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
			database.ID,
			1,
		)
		assert.NoError(t, err)
		assert.Len(t, attempts, 1)
		assert.Equal(t, databases.HealthStatusAvailable, attempts[0].Status)

		// Verify database health status was updated to available
		mockDatabaseService.AssertCalled(
			t,
			"SetHealthStatus",
			database.ID,
			&availableStatus,
		)

		// Verify notification was sent for recovery
		mockSender.AssertCalled(
			t,
			"SendNotification",
			mock.Anything,
			onlineNotification(database.Name),
		)
	})

	t.Run(
		"Test_DbHealthcheckExecutedFast_HealthcheckNotExecutedFasterThanInterval",
		func(t *testing.T) {
			database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
			defer databases.RemoveTestDatabase(t.Context(), database)

			// Setup mock notifier sender
			mockSender := &MockHealthcheckAttemptSender{}
			mockSender.On("SendNotification", mock.Anything, mock.Anything).Return()

			// Setup mock database service - connection succeeds
			mockDatabaseService := &MockDatabaseService{}
			mockDatabaseService.On("ProbeDatabaseLiveness", database).Return(nil)
			availableStatus := databases.HealthStatusAvailable
			mockDatabaseService.On("SetHealthStatus", database.ID, &availableStatus).
				Return(nil)
			mockDatabaseService.On("GetDatabaseByID", database.ID).
				Return(database, nil)

			// Setup healthcheck config with 5 minute interval
			healthcheckConfig := &healthcheck_config.HealthcheckConfig{
				DatabaseID:                        database.ID,
				IsHealthcheckEnabled:              true,
				IsSentNotificationWhenUnavailable: true,
				IntervalMinutes:                   5, // 5 minute interval
				AttemptsBeforeConcideredAsDown:    1,
				StoreAttemptsDays:                 7,
			}

			// Create use case with mock sender
			useCase := newTestUseCase(mockSender, mockDatabaseService)

			// Execute first healthcheck
			now := time.Now().UTC()
			err := useCase.Execute(t.Context(), logger.GetLogger(), now, healthcheckConfig)
			assert.NoError(t, err)

			// Verify first attempt was created
			attempts, err := useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
				database.ID,
				10,
			)
			assert.NoError(t, err)
			assert.Len(t, attempts, 1)
			assert.Equal(t, databases.HealthStatusAvailable, attempts[0].Status)

			// Try to execute second healthcheck immediately (should be skipped)
			err = useCase.Execute(t.Context(), logger.GetLogger(), now.Add(1*time.Second), healthcheckConfig)
			assert.NoError(t, err)

			// Verify no new attempt was created (still only 1 attempt)
			attempts, err = useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
				database.ID,
				10,
			)
			assert.NoError(t, err)
			assert.Len(
				t,
				attempts,
				1,
				"Second healthcheck should not have been executed due to interval constraint",
			)

			// Try to execute third healthcheck after 4 minutes (still too early)
			err = useCase.Execute(t.Context(), logger.GetLogger(), now.Add(4*time.Minute), healthcheckConfig)
			assert.NoError(t, err)

			// Verify still only 1 attempt
			attempts, err = useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
				database.ID,
				10,
			)
			assert.NoError(t, err)
			assert.Len(
				t,
				attempts,
				1,
				"Third healthcheck should not have been executed due to interval constraint",
			)

			// Execute fourth healthcheck after 5 minutes (should be executed)
			err = useCase.Execute(t.Context(), logger.GetLogger(), now.Add(5*time.Minute), healthcheckConfig)
			assert.NoError(t, err)

			// Verify new attempt was created (now should be 2 attempts)
			attempts, err = useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
				database.ID,
				10,
			)
			assert.NoError(t, err)
			assert.Len(
				t,
				attempts,
				2,
				"Fourth healthcheck should have been executed after interval passed",
			)
			assert.Equal(t, databases.HealthStatusAvailable, attempts[0].Status)
			assert.Equal(t, databases.HealthStatusAvailable, attempts[1].Status)
		},
	)

	t.Run(
		"Test_CheckPhysicalDbHealth_WhenClusterIsReplicationReady_AttemptMarkedAvailable",
		func(t *testing.T) {
			for _, physicalVersionTag := range databases.PhysicalPostgresVersionTags {
				t.Run("pg"+physicalVersionTag, func(t *testing.T) {
					database := databases.CreateTestPhysicalPostgresDatabase(
						workspace.ID,
						notifier,
						physicalVersionTag,
					)
					defer databases.RemoveTestDatabase(t.Context(), database)

					mockSender := &MockHealthcheckAttemptSender{}
					mockSender.On("SendNotification", mock.Anything, mock.Anything).Return()

					useCase := newTestUseCase(mockSender, databases.GetDatabaseService())

					err := useCase.Execute(
						t.Context(),
						logger.GetLogger(),
						time.Now().UTC(),
						newTestHealthcheckConfig(database.ID),
					)
					assert.NoError(t, err)

					attempts, err := useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
						database.ID,
						1,
					)
					assert.NoError(t, err)
					assert.Len(t, attempts, 1)
					assert.Equal(t, databases.HealthStatusAvailable, attempts[0].Status)

					persistedDatabase, err := databases.GetDatabaseService().
						GetDatabaseByID(database.ID)
					assert.NoError(t, err)
					if assert.NotNil(t, persistedDatabase.HealthStatus) {
						assert.Equal(
							t,
							databases.HealthStatusAvailable,
							*persistedDatabase.HealthStatus,
						)
					}
				})
			}
		},
	)

	t.Run(
		"Test_CheckPhysicalDbHealth_WhenConnectionFails_DbMarkedAsUnavailable",
		func(t *testing.T) {
			database := databases.CreateTestPhysicalPostgresDatabase(
				workspace.ID,
				notifier,
				anyPhysicalVersionTag,
			)
			defer databases.RemoveTestDatabase(t.Context(), database)

			mockSender := &MockHealthcheckAttemptSender{}
			mockSender.On("SendNotification", mock.Anything, mock.Anything).Return()

			mockDatabaseService := &MockDatabaseService{}
			mockDatabaseService.On("ProbeDatabaseLiveness", database).
				Return(errors.New("replication connection refused"))
			unavailableStatus := databases.HealthStatusUnavailable
			mockDatabaseService.On("SetHealthStatus", database.ID, &unavailableStatus).Return(nil)
			mockDatabaseService.On("GetDatabaseByID", database.ID).Return(database, nil)

			useCase := newTestUseCase(mockSender, mockDatabaseService)

			err := useCase.Execute(
				t.Context(),
				logger.GetLogger(),
				time.Now().UTC(),
				newTestHealthcheckConfig(database.ID),
			)
			assert.NoError(t, err)

			attempts, err := useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
				database.ID,
				1,
			)
			assert.NoError(t, err)
			assert.Len(t, attempts, 1)
			assert.Equal(t, databases.HealthStatusUnavailable, attempts[0].Status)

			mockDatabaseService.AssertCalled(t, "SetHealthStatus", database.ID, &unavailableStatus)
			mockSender.AssertCalled(
				t,
				"SendNotification",
				mock.Anything,
				unavailableNotification(database.Name),
			)
		},
	)
}
