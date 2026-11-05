package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/notifiers"
)

func TestServe(t *testing.T) {
	t.Run("answers requests until its context ends and then stops every loop in time", func(t *testing.T) {
		testApp := newTestApp(t)
		address := reserveFreeAddress(t)
		serveCtx, stopServing := context.WithCancel(t.Context())
		serveErrors := make(chan error, 1)

		go func() { serveErrors <- testApp.Serve(serveCtx, address) }()

		requireStatus(t, "http://"+address+"/api/v1/system/version", http.StatusOK)
		requireStatus(t, "http://"+address+"/api/v1/no-such-route", http.StatusNotFound)

		stopServing()
		stoppedAt := time.Now()

		select {
		case err := <-serveErrors:
			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Fatal("Serve did not return after its context ended")
		}

		assert.Less(t, time.Since(stoppedAt), 8*time.Second)
		assert.Zero(t, testApp.backgroundGroup.GetRunningCount())
	})

	t.Run("returns after the running job pass ends when its context ends", func(t *testing.T) {
		testApp := newTestApp(t)
		var isPassStarted, isPassFinished atomic.Bool
		testApp.runner = background.NewRunner(
			background.RunnerDeps{Logger: testApp.logger, Clock: testApp.environment.Clock},
			background.Job{
				Name:     "slow_cleanup",
				Interval: time.Hour,
				Run: func(ctx context.Context, _ time.Time) error {
					isPassStarted.Store(true)
					<-ctx.Done()
					<-time.After(200 * time.Millisecond)
					isPassFinished.Store(true)

					return nil
				},
			},
		)
		serveCtx, stopServing := context.WithCancel(t.Context())
		serveErrors := make(chan error, 1)

		address := reserveFreeAddress(t)

		go func() { serveErrors <- testApp.Serve(serveCtx, address) }()

		testwait.Until(t, "the job pass to start", func(context.Context) error {
			if !isPassStarted.Load() {
				return errors.New("the pass has not started")
			}

			return nil
		})

		stopServing()

		require.NoError(t, <-serveErrors)
		assert.True(t, isPassFinished.Load())
	})

	t.Run("lets a delivery in progress finish and record its outcome before it returns", func(t *testing.T) {
		testApp := newTestApp(t)
		var deliveredRequestCount atomic.Int64
		slowWebhook := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			<-time.After(2 * time.Second)
			deliveredRequestCount.Add(1)
			writer.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(slowWebhook.Close)
		notifierID := insertWebhookNotifier(t, testApp, slowWebhook.URL)
		serveCtx, stopServing := context.WithCancel(t.Context())
		serveErrors := make(chan error, 1)
		address := reserveFreeAddress(t)

		go func() { serveErrors <- testApp.Serve(serveCtx, address) }()

		requireStatus(t, "http://"+address+"/api/v1/system/version", http.StatusOK)
		testApp.Notify(t.Context(), []uuid.UUID{notifierID}, notifiers.Notification{
			Category: notifiers.CategoryBackupFailed,
			Heading:  "Backup failed",
			Message:  "The disk of the storage is full",
		})
		stopServing()

		require.NoError(t, <-serveErrors)
		assert.Equal(t, int64(1), deliveredRequestCount.Load())
		var lastSendError sql.NullString
		require.NoError(t, testApp.db.Raw("SELECT last_send_error FROM notifiers WHERE id = ?", notifierID).
			Row().Scan(&lastSendError))
		assert.False(t, lastSendError.Valid, lastSendError.String)
	})

	t.Run("returns an error before any loop starts when the port is busy", func(t *testing.T) {
		testApp := newTestApp(t)

		var listenConfig net.ListenConfig
		busyListener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { _ = busyListener.Close() })

		err = testApp.Serve(t.Context(), busyListener.Addr().String())

		require.Error(t, err)
		assert.Nil(t, testApp.backgroundGroup)
	})
}

func TestRouter(t *testing.T) {
	swaggerPaths := []string{"/api/v1/docs/swagger/index.html", "/api/v1/docs/swagger/doc.json"}

	for _, envMode := range []config.EnvMode{config.EnvModeProduction, config.EnvModeDevelopment} {
		t.Run("answers Problem 404 for the API documentation in "+string(envMode)+" mode", func(t *testing.T) {
			testApp := newTestApp(t)
			testApp.environment.Config.EnvMode = envMode
			handler, err := testApp.NewHandler()
			require.NoError(t, err)

			for _, swaggerPath := range swaggerPaths {
				response := httptest.NewRecorder()
				handler.ServeHTTP(
					response,
					httptest.NewRequestWithContext(t.Context(), http.MethodGet, swaggerPath, nil),
				)

				assert.Equal(t, http.StatusNotFound, response.Code, swaggerPath)
				assert.Equal(t, "application/problem+json", response.Header().Get("Content-Type"), swaggerPath)

				var problemBody struct {
					Code string `json:"code"`
				}
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problemBody), swaggerPath)
				assert.Equal(t, "not_found", problemBody.Code, swaggerPath)
			}
		})
	}
}

func TestNewRouter(t *testing.T) {
	t.Run("lets a handler read the client ip through the context it receives", func(t *testing.T) {
		testApp := newTestApp(t)
		router := testApp.newRouter()
		clientIPByRequest := make(chan string, 1)
		router.GET("/api/v1/probe", func(ginContext *gin.Context) {
			var handlerContext context.Context = ginContext

			clientIPByRequest <- httpserver.GetClientIP(handlerContext)
			ginContext.Status(http.StatusNoContent)
		})

		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/probe", nil)
		request.RemoteAddr = "203.0.113.7:51234"

		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		require.Equal(t, http.StatusNoContent, response.Code)
		assert.Equal(t, "203.0.113.7", <-clientIPByRequest)
	})
}

// insertWebhookNotifier writes a workspace and a webhook notifier whose previous delivery failed into
// the tables, because the server test has no API client to create them with.
func insertWebhookNotifier(t *testing.T, testApp *App, webhookURL string) uuid.UUID {
	t.Helper()

	workspaceID := uuid.New()
	notifierID := uuid.New()

	require.NoError(t, testApp.db.Exec("INSERT INTO workspaces (id, name) VALUES (?, ?)",
		workspaceID, "Shutdown workspace").Error)
	require.NoError(t, testApp.db.Exec(
		"INSERT INTO notifiers (id, workspace_id, name, type, last_send_error) VALUES (?, ?, ?, 'WEBHOOK', ?)",
		notifierID, workspaceID, "Slow webhook", "the previous delivery failed").Error)
	require.NoError(t, testApp.db.Exec(
		"INSERT INTO notifiers_webhook (notifier_id, webhook_url, webhook_method) VALUES (?, ?, 'POST')",
		notifierID, webhookURL).Error)

	return notifierID
}

func reserveFreeAddress(t *testing.T) string {
	t.Helper()

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	return address
}

// requireStatus waits until the server answers, since Serve opens its port in the test's goroutine.
func requireStatus(t *testing.T, url string, expectedStatus int) {
	t.Helper()

	var answeredStatus int

	testwait.Until(t, "GET "+url+" to answer", func(ctx context.Context) error {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}

		response, err := http.DefaultClient.Do(request)
		if err != nil {
			return err
		}

		_ = response.Body.Close()
		answeredStatus = response.StatusCode

		return nil
	})

	require.Equal(t, expectedStatus, answeredStatus, url)
}
