package verification_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/verification/internal/agentapi"
)

type agentRequestContextKey struct{}

func TestStrictRequestContext(t *testing.T) {
	t.Parallel()

	t.Run("retains agent request metadata when the gin context is reused", func(t *testing.T) {
		t.Parallel()
		firstAgentID := uuid.New()
		secondAgentID := uuid.New()
		firstCtx, cancelFirst := context.WithCancel(
			context.WithValue(t.Context(), agentRequestContextKey{}, firstAgentID),
		)
		defer cancelFirst()
		secondCtx, cancelSecond := context.WithCancel(
			context.WithValue(t.Context(), agentRequestContextKey{}, secondAgentID),
		)
		defer cancelSecond()
		handler := &agentRequestContextHandler{StrictServerInterface: nil, requestContexts: nil}
		adapter := agentapi.NewStrictHandler(handler, nil)
		ginContext, router := gin.CreateTestContext(httptest.NewRecorder())
		router.ContextWithFallback = true
		ginContext.Request = httptest.NewRequestWithContext(firstCtx, http.MethodGet, "/backup-stream", nil)
		verificationID := uuid.New()

		adapter.StreamVerificationBackup(ginContext, firstAgentID, verificationID)

		require.Len(t, handler.requestContexts, 1)

		ginContext.Request = httptest.NewRequestWithContext(secondCtx, http.MethodGet, "/backup-stream", nil)

		adapter.StreamVerificationBackup(ginContext, secondAgentID, verificationID)

		require.Len(t, handler.requestContexts, 2)
		firstHandlerCtx := handler.requestContexts[0]
		secondHandlerCtx := handler.requestContexts[1]
		assert.Same(t, firstCtx, firstHandlerCtx)
		assert.Same(t, secondCtx, secondHandlerCtx)
		assert.IsNotType(t, &gin.Context{}, firstHandlerCtx)
		assert.Equal(t, firstAgentID, firstHandlerCtx.Value(agentRequestContextKey{}))
		assert.Equal(t, secondAgentID, secondHandlerCtx.Value(agentRequestContextKey{}))

		cancelSecond()

		require.NoError(t, firstHandlerCtx.Err())
		require.ErrorIs(t, secondHandlerCtx.Err(), context.Canceled)

		cancelFirst()

		require.ErrorIs(t, firstHandlerCtx.Err(), context.Canceled)
	})
}

type agentRequestContextHandler struct {
	agentapi.StrictServerInterface
	requestContexts []context.Context
}

func (h *agentRequestContextHandler) StreamVerificationBackup(
	ctx context.Context,
	_ agentapi.StreamVerificationBackupRequestObject,
) (agentapi.StreamVerificationBackupResponseObject, error) {
	h.requestContexts = append(h.requestContexts, ctx)

	return nil, nil
}
