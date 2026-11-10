package physicaltest

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

type SchemaRestoreSpec struct {
	Image           string
	Stage           SchemaRecoveryStage
	HasIncrementals bool
}

type recoveryObservation struct {
	Schema    SchemaState
	Sequences []SequenceState
}

func RunSchemaRecovery(t *testing.T, spec SchemaRestoreSpec) {
	t.Helper()

	team := setupReplicationOnlyTeam(t, physicalSourceSpec{
		Image: spec.Image, BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
	})
	sourceConn := openConnAt(t, team.Settings.Host, team.Settings.Port)
	clockConn := openConnAt(t, team.Settings.Host, team.Settings.Port)
	schema := NewSchemaRecovery(t, sourceConn)
	sequences := NewSequenceRecovery(t, sourceConn, schema)

	team.enableBackups(t, true)
	chain := team.waitForChain(t, 0)
	schema.CreatePostFullObjects(t, sourceConn)
	sequences.PrepareCalls(t, sourceConn)
	if spec.HasIncrementals {
		chain = team.buildIncremental(
			t,
			sourceConn,
			incrementalSpec{ParentStopLSN: chainTipStopLSN(t, chain), Count: 1},
		)
	}
	schema.ApplyBetweenIncrementals(t, sourceConn)
	if spec.HasIncrementals {
		chain = team.buildIncremental(
			t,
			sourceConn,
			incrementalSpec{ParentStopLSN: chainTipStopLSN(t, chain), Count: 2},
		)
	}

	schema.RequireStage(t, sourceConn, SchemaBeforeWal)
	sequences.RequireDefinitions(t, sourceConn, SchemaBeforeWal)
	observations := map[SchemaRecoveryStage]recoveryObservation{
		SchemaBeforeWal: {Schema: schema.GetState(t, sourceConn), Sequences: sequences.GetStates(t, sourceConn)},
	}
	schema.BeginWalChanges(t, sourceConn)
	sequences.ApplyWalChanges(t, sourceConn)
	beforeTarget := targetTimeWithMargin(t, clockConn)
	schema.execute(t, sourceConn, "COMMIT")
	afterTarget := targetTimeWithMargin(t, clockConn)
	schema.RequireStage(t, sourceConn, SchemaAfterWal)
	sequences.RequireDefinitions(t, sourceConn, SchemaAfterWal)
	observations[SchemaAfterWal] = recoveryObservation{
		Schema: schema.GetState(t, sourceConn), Sequences: sequences.GetStates(t, sourceConn),
	}
	schema.ApplyLaterChanges(t, sourceConn)
	schema.RequireStage(t, sourceConn, SchemaLatest)
	observations[SchemaLatest] = recoveryObservation{
		Schema: schema.GetState(t, sourceConn), Sequences: sequences.GetStates(t, sourceConn),
	}

	throughLSN := currentLSN(t, sourceConn)
	forceWalRotation(t, sourceConn)
	team.waitForReplayableThroughLSN(t, chain, throughLSN)
	incrementalCount := 0
	if spec.HasIncrementals {
		incrementalCount = 2
	}
	require.Len(t, team.waitForChain(t, incrementalCount), incrementalCount+1)
	require.Len(t, team.listBackups(t, "FULL"), 1)
	require.Len(t, team.listBackups(t, "INCREMENTAL"), incrementalCount)

	var targetTime *time.Time
	switch spec.Stage {
	case SchemaBeforeWal:
		targetTime = &beforeTarget
	case SchemaAfterWal:
		targetTime = &afterTarget
	case SchemaLatest:
		targetTime = nil
	}
	bundle := team.downloadRestoreBundle(t, targetTime)
	target := prepareRestoreTarget(t, spec.Image)
	reconstructCluster(
		t,
		target,
		restoreSpec{App: team.App, Image: spec.Image, BundlePath: bundle, TargetTime: targetTime},
	)
	startRestoredCluster(t, target, spec.Image)
	restoredConn := openConnAt(t, target.Host(), target.MappedPort())
	require.Equal(t, observations[spec.Stage].Schema, schema.GetState(t, restoredConn))
	require.Equal(t, observations[spec.Stage].Sequences, sequences.GetStates(t, restoredConn))
	schema.RequireStage(t, restoredConn, spec.Stage)
	sequences.RequireDefinitions(t, restoredConn, spec.Stage)

	sequences.RequireAllocations(t, restoredConn, spec.Stage)

	secondRestoredConn := openConnAt(t, target.Host(), target.MappedPort())
	sequences.RequireDistinctCaches(t, []*pgx.Conn{restoredConn, secondRestoredConn})
	schema.RequireWrites(t, restoredConn)
	schema.RequireObservationSensitivity(t, restoredConn, spec.Stage)
	sequences.RequireDefinitionSensitivity(t, restoredConn)
}
