package httpserver_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	systemhttpapi "databasus-backend/internal/app/systemhttpapi"
	"databasus-backend/internal/modules/users"
)

type requestContextKey struct{}

type strictMiddlewareContextKey struct{}

func TestStrictRequestContext(t *testing.T) {
	t.Parallel()

	t.Run("retains request identity and cancellation when the gin context is reused", func(t *testing.T) {
		t.Parallel()
		firstActor := users.Actor{UserID: uuid.New(), Role: users.RoleMember}
		secondActor := users.Actor{UserID: uuid.New(), Role: users.RoleAdmin}
		firstDeadline := time.Now().Add(time.Hour)
		firstCtx, cancelFirst := context.WithDeadline(
			users.WithActor(context.WithValue(t.Context(), requestContextKey{}, "first"), firstActor),
			firstDeadline,
		)
		defer cancelFirst()
		secondCtx, cancelSecond := context.WithDeadline(
			users.WithActor(context.WithValue(t.Context(), requestContextKey{}, "second"), secondActor),
			firstDeadline.Add(time.Hour),
		)
		defer cancelSecond()

		handler := &systemRequestContextHandler{StrictServerInterface: nil, requestContexts: nil}
		adapter := systemhttpapi.NewStrictHandler(handler, []systemhttpapi.StrictMiddlewareFunc{
			func(next systemhttpapi.StrictHandlerFunc, operationID string) systemhttpapi.StrictHandlerFunc {
				return func(ctx *gin.Context, request any) (any, error) {
					ctx.Request = ctx.Request.WithContext(
						context.WithValue(ctx.Request.Context(), strictMiddlewareContextKey{}, operationID),
					)

					return next(ctx, request)
				}
			},
		})
		ginContext, router := gin.CreateTestContext(httptest.NewRecorder())
		router.ContextWithFallback = true
		ginContext.Request = httptest.NewRequestWithContext(firstCtx, http.MethodGet, "/api/v1/system/version", nil)

		adapter.GetVersion(ginContext)

		require.Len(t, handler.requestContexts, 1)

		firstRequestCtx := ginContext.Request.Context()
		ginContext.Request = httptest.NewRequestWithContext(secondCtx, http.MethodGet, "/api/v1/system/version", nil)

		adapter.GetVersion(ginContext)

		require.Len(t, handler.requestContexts, 2)
		firstHandlerCtx := handler.requestContexts[0]
		secondHandlerCtx := handler.requestContexts[1]
		assert.Same(t, firstRequestCtx, firstHandlerCtx)
		assert.Same(t, ginContext.Request.Context(), secondHandlerCtx)
		assert.IsNotType(t, &gin.Context{}, firstHandlerCtx)
		assert.Equal(t, "first", firstHandlerCtx.Value(requestContextKey{}))
		assert.Equal(t, "second", secondHandlerCtx.Value(requestContextKey{}))
		assert.Equal(t, "GetVersion", firstHandlerCtx.Value(strictMiddlewareContextKey{}))
		capturedActor, hasActor := users.GetActor(firstHandlerCtx)
		assert.True(t, hasActor)
		assert.Equal(t, firstActor, capturedActor)
		capturedActor, hasActor = users.GetActor(secondHandlerCtx)
		assert.True(t, hasActor)
		assert.Equal(t, secondActor, capturedActor)
		capturedDeadline, hasDeadline := firstHandlerCtx.Deadline()
		assert.True(t, hasDeadline)
		assert.Equal(t, firstDeadline, capturedDeadline)

		cancelSecond()

		require.NoError(t, firstHandlerCtx.Err())
		require.ErrorIs(t, secondHandlerCtx.Err(), context.Canceled)

		cancelFirst()

		require.ErrorIs(t, firstHandlerCtx.Err(), context.Canceled)
	})
}

type systemRequestContextHandler struct {
	systemhttpapi.StrictServerInterface
	requestContexts []context.Context
}

func (h *systemRequestContextHandler) GetVersion(
	ctx context.Context,
	_ systemhttpapi.GetVersionRequestObject,
) (systemhttpapi.GetVersionResponseObject, error) {
	h.requestContexts = append(h.requestContexts, ctx)

	return systemhttpapi.GetVersion200JSONResponse{Version: "test"}, nil
}
