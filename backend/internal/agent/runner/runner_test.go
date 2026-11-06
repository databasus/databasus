package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/agent/api"
	"databasus-backend/internal/agent/config"
	"databasus-backend/internal/agent/dbconn"
	"databasus-backend/internal/agent/restore"
	"databasus-backend/internal/agent/verifier"
	"databasus-backend/internal/engines/postgresql/logical/pgrestore"
	"databasus-backend/internal/lib/agentclient"
	"databasus-backend/internal/lib/background"
)

func testCapacity() config.Capacity {
	return config.Capacity{
		MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2,
		CPUPerJob: 2, RAMMbPerJob: 2048,
	}
}

func postgresJob() *agentclient.VerificationAssignment {
	return &agentclient.VerificationAssignment{
		VerificationID:     uuid.New(),
		BackupID:           uuid.New(),
		BackupSizeMb:       50,
		MaxContainerDiskMb: 200,
		Database: agentclient.AssignedDatabase{
			Type:              "POSTGRES_LOGICAL",
			PostgresqlLogical: &agentclient.AssignedPostgresql{Version: "16"},
		},
	}
}

func newTestRunner(t *testing.T,
	apiClient APIClient, spawner Spawner, restorer Restorer,
	stats StatsCollector, reg Registrar,
) *Runner {
	t.Helper()
	verificationRunner := NewRunner(
		Deps{
			DiagnosticSecrets: nil,
			APIClient:         apiClient,
			Capacity:          testCapacity(),
			Pool:              NewPool(2),
			Spawner:           spawner,
			Restorer:          restorer,
			StatsCollector:    stats,
			Heartbeat:         reg,
			Logger:            slog.New(slog.DiscardHandler),
			Group:             newTestGroup(t),
		},
	)
	verificationRunner.backupDownloadBackoffFn = func(int) time.Duration { return 0 }
	return verificationRunner
}

func testConn() dbconn.Conn {
	return dbconn.Conn{
		Host: "127.0.0.1", Port: 5432,
		User: "postgres", Password: "pw", Database: "postgres",
	}
}

func fakeContainerWith(conn dbconn.Conn) *fakeContainer {
	return &fakeContainer{inContainerConn: conn, verifierConn: conn}
}

func okSpawner() *fakeSpawner {
	return &fakeSpawner{container: fakeContainerWith(testConn())}
}

func TestExecuteJob(t *testing.T) {
	t.Run("reports failed without exit code when the PostgreSQL major version is unsupported", func(t *testing.T) {
		apiClient := &fakeAPI{}
		r := newTestRunner(t, apiClient, okSpawner(), &fakeRestorer{}, &fakeStats{}, newFakeRegistrar())

		job := postgresJob()
		job.Database.Type = "MYSQL"

		r.executeJob(t.Context(), job)

		report, ok := apiClient.lastReport()
		require.True(t, ok)
		assert.Equal(t, agentclient.VerificationReportStatusFAILED, report.Status)
		assert.Nil(t, report.PgRestoreExitCode)
	})
	t.Run("reports failed without exit code when container creation fails", func(t *testing.T) {
		apiClient := &fakeAPI{}
		spawner := &fakeSpawner{err: errors.New("image pull failed with agent-secret")}
		r := newTestRunner(t, apiClient, spawner, &fakeRestorer{}, &fakeStats{}, newFakeRegistrar())
		r.diagnosticSecrets = []string{"agent-secret"}

		r.executeJob(t.Context(), postgresJob())

		report, ok := apiClient.lastReport()
		require.True(t, ok)
		assert.Equal(t, agentclient.VerificationReportStatusFAILED, report.Status)
		assert.Nil(t, report.PgRestoreExitCode)
		require.NotNil(t, report.FailureDiagnostic)
		assert.Contains(t, report.FailureDiagnostic.Output, "image pull failed")
		assert.NotContains(t, report.FailureDiagnostic.Output, "agent-secret")
		assert.NotContains(t, *report.FailMessage, "agent-secret")
	})
	t.Run("sends no report when the backup stream returns 410", func(t *testing.T) {
		apiClient := &fakeAPI{downloadErr: &api.ResponseError{Op: "backup stream", StatusCode: 410}}
		r := newTestRunner(t, apiClient, okSpawner(), &fakeRestorer{}, &fakeStats{}, newFakeRegistrar())

		r.executeJob(t.Context(), postgresJob())

		assert.Equal(t, 0, apiClient.reportCalled, "a 410 on the stream is an abort — no report")
	})
	t.Run("reports failed without exit code when the backup stream exhausts retries", func(t *testing.T) {
		apiClient := &fakeAPI{downloadErr: errors.New("connection refused")}
		r := newTestRunner(t, apiClient, okSpawner(), &fakeRestorer{}, &fakeStats{}, newFakeRegistrar())

		r.executeJob(t.Context(), postgresJob())

		report, ok := apiClient.lastReport()
		require.True(t, ok)
		assert.Equal(t, agentclient.VerificationReportStatusFAILED, report.Status)
		assert.Nil(t, report.PgRestoreExitCode)
	})
	t.Run("reports failed without exit code when the restore exhausts disk space", func(t *testing.T) {
		apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
		restorer := &fakeRestorer{
			runErr:    fmt.Errorf("restore: %w", restore.ErrRestoreFailed),
			runResult: restore.Result{PgRestoreExitCode: 1, StderrTail: "could not write: No space left on device"},
		}
		r := newTestRunner(t, apiClient, okSpawner(), restorer, &fakeStats{}, newFakeRegistrar())

		r.executeJob(t.Context(), postgresJob())

		report, ok := apiClient.lastReport()
		require.True(t, ok)
		assert.Equal(t, agentclient.VerificationReportStatusFAILED, report.Status)
		assert.Nil(t, report.PgRestoreExitCode,
			"hitting the estimate-derived disk ceiling is agent infra, not BackupRejected")
	})
	t.Run("reports terminal disk limit exceeded when disk usage reaches its ceiling", func(t *testing.T) {
		apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
		jobContainer := fakeContainerWith(testConn())
		jobContainer.diskUsageBytes.Store(1 << 40)
		restorer := &fakeRestorer{runBlocks: true}

		r := newTestRunner(t,
			apiClient, &fakeSpawner{container: jobContainer}, restorer, &fakeStats{}, newFakeRegistrar())
		r.diskWatchInterval = time.Millisecond

		r.executeJob(t.Context(), postgresJob())

		report, ok := apiClient.lastReport()
		require.True(t, ok)
		assert.Equal(t, agentclient.VerificationReportStatusFAILED, report.Status)
		require.NotNil(t, report.FailureKind)
		assert.Equal(t, "DISK_LIMIT_EXCEEDED", *report.FailureKind)
		assert.Nil(t, report.PgRestoreExitCode, "a disk-limit verdict carries no pg_restore exit code")
		assert.True(t, jobContainer.terminated, "the container must be torn down on disk-limit abort")
	})
	t.Run(
		"reports terminal disk limit exceeded when disk usage reaches its ceiling during owner role creation",
		func(t *testing.T) {
			apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
			jobContainer := fakeContainerWith(testConn())
			restorer := &fakeRestorer{
				ensureRolesBlocks:    true,
				onEnsureRolesEntered: func() { jobContainer.diskUsageBytes.Store(1 << 40) },
			}

			r := newTestRunner(t,
				apiClient, &fakeSpawner{container: jobContainer}, restorer, &fakeStats{}, newFakeRegistrar())

			r.diskWatchInterval = time.Millisecond

			r.executeJob(t.Context(), postgresJob())

			report, ok := apiClient.lastReport()
			require.Contains(t, restorer.calls, "roles", "the watcher must trip inside the owner-roles stage")
			require.True(t, ok, "a stage cancelled by the watcher must still post the disk verdict")
			assert.Equal(t, agentclient.VerificationReportStatusFAILED, report.Status)
			require.NotNil(t, report.FailureKind)
			assert.Equal(t, "DISK_LIMIT_EXCEEDED", *report.FailureKind)
		},
	)
	t.Run("does not falsely trip when the server sends no disk budget", func(t *testing.T) {
		apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
		jobContainer := fakeContainerWith(testConn())
		jobContainer.diskUsageBytes.Store(1 << 40)

		job := postgresJob()
		job.MaxContainerDiskMb = 0

		r := newTestRunner(t,
			apiClient, &fakeSpawner{container: jobContainer}, &fakeRestorer{}, &fakeStats{}, newFakeRegistrar())

		r.executeJob(t.Context(), job)

		report, ok := apiClient.lastReport()
		require.True(t, ok)
		assert.Equal(t, agentclient.VerificationReportStatusCOMPLETED, report.Status,
			"a zero budget disables the watcher; the job must not be falsely failed")
		assert.Nil(t, report.FailureKind)
	})
	t.Run("reports failed with exit code when pg_restore fails", func(t *testing.T) {
		apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
		restorer := &fakeRestorer{
			runErr:    fmt.Errorf("restore: %w", restore.ErrRestoreFailed),
			runResult: restore.Result{PgRestoreExitCode: 1, DurationMs: 42, StderrTail: "syntax error in dump"},
		}
		r := newTestRunner(t, apiClient, okSpawner(), restorer, &fakeStats{}, newFakeRegistrar())

		r.executeJob(t.Context(), postgresJob())

		report, ok := apiClient.lastReport()
		require.True(t, ok)
		assert.Equal(t, agentclient.VerificationReportStatusFAILED, report.Status)
		require.NotNil(t, report.PgRestoreExitCode)
		assert.Equal(t, 1, *report.PgRestoreExitCode)
	})
	t.Run("reports completed when the only restore errors concern missing extensions", func(t *testing.T) {
		apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
		restorer := &fakeRestorer{
			runErr: fmt.Errorf("restore: %w", restore.ErrRestoreFailed),
			runResult: restore.Result{
				PgRestoreExitCode:      1,
				DurationMs:             10,
				HasOnlyIgnorableErrors: true,
				UnavailableExtensions:  []string{"set_user"},
			},
		}
		stats := &fakeStats{stats: verifier.Stats{DBSizeBytes: 9_000_000, TableCount: 3}}
		r := newTestRunner(t, apiClient, okSpawner(), restorer, stats, newFakeRegistrar())
		r.connAlive = func(context.Context, dbconn.Conn) bool { return true }

		r.executeJob(t.Context(), postgresJob())

		report, ok := apiClient.lastReport()
		require.True(t, ok)
		assert.Nil(t, report.FailureDiagnostic)
		assert.Equal(t, agentclient.VerificationReportStatusCOMPLETED, report.Status,
			"a restore whose only failures are missing extensions is restorable")
		require.NotNil(t, report.PgRestoreExitCode)
		assert.Equal(t, 1, *report.PgRestoreExitCode,
			"the real non-zero exit code is preserved on the success report")
		require.NotNil(t, report.DBSizeBytesAfterRestore)
		assert.Equal(t, int64(9_000_000), *report.DBSizeBytesAfterRestore)
	})
	t.Run("reports completed when the only restore error concerns the existing public schema", func(t *testing.T) {
		apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
		restorer := &fakeRestorer{
			runErr:    fmt.Errorf("restore: %w", restore.ErrRestoreFailed),
			runResult: restore.Result{PgRestoreExitCode: 1, DurationMs: 10, HasOnlyIgnorableErrors: true},
		}
		stats := &fakeStats{stats: verifier.Stats{DBSizeBytes: 9_000_000, TableCount: 3}}
		r := newTestRunner(t, apiClient, okSpawner(), restorer, stats, newFakeRegistrar())
		r.connAlive = func(context.Context, dbconn.Conn) bool { return true }

		r.executeJob(t.Context(), postgresJob())

		report, ok := apiClient.lastReport()
		require.True(t, ok)
		assert.Equal(t, agentclient.VerificationReportStatusCOMPLETED, report.Status,
			"an archive dumped with -n public collides with the target's own public schema; the data restored")
	})
	t.Run("reports failed without exit code when the restore command cannot execute", func(t *testing.T) {
		apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
		restorer := &fakeRestorer{
			runErr: errors.New("exec create failed"),
			runResult: restore.Result{
				FailureDiagnostic: &pgrestore.FailureDiagnostic{
					Output:      "captured execution stderr",
					IsTruncated: false,
				},
			},
		}
		r := newTestRunner(t, apiClient, okSpawner(), restorer, &fakeStats{}, newFakeRegistrar())

		r.executeJob(t.Context(), postgresJob())

		report, ok := apiClient.lastReport()
		require.True(t, ok)
		assert.Equal(t, agentclient.VerificationReportStatusFAILED, report.Status)
		assert.Nil(t, report.PgRestoreExitCode)
		require.NotNil(t, report.FailureDiagnostic)
		assert.Contains(t, report.FailureDiagnostic.Output, "exec create failed")
		assert.Contains(t, report.FailureDiagnostic.Output, "captured execution stderr")
	})
	t.Run("reports failed without exit code when verification fails and its connection is dead", func(t *testing.T) {
		apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
		restorer := &fakeRestorer{runResult: restore.Result{PgRestoreExitCode: 0}}
		stats := &fakeStats{err: errors.New("tier 1 failed")}
		r := newTestRunner(t, apiClient, okSpawner(), restorer, stats, newFakeRegistrar())
		r.connAlive = func(context.Context, dbconn.Conn) bool { return false }

		r.executeJob(t.Context(), postgresJob())

		report, ok := apiClient.lastReport()
		require.True(t, ok)
		assert.Equal(t, agentclient.VerificationReportStatusFAILED, report.Status)
		assert.Nil(t, report.PgRestoreExitCode, "a dead connection is agent infra (retryable)")
	})
	t.Run("reports failed with exit code zero when verification fails and its connection is alive", func(t *testing.T) {
		apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
		restorer := &fakeRestorer{runResult: restore.Result{PgRestoreExitCode: 0}}
		stats := &fakeStats{err: errors.New("tier 1 impossible result " + testConn().Password)}
		restorer.runResult.FailureDiagnostic = &pgrestore.FailureDiagnostic{
			Output:      "successful restore warning",
			IsTruncated: false,
		}
		r := newTestRunner(t, apiClient, okSpawner(), restorer, stats, newFakeRegistrar())
		r.connAlive = func(context.Context, dbconn.Conn) bool { return true }

		r.executeJob(t.Context(), postgresJob())

		report, ok := apiClient.lastReport()
		require.True(t, ok)
		assert.Equal(t, agentclient.VerificationReportStatusFAILED, report.Status)
		require.NotNil(t, report.PgRestoreExitCode)
		assert.Equal(t, 0, *report.PgRestoreExitCode,
			"a live conn with a broken tier-1 result is BackupRejected with exit 0")
		require.NotNil(t, report.FailureDiagnostic)
		assert.Contains(t, report.FailureDiagnostic.Output, "tier 1 impossible result")
		assert.NotContains(t, report.FailureDiagnostic.Output, "successful restore warning")
		assert.NotContains(t, report.FailureDiagnostic.Output, testConn().Password)
		assert.NotContains(t, *report.FailMessage, testConn().Password)
	})
	t.Run("reports completed with stats when every stage succeeds", func(t *testing.T) {
		apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
		restorer := &fakeRestorer{runResult: restore.Result{PgRestoreExitCode: 0, DurationMs: 1234}}
		stats := &fakeStats{stats: verifier.Stats{
			DBSizeBytes: 9_000_000,
			SchemaCount: 2,
			TableCount:  3,
			TableStats:  []verifier.TableStat{{SchemaName: "public", Name: "t1", RowCount: 10}},
		}}
		r := newTestRunner(t, apiClient, okSpawner(), restorer, stats, newFakeRegistrar())
		r.connAlive = func(context.Context, dbconn.Conn) bool { return true }

		r.executeJob(t.Context(), postgresJob())

		report, ok := apiClient.lastReport()
		require.True(t, ok)
		assert.Equal(t, agentclient.VerificationReportStatusCOMPLETED, report.Status)
		require.NotNil(t, report.DBSizeBytesAfterRestore)
		assert.Equal(t, int64(9_000_000), *report.DBSizeBytesAfterRestore)
		require.NotNil(t, report.TableCount)
		assert.Equal(t, 3, *report.TableCount)
		require.Len(t, report.TableStats, 1)
		assert.Equal(t, "public", report.TableStats[0].SchemaName)
	})
	t.Run("runs pre restore then restore then post restore when a TimescaleDB job succeeds", func(t *testing.T) {
		apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
		restorer := &fakeRestorer{runResult: restore.Result{PgRestoreExitCode: 0}}
		stats := &fakeStats{stats: verifier.Stats{TableCount: 1}}
		r := newTestRunner(t, apiClient, okSpawner(), restorer, stats, newFakeRegistrar())
		r.connAlive = func(context.Context, dbconn.Conn) bool { return true }

		r.executeJob(t.Context(), timescaleJob())

		report, ok := apiClient.lastReport()
		require.True(t, ok)
		assert.Equal(t, agentclient.VerificationReportStatusCOMPLETED, report.Status)
		assert.Equal(t, []string{"roles", "pre", "restore", "post"}, restorer.calls,
			"timescaledb restore must enter restoring mode, restore, then leave it")
		assert.Equal(t, 1, restorer.runParallelJobs,
			"timescaledb restore must be single-threaded to keep _timescaledb_catalog FK order")
		assert.True(t, restorer.runIsTimescaledb,
			"the restorer decides on --clean from this flag")
	})
	t.Run("does not run timescale hooks when the job uses plain PostgreSQL", func(t *testing.T) {
		apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
		restorer := &fakeRestorer{runResult: restore.Result{PgRestoreExitCode: 0}}
		stats := &fakeStats{stats: verifier.Stats{TableCount: 1}}
		r := newTestRunner(t, apiClient, okSpawner(), restorer, stats, newFakeRegistrar())
		r.connAlive = func(context.Context, dbconn.Conn) bool { return true }

		r.executeJob(t.Context(), postgresJob())

		assert.Equal(t, []string{"roles", "restore"}, restorer.calls,
			"a non-timescaledb restore must not run the timescaledb hooks")
		assert.Equal(t, 2, restorer.runParallelJobs,
			"a non-timescaledb restore keeps parallel jobs (CPUPerJob)")
		assert.False(t, restorer.runIsTimescaledb,
			"the restorer decides on --clean from this flag")
	})
	t.Run("reports failed without restoring when owner role creation fails", func(t *testing.T) {
		apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
		restorer := &fakeRestorer{ensureRolesErr: errors.New("connect restore target: refused")}
		r := newTestRunner(t, apiClient, okSpawner(), restorer, &fakeStats{}, newFakeRegistrar())

		r.executeJob(t.Context(), postgresJob())

		report, ok := apiClient.lastReport()
		require.True(t, ok)
		assert.Equal(t, agentclient.VerificationReportStatusFAILED, report.Status)
		assert.Nil(t, report.PgRestoreExitCode,
			"role setup is agent infrastructure, so the backup must not be blamed")
		assert.Equal(t, []string{"roles"}, restorer.calls)
	})
	t.Run("sends no report when the job is aborted before its failure report", func(t *testing.T) {
		apiClient := &fakeAPI{}
		spawner := &fakeSpawner{err: errors.New("spawn failed")}
		reg := newFakeRegistrar()
		reg.aborted = true
		r := newTestRunner(t, apiClient, spawner, &fakeRestorer{}, &fakeStats{}, reg)

		r.executeJob(t.Context(), postgresJob())

		assert.Equal(t, 0, apiClient.reportCalled,
			"an aborted/cancelled verification must not get a spurious FAILED report")
	})
}

func timescaleJob() *agentclient.VerificationAssignment {
	job := postgresJob()
	job.TimescaledbVersion = "2.17.0"

	return job
}

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("returns the panic after draining jobs when a restore panics", func(t *testing.T) {
		t.Parallel()
		apiClient := &fakeAPI{claims: []*agentclient.VerificationAssignment{postgresJob()}}
		spawner := okSpawner()
		registrar := newFakeRegistrar()
		restorer := &fakeRestorer{onEnsureRolesEntered: func() { panic("restore panic") }}
		verificationRunner := newTestRunner(t, apiClient, spawner, restorer, &fakeStats{}, registrar)
		var logOutput bytes.Buffer
		verificationRunner.group = background.New(background.Deps{
			Logger: slog.New(slog.NewTextHandler(&logOutput, nil)),
		})

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		err := verificationRunner.Run(ctx)
		var panicError *background.PanicError
		require.ErrorAs(t, err, &panicError)
		assert.Equal(t, "restore panic", panicError.Value)
		assert.NotEmpty(t, panicError.Stack)

		jobContainer, isFakeContainer := spawner.container.(*fakeContainer)
		require.True(t, isFakeContainer)
		assert.True(t, jobContainer.terminated)
		assert.Empty(t, registrar.registered)
		assert.Empty(t, verificationRunner.pool.slots)

		require.NoError(t, verificationRunner.group.Wait(ctx))
		assert.Contains(t, logOutput.String(), "background goroutine panicked")
		assert.Contains(t, logOutput.String(), "verification_job")
		assert.Contains(t, logOutput.String(), "restore panic")
	})
	t.Run("cancels and drains sibling jobs when a restore panics", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			apiClient := &fakeAPI{claims: []*agentclient.VerificationAssignment{postgresJob(), postgresJob()}}
			spawner := &recordingSpawner{}
			registrar := newFakeRegistrar()
			restorer := &panickingRestorer{
				Restorer:               &fakeRestorer{},
				blockedRestoreStarted:  make(chan struct{}),
				blockedRestoreFinished: make(chan struct{}),
			}
			verificationRunner := newTestRunner(t, apiClient, spawner, restorer, &fakeStats{}, registrar)
			ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
			defer cancel()
			startedAt := time.Now()

			err := verificationRunner.Run(ctx)
			var panicError *background.PanicError
			require.ErrorAs(t, err, &panicError)
			assert.Equal(t, "restore panic", panicError.Value)
			require.NoError(t, ctx.Err(), "job cancellation must precede the parent deadline")
			assert.Less(t, time.Since(startedAt), 5*time.Second)
			assert.True(t, restorer.isBlockedRestoreCancelled.Load())

			select {
			case <-restorer.blockedRestoreFinished:
			default:
				t.Error("Run returned before the blocked restore finished")
			}

			jobContainers := spawner.snapshot()
			require.Len(t, jobContainers, 2)
			for _, jobContainer := range jobContainers {
				assert.True(t, jobContainer.terminated.Load(), "Run must wait for every container's cleanup")
			}

			assert.Empty(t, registrar.registered)
			assert.Empty(t, verificationRunner.pool.slots)
		})
	})
	t.Run("returns the panic when the parent context is also cancelled", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		apiClient := &fakeAPI{claims: []*agentclient.VerificationAssignment{postgresJob()}}
		restorer := &fakeRestorer{onEnsureRolesEntered: func() {
			cancel()
			panic("restore panic during shutdown")
		}}
		verificationRunner := newTestRunner(t, apiClient, okSpawner(), restorer, &fakeStats{}, newFakeRegistrar())

		err := verificationRunner.Run(ctx)
		var panicError *background.PanicError
		require.ErrorAs(t, err, &panicError)
		assert.Equal(t, "restore panic during shutdown", panicError.Value)
	})
	t.Run("stops without an error when the parent context has a cancellation cause", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancelCause(t.Context())
		cancel(errors.New("parent shutdown"))
		verificationRunner := newTestRunner(
			t,
			&fakeAPI{},
			okSpawner(),
			&fakeRestorer{},
			&fakeStats{},
			newFakeRegistrar(),
		)

		require.NoError(t, verificationRunner.Run(ctx))
	})

	t.Run("drains and stops when the context is cancelled", func(t *testing.T) {
		t.Parallel()

		apiClient := &fakeAPI{}
		r := newTestRunner(t, apiClient, okSpawner(), &fakeRestorer{}, &fakeStats{}, newFakeRegistrar())

		ctx, cancel := context.WithCancel(t.Context())

		done := make(chan error, 1)
		go func() {
			done <- r.Run(ctx)
		}()

		cancel()

		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Run did not return after context cancellation")
		}
	})
	t.Run("panics when Run is called twice", func(t *testing.T) {
		t.Parallel()

		r := newTestRunner(t, &fakeAPI{}, okSpawner(), &fakeRestorer{}, &fakeStats{}, newFakeRegistrar())

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		require.NoError(t, r.Run(ctx))

		assert.Panics(t, func() { _ = r.Run(ctx) })
	})
}

func newTestGroup(t *testing.T) *background.Group {
	t.Helper()
	group := background.New(background.Deps{Logger: slog.New(slog.DiscardHandler)})
	t.Cleanup(func() { _ = group.Wait(context.WithoutCancel(t.Context())) })
	return group
}

func TestExecuteJobMissingFailureDiagnostic(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name       string
		diagnostic *pgrestore.FailureDiagnostic
	}{
		{name: "omits the diagnostic when a rejected restore has no captured stderr", diagnostic: nil},
		{name: "omits the diagnostic after sanitization removes all captured output", diagnostic: &pgrestore.FailureDiagnostic{Output: "\x00 \n", IsTruncated: false}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
			restorer := &fakeRestorer{
				runErr:    fmt.Errorf("restore: %w", restore.ErrRestoreFailed),
				runResult: restore.Result{PgRestoreExitCode: 1, FailureDiagnostic: scenario.diagnostic},
			}
			verificationRunner := newTestRunner(t, apiClient, okSpawner(), restorer, &fakeStats{}, newFakeRegistrar())

			verificationRunner.executeJob(t.Context(), postgresJob())

			report, isReported := apiClient.lastReport()
			require.True(t, isReported)
			assert.Equal(t, agentclient.VerificationReportStatusFAILED, report.Status)
			require.NotNil(t, report.PgRestoreExitCode)
			assert.Equal(t, 1, *report.PgRestoreExitCode)
			require.NotNil(t, report.FailMessage)
			assert.Contains(t, *report.FailMessage, "pg_restore failed")
			assert.Nil(t, report.FailureDiagnostic)
		})
	}
}
