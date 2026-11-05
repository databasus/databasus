package apierror_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/logging"
)

const requestIDHeader = "X-Request-Id"

func TestHooks(t *testing.T) {
	t.Parallel()

	t.Run("answers 400 invalid_request when the strict server cannot decode the request", func(t *testing.T) {
		t.Parallel()
		hooks := apierror.NewHooks(apierror.Deps{Logger: slog.New(slog.DiscardHandler)})
		router := gin.New()
		router.POST("/api/v1/items", func(ctx *gin.Context) {
			hooks.HandleRequestError(ctx, errors.New("can't decode JSON body: unexpected EOF"))
		})

		response := serveMethod(t, router, http.MethodPost, "/api/v1/items")

		requireProblem(t, response, http.StatusBadRequest, "invalid_request")
	})

	t.Run("answers 400 invalid_request when a path parameter is malformed", func(t *testing.T) {
		t.Parallel()
		hooks := apierror.NewHooks(apierror.Deps{Logger: slog.New(slog.DiscardHandler)})
		router := gin.New()
		router.GET("/api/v1/items/:itemId", newItemHandler(hooks))

		response := serve(t, router, "/api/v1/items/not-a-uuid")

		requireProblem(t, response, http.StatusBadRequest, "invalid_request")
	})

	t.Run("answers the problem when the handler returns one", func(t *testing.T) {
		t.Parallel()
		hooks := apierror.NewHooks(apierror.Deps{Logger: slog.New(slog.DiscardHandler)})
		router := gin.New()
		router.GET("/api/v1/items/:itemId", func(ctx *gin.Context) {
			hooks.HandleHandlerError(ctx, fmt.Errorf("find item: %w", &apierror.Problem{
				Code:   apierror.CodeNotFound,
				Detail: "item not found",
				Errors: nil,
			}))
		})

		response := serve(t, router, "/api/v1/items/"+uuid.NewString())

		writtenProblem := requireProblem(t, response, http.StatusNotFound, "not_found")
		assert.Equal(t, "item not found", writtenProblem.Detail)
	})

	t.Run(
		"answers 500 without the error text and logs it under the request ID when a handler fails",
		func(t *testing.T) {
			t.Parallel()
			capturedRecords := &recordCapturer{records: nil, mutex: sync.Mutex{}}
			hooks := apierror.NewHooks(apierror.Deps{Logger: logging.NewWithSinks(capturedRecords).GetLogger()})
			router := gin.New()
			router.Use(assignRequestID)
			router.GET("/api/v1/items", func(ctx *gin.Context) {
				hooks.HandleHandlerError(ctx, errors.New(
					`ERROR: duplicate key value violates unique constraint "storages_name_key" (name)=(secret-bucket)`,
				))
			})

			response := serve(t, router, "/api/v1/items")

			writtenProblem := requireProblem(t, response, http.StatusInternalServerError, "internal")
			assert.Equal(t, "an internal error occurred", writtenProblem.Detail)
			assert.NotContains(t, response.Body.String(), "storages_name_key")
			assert.NotContains(t, response.Body.String(), "secret-bucket")

			failureRecord := capturedRecords.requireSingle(t, slog.LevelError)
			assert.Equal(t, response.Header().Get(requestIDHeader), failureRecord.requestID)
			assert.Contains(t, failureRecord.attributes["error"], "storages_name_key")
		},
	)

	t.Run("answers 500 with the fixed detail when the response cannot be encoded", func(t *testing.T) {
		t.Parallel()
		hooks := apierror.NewHooks(apierror.Deps{Logger: slog.New(slog.DiscardHandler)})
		router := gin.New()
		router.GET("/api/v1/items", func(ctx *gin.Context) {
			hooks.HandleResponseError(ctx, errors.New("json: unsupported value: NaN"))
		})

		response := serve(t, router, "/api/v1/items")

		writtenProblem := requireProblem(t, response, http.StatusInternalServerError, "internal")
		assert.Equal(t, "an internal error occurred", writtenProblem.Detail)
	})
}

// newItemHandler binds its path parameter the way a generated wrapper does, so the test reaches
// HandleParameterError through the same call.
func newItemHandler(hooks *apierror.Hooks) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var itemID openapi_types.UUID

		err := runtime.BindStyledParameterWithOptions(
			"simple",
			"itemId",
			ctx.Param("itemId"),
			&itemID,
			runtime.BindStyledParameterOptions{
				ParamLocation: runtime.ParamLocationPath,
				Explode:       false,
				Required:      true,
			},
		)
		if err != nil {
			hooks.HandleParameterError(ctx, fmt.Errorf("Invalid format for parameter itemId: %w", err),
				http.StatusBadRequest)

			return
		}

		ctx.Status(http.StatusNoContent)
	}
}

func assignRequestID(ctx *gin.Context) {
	requestID := uuid.NewString()

	ctx.Header(requestIDHeader, requestID)
	ctx.Request = ctx.Request.WithContext(logging.With(ctx.Request.Context(), slog.String("request_id", requestID)))

	ctx.Next()
}

type capturedRecord struct {
	level      slog.Level
	requestID  string
	attributes map[string]string
}

type recordCapturer struct {
	records []capturedRecord
	mutex   sync.Mutex
}

func (c *recordCapturer) Enabled(context.Context, slog.Level) bool { return true }

func (c *recordCapturer) WithAttrs([]slog.Attr) slog.Handler { return c }

func (c *recordCapturer) WithGroup(string) slog.Handler { return c }

func (c *recordCapturer) Handle(_ context.Context, record slog.Record) error {
	captured := capturedRecord{
		level:      record.Level,
		requestID:  "",
		attributes: map[string]string{},
	}

	record.Attrs(func(attr slog.Attr) bool {
		captured.attributes[attr.Key] = attr.Value.String()

		return true
	})

	captured.requestID = captured.attributes["request_id"]

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.records = append(c.records, captured)

	return nil
}

func (c *recordCapturer) requireSingle(t *testing.T, level slog.Level) capturedRecord {
	t.Helper()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	require.Len(t, c.records, 1)
	require.Equal(t, level, c.records[0].level)

	return c.records[0]
}
