package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/clienttools"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/diskusage"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/lib/toolstub"
)

func TestInstanceHealth(t *testing.T) {
	t.Run("lists every dependency and job when the instance is healthy", func(t *testing.T) {
		application := newTestApp(t)
		application.clientToolsChecker = clienttools.NewChecker(nil)
		client := newHealthClient(t, application)
		response, err := client.GetInstanceHealthWithResponse(t.Context())
		require.NoError(t, err)
		require.NotNil(t, response.JSON200, string(response.Body))
		assert.Equal(t, "*", response.HTTPResponse.Header.Get("Access-Control-Allow-Origin"))
		assert.Len(t, response.JSON200.Checks, 4)
		require.Len(t, response.JSON200.Jobs, len(application.jobs))

		names := make([]string, 0, len(response.JSON200.Jobs))
		for _, job := range response.JSON200.Jobs {
			names = append(names, job.Name)
			assert.Positive(t, job.MaxPassDurationSeconds)
			assert.Nil(t, job.LastStartedAt)
			assert.Nil(t, job.LastCompletedAt)
			assert.Nil(t, job.LastOutcome)
		}
		assert.True(t, slices.IsSorted(names))
	})

	t.Run("reports the intervals of the physical backup scheduler and the backup request job", func(t *testing.T) {
		application := newTestApp(t)
		application.clientToolsChecker = clienttools.NewChecker(nil)
		client := newHealthClient(t, application)

		response, err := client.GetInstanceHealthWithResponse(t.Context())
		require.NoError(t, err)
		require.NotNil(t, response.JSON200, string(response.Body))

		intervalSecondsByJob := make(map[string]float64, len(response.JSON200.Jobs))
		for _, job := range response.JSON200.Jobs {
			intervalSecondsByJob[job.Name] = job.IntervalSeconds
		}

		assert.InDelta(t, 15, intervalSecondsByJob["backups_physical_scheduler"], 0)
		assert.InDelta(t, 1, intervalSecondsByJob["backups_physical_request_scheduler"], 0)
	})

	for _, scenario := range []struct {
		name                string
		used                int64
		isInspectionFailure bool
		expectedStatus      int
		failureCode         string
	}{
		{"passes when disk usage is below the boundary", 94, false, 200, ""},
		{"fails when disk usage reaches the boundary", 95, false, 503, "DISK_FULL"},
		{"hides internal paths when disk inspection fails", 0, true, 503, "INSPECTION_FAILED"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			application := newTestApp(t)
			application.clientToolsChecker = clienttools.NewChecker(nil)
			application.inspectDisk = func(path string) (diskusage.Usage, error) {
				assert.Equal(t, application.environment.Config.DataDirectory, path)
				if scenario.isInspectionFailure {
					return diskusage.Usage{}, errors.New("private/password/path")
				}
				return diskusage.Usage{TotalBytes: 100, UsedBytes: scenario.used, FreeBytes: 100 - scenario.used}, nil
			}

			client := newHealthClient(t, application)
			response, err := client.GetInstanceHealthWithResponse(t.Context())
			require.NoError(t, err)
			assert.Equal(t, scenario.expectedStatus, response.StatusCode(), string(response.Body))
			assert.Equal(t, "*", response.HTTPResponse.Header.Get("Access-Control-Allow-Origin"))
			assert.NotContains(t, string(response.Body), "private/password/path")

			if scenario.failureCode != "" {
				require.NotNil(t, response.ApplicationproblemJSON503)
				assert.Equal(t, apiclient.ProblemCodeInstanceUnhealthy, response.ApplicationproblemJSON503.Code)
				assert.Equal(t, scenario.failureCode, string(*response.ApplicationproblemJSON503.Checks[0].FailureCode))
				assert.Len(t, response.ApplicationproblemJSON503.Checks, 4)
			}
		})
	}
	t.Run("retains all failures when the database and client tools fail together", func(t *testing.T) {
		application := newTestApp(t)
		application.clientToolsChecker = clienttools.NewChecker(
			[]clienttools.Bundle{
				{
					Engine:           "PostgreSQL",
					Version:          "17",
					BinDir:           t.TempDir(),
					RequiredCommands: []string{"missing"},
					IsFatal:          true,
				},
			},
		)

		client := newHealthClient(t, application)
		application.Close()
		response, err := client.GetInstanceHealthWithResponse(t.Context())
		require.NoError(t, err)
		require.NotNil(t, response.ApplicationproblemJSON503, string(response.Body))

		failedChecks := 0
		for _, check := range response.ApplicationproblemJSON503.Checks {
			if check.Outcome == apiclient.HealthCheckOutcomeFAILED {
				failedChecks++
			}
		}
		assert.Equal(t, 3, failedChecks)
		assert.NotContains(t, string(response.Body), application.environment.Config.DatabaseDsn)

		version, err := client.GetVersionWithResponse(t.Context())
		require.NoError(t, err)
		require.NotNil(t, version.JSON200)
	})
	t.Run("restores freshness when a stale pass completes with an error", func(t *testing.T) {
		application := newTestApp(t)
		application.clientToolsChecker = clienttools.NewChecker(nil)
		manualClock := clock.NewManual(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
		application.environment.Clock = manualClock

		job := background.Job{
			Name:     "controlled_work",
			Interval: time.Second,
			Run:      func(context.Context, time.Time) error { return errors.New("private token") },
		}
		application.runner = background.NewRunner(
			background.RunnerDeps{Logger: application.logger, Clock: manualClock},
			job,
		)
		application.jobHealthPolicies = newJobHealthPolicies(
			[]background.Job{job},
			application.environment.StorageFileTimings,
		)

		client := newHealthClient(t, application)
		manualClock.Advance(5 * time.Minute)
		response, err := client.GetInstanceHealthWithResponse(t.Context())
		require.NoError(t, err)
		require.NotNil(t, response.ApplicationproblemJSON503)
		assert.True(t, response.ApplicationproblemJSON503.Jobs[0].IsStale)

		require.Error(t, application.RunJobPass(t.Context(), job.Name))
		recovered, err := client.GetInstanceHealthWithResponse(t.Context())
		require.NoError(t, err)
		require.NotNil(t, recovered.JSON200, string(recovered.Body))
		assert.Equal(t, apiclient.JobHealthLastOutcomeFAILED, *recovered.JSON200.Jobs[0].LastOutcome)
		assert.NotContains(t, string(recovered.Body), "private token")
	})
}

func TestHealthClientTools(t *testing.T) {
	for _, scenario := range []struct {
		name           string
		isFatal        bool
		isPresent      bool
		expectedStatus int
	}{
		{"fails when a required client is missing", true, false, 503},
		{"fails when a required client cannot start", true, true, 503},
		{"passes when a nonessential client fails", false, true, 200},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			application := newTestApp(t)
			directory := t.TempDir()
			if scenario.isPresent {
				toolstub.Write(t, filepath.Join(directory, "tool"), toolstub.Spec{Script: "exit 1"})
			}
			application.clientToolsChecker = clienttools.NewChecker(
				[]clienttools.Bundle{
					{
						Engine:           "test",
						Version:          "1",
						BinDir:           directory,
						RequiredCommands: []string{"tool"},
						IsFatal:          scenario.isFatal,
					},
				},
			)

			response, err := newHealthClient(t, application).GetInstanceHealthWithResponse(t.Context())
			require.NoError(t, err)
			assert.Equal(t, scenario.expectedStatus, response.StatusCode(), string(response.Body))
			assert.NotContains(t, string(response.Body), directory)
		})
	}
}

func TestHealthRunningJob(t *testing.T) {
	t.Run("returns the previous outcome when a running pass stalls", func(t *testing.T) {
		application := newTestApp(t)
		application.clientToolsChecker = clienttools.NewChecker(nil)
		manualClock := clock.NewManual(time.Now().UTC())
		application.environment.Clock = manualClock

		started := make(chan struct{})
		release := make(chan struct{})
		isBlocked := false

		job := background.Job{Name: "controlled", Interval: time.Second, Run: func(context.Context, time.Time) error {
			if isBlocked {
				close(started)
				<-release
			}
			return errors.New("sensitive failure")
		}}
		application.runner = background.NewRunner(
			background.RunnerDeps{Logger: application.logger, Clock: manualClock},
			job,
		)
		application.jobHealthPolicies = newJobHealthPolicies(
			[]background.Job{job},
			application.environment.StorageFileTimings,
		)

		require.Error(t, application.RunJobPass(t.Context(), job.Name))
		isBlocked = true
		completed := make(chan error, 1)
		go func() { completed <- application.RunJobPass(t.Context(), job.Name) }()
		<-started

		client := newHealthClient(t, application)
		requestContext, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		response, err := client.GetInstanceHealthWithResponse(requestContext)
		close(release)
		require.Error(t, <-completed)
		require.NoError(t, err)
		require.NotNil(t, response.JSON200, string(response.Body))
		assert.True(t, response.JSON200.Jobs[0].IsRunning)
		assert.Equal(t, apiclient.JobHealthLastOutcomeFAILED, *response.JSON200.Jobs[0].LastOutcome)
		assert.NotNil(t, response.JSON200.Jobs[0].LastCompletedAt)
		assert.NotContains(t, string(response.Body), "sensitive failure")
	})
}

func TestInstanceHealthDiagnostics(t *testing.T) {
	t.Run("logs measurements and identifiers with the request when several monitoring checks fail", func(t *testing.T) {
		application := newTestApp(t)

		var records bytes.Buffer
		application.logger = logging.NewWithSinks(slog.NewJSONHandler(&records, nil)).GetLogger()
		application.clientToolsChecker = clienttools.NewChecker(nil)
		application.inspectDisk = func(string) (diskusage.Usage, error) {
			return diskusage.Usage{TotalBytes: 100, UsedBytes: 95, FreeBytes: 5}, nil
		}
		manualClock := clock.NewManual(time.Now().UTC())
		application.environment.Clock = manualClock

		job := background.Job{
			Name:     "stalled_work",
			Interval: time.Second,
			Run:      func(context.Context, time.Time) error { return nil },
		}
		application.runner = background.NewRunner(
			background.RunnerDeps{Logger: application.logger, Clock: manualClock},
			job,
		)
		application.jobHealthPolicies = newJobHealthPolicies(
			[]background.Job{job},
			application.environment.StorageFileTimings,
		)

		client := newHealthClient(t, application)
		signedUp, err := requestSignUp(t.Context(), client, newConsoleEmail())
		require.NoError(t, err)
		require.NotNil(t, signedUp.JSON201, string(signedUp.Body))

		createdAgent, err := client.CreateVerificationAgentWithResponse(
			t.Context(),
			apiclient.VerificationAgentInput{Name: "private agent"},
			func(_ context.Context, request *http.Request) error {
				request.Header.Set("Authorization", "Bearer "+signedUp.JSON201.Token)
				return nil
			},
		)
		require.NoError(t, err)
		require.NotNil(t, createdAgent.JSON201, string(createdAgent.Body))
		agentID := createdAgent.JSON201.Agent.Id
		require.NoError(
			t,
			application.db.Exec(
				"UPDATE verification_agents SET created_at = ? WHERE id = ?",
				time.Now().UTC().Add(-time.Hour),
				agentID,
			).Error,
		)

		manualClock.Advance(5 * time.Minute)
		response, err := client.GetInstanceHealthWithResponse(t.Context())
		require.NoError(t, err)
		require.NotNil(t, response.ApplicationproblemJSON503, string(response.Body))
		requestID := response.HTTPResponse.Header.Get("X-Request-ID")
		require.NotEmpty(t, requestID)
		assert.NotContains(t, string(response.Body), agentID.String())
		assert.NotContains(t, string(response.Body), "private agent")

		diagnostics := make(map[string]map[string]any)
		for line := range strings.SplitSeq(strings.TrimSpace(records.String()), "\n") {
			var record map[string]any
			require.NoError(t, json.Unmarshal([]byte(line), &record))
			message, isMessage := record["msg"].(string)
			require.True(t, isMessage)
			diagnostics[message] = record
		}

		for _, message := range []string{"instance disk usage exceeds health limit", "verification agent is stale", "background job is stale"} {
			require.Contains(t, diagnostics, message)
			assert.Equal(t, requestID, diagnostics[message]["request_id"])
		}
		assert.InDelta(t, 100, diagnostics["instance disk usage exceeds health limit"]["total_bytes"], 0)
		assert.InDelta(t, 95, diagnostics["instance disk usage exceeds health limit"]["used_bytes"], 0)
		assert.InDelta(t, 5, diagnostics["instance disk usage exceeds health limit"]["free_bytes"], 0)
		assert.Equal(t, agentID.String(), diagnostics["verification agent is stale"]["agent_id"])
		assert.Equal(t, "stalled_work", diagnostics["background job is stale"]["job_name"])
	})
}

func newHealthClient(t *testing.T, application *App) *apiclient.ClientWithResponses {
	t.Helper()
	handler, err := application.NewHandler()
	require.NoError(t, err)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return apiclient.New(t, server.URL)
}
