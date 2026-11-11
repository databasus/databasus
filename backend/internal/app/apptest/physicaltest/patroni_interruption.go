package physicaltest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

type PatroniInterruptionSpec struct {
	Image            string
	Kind             string
	ShouldHoldOutage bool
}

func RunPatroniInterruptedBackup(t *testing.T, spec PatroniInterruptionSpec) {
	t.Helper()

	events := &patroniScheduleEvents{}
	observer := patroniScheduleObserver{Handler: slog.DiscardHandler, Events: events}
	source := newPatroniSource(
		t,
		patroniSourceSpec{
			Image:                    spec.Image,
			ShouldIntercept:          true,
			ShouldIgnoreSlot:         true,
			LogSink:                  observer,
			ShouldKeepSlotOnAllNodes: spec.ShouldHoldOutage,
		},
	)
	connection := source.connect(t)
	createMarkerTable(t, connection)
	source.Team.enableBackups(t, true)
	chain := source.Team.waitForChain(t, 0)
	source.applySchemaChanges(t, connection)
	_, err := connection.Exec(
		t.Context(),
		`CREATE TABLE transfer_payload (id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, payload text);
		INSERT INTO transfer_payload(payload) SELECT
		(SELECT string_agg(md5(random()::text || block::text || row_number::text), '') FROM generate_series(1,128) block)
		FROM generate_series(1,8192) row_number`,
	)
	require.NoError(t, err)
	_, err = connection.Exec(t.Context(), "CHECKPOINT")
	require.NoError(t, err)
	forceWalRotation(t, connection)
	if spec.Kind == "INCREMENTAL" {
		waitForWalSummaries(t, connection, chainTipStopLSN(t, chain))
	}

	gate := source.Proxy.HoldTransfer(t)
	source.Team.call(
		t,
		physicalResponseSpec{
			Call:   apptest.PhysicalStartRequest(source.Team.Database.Id, spec.Kind),
			Status: http.StatusAccepted,
		},
	)
	var transfer containers.PostgresProxyEvent
	testwait.Until(t, "the backup transfers bytes through the controlled transport", func(context.Context) error {
		select {
		case transfer = <-gate.Reached:
			return nil
		default:
			return fmt.Errorf("backup has not reached the transfer gate")
		}
	})

	require.Positive(t, transfer.TransferredBytes)
	require.Positive(t, transfer.BackendPID)

	var transferPID int
	hasLoggedProgress := false
	testwait.Until(t, "the matched pg_basebackup backend has streamed positive bytes", func(ctx context.Context) error {
		err := connection.QueryRow(ctx, `SELECT progress.pid FROM pg_stat_progress_basebackup progress
		JOIN pg_stat_activity activity ON activity.pid=progress.pid
		WHERE progress.pid=$1 AND progress.backup_streamed > 0 AND activity.application_name='pg_basebackup' AND activity.state='active'`, transfer.BackendPID).Scan(&transferPID)
		if err != nil && !hasLoggedProgress {
			var progress string
			require.NoError(t, connection.QueryRow(ctx, `SELECT COALESCE(json_agg(json_build_object(
				'pid', progress.pid, 'phase', progress.phase, 'streamed', progress.backup_streamed,
				'application', activity.application_name, 'state', activity.state))::text, 'no server backup')
				FROM pg_stat_progress_basebackup progress JOIN pg_stat_activity activity ON activity.pid=progress.pid`).Scan(&progress))
			t.Logf("held transfer with %d archive bytes: %s", transfer.TransferredBytes, progress)
			hasLoggedProgress = true
		}

		return err
	})

	var attempt backupRow
	for _, backup := range source.Team.listBackups(t, spec.Kind) {
		if backup.Status == "IN_PROGRESS" {
			attempt = backup
			break
		}
	}

	require.NotEqual(t, uuid.Nil, attempt.ID)

	previous := source.Fixture.GetIdentity(t)
	promoted := source.Fixture.Switchover(t)
	require.NotEqual(t, previous.Node, promoted.Node)
	t.Logf(
		"interrupted %s attempt %s during backend %d transfer (%d observed bytes), timeline %d -> %d",
		spec.Kind,
		attempt.ID,
		transferPID,
		transfer.TransferredBytes,
		previous.Timeline,
		promoted.Timeline,
	)

	if spec.ShouldHoldOutage {
		source.Proxy.SetUnavailable(true)
	}

	gate.Release()
	failure := source.waitForTerminalAttempt(t, attempt.ID)
	var scheduledRetryID uuid.UUID
	switch {
	case spec.ShouldHoldOutage:
		require.Equal(t, "ERROR", failure.Status)
		require.Equal(t, "PG_BASEBACKUP_FAILED", failure.Reason)
		require.Contains(t, failure.Message, "pg_basebackup")
		hasRefusedConnection := false
		for _, event := range source.Proxy.GetEvents() {
			hasRefusedConnection = hasRefusedConnection || event.Kind == "unavailable"
		}
		require.True(t, hasRefusedConnection, "failure recheck must observe the controlled outage")
		scheduledRetryID = source.requireScheduledRetry(t, patroniRetrySpec{AttemptID: attempt.ID, Events: events})
	case spec.Kind == "FULL":
		require.Equal(t, "ERROR", failure.Status)
		require.Equal(t, "FAILOVER_DURING_BACKUP", failure.Reason)
	default:
		require.Equal(t, "CHAIN_BROKEN", failure.Status)
		require.Equal(t, "TIMELINE_SWITCH_DETECTED", failure.Reason)
	}

	source.requirePartialBackupRefused(t, attempt.ID)

	var replacement backupRow
	if spec.ShouldHoldOutage {
		replacement = source.waitForCompletedFull(t, scheduledRetryID)
	} else {
		replacement = source.waitForNewFull(t, attempt.ID)
	}

	require.Equal(t, promoted.Timeline, replacement.TimelineID)
	if !spec.ShouldHoldOutage {
		source.requireSingleReplacement(t, patroniReplacement{Preceding: attempt.ID, Replacement: replacement.ID})
	}

	point := source.captureRecoveryPoint(t, []backupRow{replacement}, "after-interrupted-"+spec.Kind)
	source.restoreRecoveryPoint(t, point)
}

type patroniAttemptFailure struct {
	Status  string
	Reason  string
	Message string
}

func (source patroniSource) waitForTerminalAttempt(t *testing.T, id uuid.UUID) patroniAttemptFailure {
	t.Helper()

	var failure patroniAttemptFailure
	testwait.Until(t, "the interrupted backup reaches a terminal failure", func(context.Context) error {
		source.Team.App.QuerySQL(
			t,
			&failure,
			"SELECT status, COALESCE(error_reason,'') AS reason, COALESCE(fail_message,'') AS message FROM backups_physical WHERE id=?",
			id,
		)
		if failure.Status == "ERROR" || failure.Status == "CHAIN_BROKEN" {
			return nil
		}

		if failure.Status == "COMPLETED" {
			t.Fatalf("backup %s completed instead of being interrupted", id)
		}

		return fmt.Errorf("backup %s: %+v", id, failure)
	})

	return failure
}

func (source patroniSource) requirePartialBackupRefused(t *testing.T, id uuid.UUID) {
	t.Helper()

	status, body := source.Team.App.CallPhysicalAPI(t, source.Team.Owner, apptest.PhysicalBackupTokenRequest(id))
	require.Equal(t, http.StatusConflict, status, "%s", body)
	require.NotContains(t, string(body), "\"token\"")
}
