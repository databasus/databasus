package restoring

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/lib/cache"
)

type restoreExecution struct {
	ctx           context.Context
	restoreID     uuid.UUID
	databaseCache *RestoreDatabaseCache
}

type recordingRestoreExecutor struct {
	executions chan restoreExecution
}

func (e *recordingRestoreExecutor) MakeRestore(
	ctx context.Context,
	restoreID uuid.UUID,
	databaseCache *RestoreDatabaseCache,
) {
	e.executions <- restoreExecution{ctx, restoreID, databaseCache}
}

func Test_StartRestore_WhenProviderFails_CarriesMetadataIntoDetachedWorker(t *testing.T) {
	providerError := errors.New("provider failed")
	executor := &recordingRestoreExecutor{executions: make(chan restoreExecution, 1)}
	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	scheduler := &RestoresScheduler{
		logger:   testLogger,
		restorer: executor,
		restoreDatabaseCache: cache.NewJSONStore[RestoreDatabaseCache](
			NewFailingRestoreMetadataStore(providerError),
			"restore_db",
		),
	}
	restoreID := uuid.New()
	databaseCache := &RestoreDatabaseCache{
		PostgresqlLogicalDatabase: &restores_core.PostgresqlTargetDatabase{
			Host: "database.internal",
		},
	}
	requestContext, cancelRequest := context.WithCancel(t.Context())

	require.NoError(t, scheduler.StartRestore(requestContext, restoreID, databaseCache))
	cancelRequest()
	execution := <-executor.executions

	assert.Equal(t, restoreID, execution.restoreID)
	assert.Same(t, databaseCache, execution.databaseCache)
	assert.NoError(t, execution.ctx.Err())
}
