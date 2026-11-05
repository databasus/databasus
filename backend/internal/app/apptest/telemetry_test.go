package apptest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

type telemetryReceiver struct {
	mutex      sync.Mutex
	reports    []map[string]any
	userAgents []string
	status     int
}

func (r *telemetryReceiver) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var report map[string]any
	if request.Method != http.MethodPost || request.Header.Get("Content-Type") != "application/json" {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := json.NewDecoder(request.Body).Decode(&report); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.reports = append(r.reports, report)
	r.userAgents = append(r.userAgents, request.UserAgent())
	writer.WriteHeader(r.status)
}

func (r *telemetryReceiver) getReports() []map[string]any {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return append([]map[string]any(nil), r.reports...)
}

func (r *telemetryReceiver) getUserAgents() []string {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return append([]string(nil), r.userAgents...)
}

func TestTelemetry(t *testing.T) {
	t.Parallel()

	t.Run("delivers anonymous api state after warmup and reuses its identity", func(t *testing.T) {
		t.Parallel()

		receiver := &telemetryReceiver{status: http.StatusAccepted}
		server := httptest.NewServer(receiver)
		t.Cleanup(server.Close)
		identityPath := filepath.Join(t.TempDir(), "instance.json")
		var instanceID any
		var installedAt any
		for range 2 {
			t.Run("reports after application starts", func(t *testing.T) {
				application := apptest.New(
					t,
					apptest.WithTelemetry(
						apptest.TelemetrySettings{ReceiverURL: server.URL, IdentityPath: identityPath},
					),
				)
				admin := application.CreateAdmin(t)
				workspace := apptest.CreateWorkspace(t, admin, "private workspace")
				application.CreateLocalStorage(t, admin, workspace.Id)

				var storageSettings apiclient.StorageSettings
				require.NoError(t, storageSettings.FromS3StorageSettings(apiclient.S3StorageSettings{
					Type: apiclient.S3StorageSettingsTypeS3, S3Bucket: "private-bucket",
					S3AccessKey: new("private-access"), S3SecretKey: new("private-secret"),
				}))
				apptest.CreateStorage(t, admin, workspace.Id, storageSettings)
				application.CreateLocalStorage(t, admin, workspace.Id)

				var notifierSettings apiclient.NotifierSettings
				require.NoError(t, notifierSettings.FromWebhookNotifierSettings(apiclient.WebhookNotifierSettings{
					Type:               apiclient.WebhookNotifierSettingsTypeWEBHOOK,
					WebhookMethod:      apiclient.WebhookNotifierSettingsWebhookMethodPOST,
					WebhookUrl:         "http://localhost/private-destination",
					Headers:            []apiclient.WebhookHeader{},
					AcceptedCategories: []apiclient.NotificationCategory{apiclient.NotificationCategoryALL},
				}))
				apptest.CreateNotifier(t, admin, workspace.Id, notifierSettings)
				require.NoError(t, notifierSettings.FromEmailNotifierSettings(apiclient.EmailNotifierSettings{
					Type:     apiclient.EmailNotifierSettingsTypeEMAIL,
					Security: apiclient.SmtpSecurityNONE, SmtpHost: "localhost", SmtpPort: 2525,
					TargetEmail: "private-recipient@example.com",
				}))
				apptest.CreateNotifier(t, admin, workspace.Id, notifierSettings)

				initialCount := len(receiver.getReports())
				application.RunJobPass(t, "telemetry_ping")
				application.Clock.Advance(time.Minute - time.Nanosecond)
				application.RunJobPass(t, "telemetry_ping")
				assert.Len(t, receiver.getReports(), initialCount)
				if initialCount == 0 {
					assert.NoFileExists(t, identityPath)
				}
				application.Clock.Advance(time.Nanosecond)
				application.RunJobPass(t, "telemetry_ping")

				reports := receiver.getReports()
				require.Len(t, reports, initialCount+1)
				report := reports[initialCount]
				assert.InDelta(t, 1, report["userCount"], 0)
				assert.Equal(t, []any{"LOCAL", "LOCAL", "S3"}, report["storages"])
				assert.Equal(t, []any{"EMAIL", "WEBHOOK"}, report["notifiers"])
				assert.Equal(
					t,
					"Databasus-Telemetry/"+getTestVersion(t, application),
					receiver.getUserAgents()[initialCount],
				)
				assert.Equal(t, []any{}, report["databases"])
				if instanceID != nil {
					assert.Equal(t, instanceID, report["instanceID"])
					assert.Equal(t, installedAt, report["installedAt"])
				}

				instanceID = report["instanceID"]
				installedAt = "2020-01-01"
				encoded, err := json.Marshal(report)
				require.NoError(t, err)
				assert.NotContains(t, string(encoded), "private workspace")
				assert.NotContains(t, string(encoded), admin.Token)
				assert.NotContains(t, string(encoded), "private-")

				identity, err := os.Stat(identityPath)
				require.NoError(t, err)
				assert.Equal(t, os.FileMode(0o600), identity.Mode().Perm())
				persistedIdentity, err := json.Marshal(map[string]any{
					"instanceID": instanceID, "installedAt": installedAt,
				})
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(identityPath, persistedIdentity, 0o600))
			})
		}
	})
	t.Run("reports empty arrays and omits user count when no user exists", func(t *testing.T) {
		t.Parallel()

		receiver := &telemetryReceiver{status: http.StatusOK}
		server := httptest.NewServer(receiver)
		t.Cleanup(server.Close)
		identityPath := filepath.Join(t.TempDir(), "instance.json")
		application := apptest.New(
			t,
			apptest.WithTelemetry(apptest.TelemetrySettings{ReceiverURL: server.URL, IdentityPath: identityPath}),
		)
		application.RunJobPass(t, "telemetry_ping")
		application.Clock.Advance(time.Minute)
		application.RunJobPass(t, "telemetry_ping")
		reports := receiver.getReports()
		require.Len(t, reports, 1)
		assert.NotContains(t, reports[0], "userCount")
		for _, collection := range []string{"databases", "storages", "notifiers", "verificationAgents"} {
			assert.Equal(t, []any{}, reports[0][collection])
		}
		assert.Equal(t, application.Clock.Now().UTC().Format("2006-01-02"), reports[0]["installedAt"])
	})

	t.Run("sends nothing when reporting is disabled", func(t *testing.T) {
		t.Parallel()

		receiver := &telemetryReceiver{status: http.StatusOK}
		server := httptest.NewServer(receiver)
		t.Cleanup(server.Close)
		identityPath := filepath.Join(t.TempDir(), "never-created.json")
		application := apptest.New(
			t,
			apptest.WithTelemetry(apptest.TelemetrySettings{ReceiverURL: server.URL, IdentityPath: identityPath}),
			apptest.WithoutTelemetry(),
		)
		application.RunJobPass(t, "telemetry_ping")
		application.Clock.Advance(48 * time.Hour)
		application.RunJobPass(t, "telemetry_ping")
		application.CreateAdmin(t)
		assert.Empty(t, receiver.getReports())
		assert.NoFileExists(t, identityPath)
	})
	t.Run("preserves corrupt identity when a report is due", func(t *testing.T) {
		t.Parallel()

		receiver := &telemetryReceiver{status: http.StatusNoContent}
		server := httptest.NewServer(receiver)
		t.Cleanup(server.Close)
		identityPath := filepath.Join(t.TempDir(), "instance.json")
		require.NoError(t, os.WriteFile(identityPath, []byte("corrupt"), 0o600))
		application := apptest.New(
			t,
			apptest.WithTelemetry(apptest.TelemetrySettings{ReceiverURL: server.URL, IdentityPath: identityPath}),
		)
		application.RunJobPass(t, "telemetry_ping")
		application.Clock.Advance(time.Minute)
		application.RunJobPass(t, "telemetry_ping")
		assert.Empty(t, receiver.getReports())
		identity, err := os.ReadFile(identityPath)
		require.NoError(t, err)
		assert.Equal(t, "corrupt", string(identity))
	})
}

func TestTelemetryIdentityFailures(t *testing.T) {
	t.Parallel()

	for _, scenario := range []string{"unreadable directory", "unwritable parent"} {
		t.Run("skips reporting when identity has an "+scenario, func(t *testing.T) {
			t.Parallel()

			receiver := &telemetryReceiver{status: http.StatusOK}
			server := httptest.NewServer(receiver)
			t.Cleanup(server.Close)
			identityPath := filepath.Join(t.TempDir(), "identity.json")
			if scenario == "unreadable directory" {
				require.NoError(t, os.Mkdir(identityPath, 0o700))
			} else {
				parentPath := filepath.Join(t.TempDir(), "file")
				require.NoError(t, os.WriteFile(parentPath, []byte("occupied"), 0o600))
				identityPath = filepath.Join(parentPath, "identity.json")
			}
			application := apptest.New(
				t,
				apptest.WithTelemetry(apptest.TelemetrySettings{ReceiverURL: server.URL, IdentityPath: identityPath}),
			)
			application.RunJobPass(t, "telemetry_ping")
			application.Clock.Advance(time.Minute)
			application.RunJobPass(t, "telemetry_ping")
			assert.Empty(t, receiver.getReports())
			version, err := application.NewClient(t).GetVersionWithResponse(t.Context())
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, version.StatusCode())
		})
	}
}
