package systemhttpapi_test

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

func TestSystemRequestContext(t *testing.T) {
	t.Parallel()

	t.Run("gives health checks the request context when a dependency fails", func(t *testing.T) {
		t.Parallel()
		logSink := &requestContextSink{requestContexts: nil, mutex: sync.Mutex{}}
		testApp := apptest.New(t, apptest.WithLogSink(logSink), apptest.WithClientToolsDir(t.TempDir()))

		response, err := testApp.NewClient(t).GetInstanceHealthWithResponse(t.Context())

		require.NoError(t, err)
		require.Equal(t, http.StatusServiceUnavailable, response.StatusCode())
		requestContexts := logSink.GetRequestContexts()
		require.NotEmpty(t, requestContexts)
		for _, requestContext := range requestContexts {
			_, isGinContext := requestContext.(*gin.Context)
			assert.False(t, isGinContext)
		}
	})
}

type requestContextSink struct {
	requestContexts []context.Context
	mutex           sync.Mutex
}

func (s *requestContextSink) Enabled(context.Context, slog.Level) bool { return true }

func (s *requestContextSink) Handle(ctx context.Context, record slog.Record) error {
	if record.Message == "instance dependency check failed" {
		s.mutex.Lock()
		s.requestContexts = append(s.requestContexts, ctx)
		s.mutex.Unlock()
	}

	return nil
}

func (s *requestContextSink) WithAttrs([]slog.Attr) slog.Handler { return s }

func (s *requestContextSink) WithGroup(string) slog.Handler { return s }

func (s *requestContextSink) GetRequestContexts() []context.Context {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return slices.Clone(s.requestContexts)
}
