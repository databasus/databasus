package physical

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/process"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

const crossingSegmentSize uint64 = 16 * 1024 * 1024

// TestTimelineCrossing pins the PostgreSQL behavior that restore and backup across a promotion rely
// on: which copy of the fork segment replay reads, how the promoted server refuses a start beyond
// the fork point, and when it accepts an incremental backup against a manifest from the former
// timeline.
//
//nolint:tparallel // Each version's servers stop before the next version starts, which bounds the memory the servers of one test hold.
func TestTimelineCrossing(t *testing.T) {
	t.Parallel()

	for _, image := range getSupportedVersionImages() {
		t.Run(image, func(t *testing.T) {
			t.Run(
				"replays a timeline 1 chain to timeline 2 from the child copy of the fork segment",
				func(t *testing.T) {
					runChainAcrossPromotion(t, image)
				},
			)

			t.Run(
				"realigns reception to the fork segment after the former primary's tail was archived",
				func(t *testing.T) {
					runReceptionAfterLostTail(t, image)
				},
			)

			t.Run("refuses an incremental backup across a promotion without wal summaries", func(t *testing.T) {
				runIncrementalWithoutStandbySummaries(t, image)
			})
		})
	}
}

func runChainAcrossPromotion(t *testing.T, image string) {
	t.Helper()

	lab := newCrossingLab(t, image)
	primaryConn := lab.primary.connectAs(t, getAdminRole())
	standbyConn := lab.standby.connectAs(t, getAdminRole())
	execStatements(t, primaryConn, "CREATE TABLE crossing_marker (phase text NOT NULL)")
	insertCrossingMarker(t, primaryConn, "before-full")
	standbyArchive := t.TempDir()
	primaryArchive := t.TempDir()
	lab.startReceiver(t, crossingReceiverSpec{Server: lab.primary, Conn: primaryConn, Directory: primaryArchive})
	lab.startReceiver(t, crossingReceiverSpec{Server: lab.standby, Conn: standbyConn, Directory: standbyArchive})

	full := lab.requireBackup(t, lab.primary, "")
	insertCrossingMarker(t, primaryConn, "after-full")
	incremental := lab.requireBackup(t, lab.primary, full)
	insertCrossingMarker(t, primaryConn, "after-incremental")
	waitForCrossingMarker(t, standbyConn, "after-incremental")

	promoteCrossingStandby(t, standbyConn)
	insertCrossingMarker(t, standbyConn, "after-promotion")
	insertCrossingMarker(t, primaryConn, "lost-on-former-primary")
	fork := lab.readFork(t, 2)
	promotedIncremental := lab.requireBackup(t, lab.standby, incremental)
	insertCrossingMarker(t, standbyConn, "after-promoted-incremental")

	lab.switchAndWaitForSegment(t, standbyConn, standbyArchive, 2)
	lab.switchAndWaitForSegment(t, primaryConn, primaryArchive, 1)
	forkSegmentNo := walmath.NewWalSegmentNo(fork.ForkLSN, crossingSegmentSize)
	parentForkSegment := forkSegmentNo.GetFilename(1, crossingSegmentSize)
	waitForCompleteSegment(t, primaryArchive, parentForkSegment)
	waitForArchivedFile(t, standbyArchive, "00000002.history")

	archive := append(
		listCompleteArchiveFiles(t, standbyArchive),
		filepath.Join(primaryArchive, parentForkSegment),
		filepath.Join(standbyArchive, "00000002.history"),
	)
	expectedMarkers := []string{
		"before-full", "after-full", "after-incremental", "after-promotion", "after-promoted-incremental",
	}

	replay := lab.restore(t, crossingRestoreSpec{
		Backups: []string{full, incremental}, ArchiveFiles: archive, TargetTimelineID: 2,
	})
	require.ElementsMatch(t, expectedMarkers, replay.Markers)
	t.Logf("files requested through restore_command: %v", replay.RequestedFiles)
	require.Contains(t, replay.RequestedFiles, "00000002.history")
	require.Contains(t, replay.RequestedFiles, forkSegmentNo.GetFilename(2, crossingSegmentSize))
	require.NotContains(
		t,
		replay.RequestedFiles,
		parentForkSegment,
		"replay must read the fork segment from the promoted timeline, not the former primary's copy",
	)

	combinedReplay := lab.restore(t, crossingRestoreSpec{
		Backups: []string{full, incremental, promotedIncremental}, ArchiveFiles: archive, TargetTimelineID: 2,
	})
	require.ElementsMatch(t, expectedMarkers, combinedReplay.Markers)
}

func runReceptionAfterLostTail(t *testing.T, image string) {
	t.Helper()

	lab := newCrossingLab(t, image)
	primaryConn := lab.primary.connectAs(t, getAdminRole())
	standbyConn := lab.standby.connectAs(t, getAdminRole())
	execStatements(t, primaryConn, "CREATE TABLE crossing_marker (phase text NOT NULL)")
	insertCrossingMarker(t, primaryConn, "before-full")
	primaryArchive := t.TempDir()
	stopPrimaryReceiver := lab.startReceiver(
		t,
		crossingReceiverSpec{Server: lab.primary, Conn: primaryConn, Directory: primaryArchive},
	)

	full := lab.requireBackup(t, lab.primary, "")
	insertCrossingMarker(t, primaryConn, "replicated-before-failover")
	waitForCrossingMarker(t, standbyConn, "replicated-before-failover")
	detachCrossingStandby(t, standbyConn)
	insertCrossingMarker(t, primaryConn, "lost-on-former-primary")
	lab.switchAndWaitForSegment(t, primaryConn, primaryArchive, 1)
	insertCrossingMarker(t, primaryConn, "lost-later-on-former-primary")
	lab.switchAndWaitForSegment(t, primaryConn, primaryArchive, 1)
	stopPrimaryReceiver()

	promoteCrossingStandby(t, standbyConn)
	fork := lab.readFork(t, 2)
	forkSegmentNo := walmath.NewWalSegmentNo(fork.ForkLSN, crossingSegmentSize)
	queue := t.TempDir()
	for _, archivedFile := range listCompleteArchiveFiles(t, primaryArchive) {
		copyCrossingFile(t, archivedFile, filepath.Join(queue, filepath.Base(archivedFile)))
	}
	slotName := createCrossingSlot(t, standbyConn)

	refusal, err := lab.runTool(
		t,
		pgshared.ExecutablePgReceivewal,
		"--directory="+queue, "--slot="+slotName, "--no-loop", "--verbose",
		"--dbname="+lab.standby.getConnString(getAdminRole(), false),
	)
	require.Error(t, err, "pg_receivewal must not resume beyond the fork point:\n%s", refusal)
	t.Logf("refusal for a start beyond the fork point: %s", refusal)
	refusedTimelineID, isForkRefusal := getForkRefusalTimeline(refusal)
	require.True(t, isForkRefusal, "the refusal must be recognized:\n%s", refusal)
	require.Equal(t, 1, refusedTimelineID)

	realignQueueToForkSegment(t, queue, forkSegmentNo.GetFilename(1, crossingSegmentSize))
	lab.startReceiver(
		t,
		crossingReceiverSpec{Server: lab.standby, Conn: standbyConn, Directory: queue, SlotName: slotName},
	)
	insertCrossingMarker(t, standbyConn, "after-promotion")
	lab.switchAndWaitForSegment(t, standbyConn, queue, 2)
	childForkSegment := forkSegmentNo.GetFilename(2, crossingSegmentSize)
	waitForCompleteSegment(t, queue, childForkSegment)
	waitForArchivedFile(t, queue, "00000002.history")

	replay := lab.restore(t, crossingRestoreSpec{
		Backups: []string{full},
		ArchiveFiles: append(
			listCompleteArchiveFiles(t, primaryArchive),
			append(listCompleteArchiveFiles(t, queue), filepath.Join(queue, "00000002.history"))...,
		),
		TargetTimelineID: 2,
	})
	require.ElementsMatch(t, []string{"before-full", "replicated-before-failover", "after-promotion"}, replay.Markers)
	require.Contains(t, replay.RequestedFiles, childForkSegment)
}

func runIncrementalWithoutStandbySummaries(t *testing.T, image string) {
	t.Helper()

	lab := newCrossingLab(t, image, containers.WithoutStandbySummarizer())
	primaryConn := lab.primary.connectAs(t, getAdminRole())
	standbyConn := lab.standby.connectAs(t, getAdminRole())
	execStatements(t, primaryConn, "CREATE TABLE crossing_marker (phase text NOT NULL)")
	full := lab.requireBackup(t, lab.primary, "")
	insertCrossingMarker(t, primaryConn, "after-full")
	waitForCrossingMarker(t, standbyConn, "after-full")

	promoteCrossingStandby(t, standbyConn)
	insertCrossingMarker(t, standbyConn, "after-promotion")

	refusal, err := lab.takeBackup(t, lab.standby, full)

	require.Error(t, err, "an incremental backup must not run on a promoted node without wal summaries")
	t.Logf("refusal without wal summaries: %s", refusal)
	require.True(t, IsWalSummarizationDisabledError(refusal), "the refusal must be recognized:\n%s", refusal)

	execStatements(t, standbyConn, "ALTER SYSTEM SET summarize_wal = on", "SELECT pg_reload_conf()")
	lab.requireBackup(t, lab.standby, full)
}

type crossingLab struct {
	image   string
	version pgshared.Version
	tools   pgshared.ClientTools
	primary testServer
	standby testServer
}

func newCrossingLab(t *testing.T, image string, options ...containers.PhysicalPostgresOption) crossingLab {
	t.Helper()

	primary, standby := containers.StartPhysicalPrimaryWithStandby(t, image, options...)
	version := pgshared.Version(strconv.Itoa(containers.PostgresMajorVersion(image)))

	return crossingLab{
		image:   image,
		version: version,
		tools:   getWalTimeClientTools(t),
		primary: testServer{endpoint: primary, image: image, version: version},
		standby: testServer{endpoint: standby, image: image, version: version},
	}
}

func (lab crossingLab) runTool(t *testing.T, executable pgshared.Executable, args ...string) ([]byte, error) {
	t.Helper()

	return process.CombinedOutput(t.Context(), process.Spec{
		Path: lab.tools.GetExecutable(lab.version, executable),
		Args: args,
		Env:  []string{"LC_ALL=C"},
	})
}

func (lab crossingLab) takeBackup(t *testing.T, server testServer, parentBackup string) ([]byte, error) {
	t.Helper()

	directory := filepath.Join(t.TempDir(), "backup")
	args := []string{
		"--pgdata=" + directory,
		"--wal-method=none",
		"--checkpoint=fast",
		"--no-sync",
		"--dbname=" + server.getConnString(getAdminRole(), false),
	}
	if parentBackup != "" {
		args = append(args, "--incremental="+filepath.Join(parentBackup, "backup_manifest"))
	}

	output, err := lab.runTool(t, pgshared.ExecutablePgBasebackup, args...)
	if err != nil {
		return output, err
	}

	return []byte(directory), nil
}

func (lab crossingLab) requireBackup(t *testing.T, server testServer, parentBackup string) string {
	t.Helper()

	output, err := lab.takeBackup(t, server, parentBackup)
	require.NoError(t, err, "pg_basebackup failed:\n%s", output)

	return string(output)
}

type crossingReceiverSpec struct {
	Server    testServer
	Conn      *pgx.Conn
	Directory string
	SlotName  string
}

// startReceiver streams WAL into the directory until the returned function or the test stops it.
// Without --no-loop it reconnects after the server ends a timeline, as a promotion does.
func (lab crossingLab) startReceiver(t *testing.T, spec crossingReceiverSpec) func() {
	t.Helper()

	slotName := spec.SlotName
	if slotName == "" {
		slotName = createCrossingSlot(t, spec.Conn)
	}

	ctx, cancel := context.WithCancel(t.Context())
	var output lockedOutput
	running, err := process.Start(ctx, process.Spec{
		Path: lab.tools.GetExecutable(lab.version, pgshared.ExecutablePgReceivewal),
		Args: []string{
			"--directory=" + spec.Directory,
			"--slot=" + slotName,
			"--synchronous",
			"--verbose",
			"--status-interval=1",
			"--dbname=" + spec.Server.getConnString(getAdminRole(), false),
		},
		Env:        []string{"LC_ALL=C"},
		Stderr:     &output,
		StopSignal: syscall.SIGINT,
		KillAfter:  10 * time.Second,
	})
	require.NoError(t, err)

	isStopped := false
	stop := func() {
		if isStopped {
			return
		}

		isStopped = true
		cancel()
		_ = running.Wait()
		t.Logf("pg_receivewal into %s:\n%s", spec.Directory, output.String())
	}
	t.Cleanup(stop)

	return stop
}

func (lab crossingLab) readFork(t *testing.T, timelineID int) TimelineFork {
	t.Helper()

	history, err := lab.standby.newSource(getAdminRole()).GetTimelineHistory(
		t.Context(),
		TimelineHistoryRequest{ExpectedSystemIdentifier: lab.primary.readSystemIdentifier(t), TimelineID: timelineID},
	)
	require.NoError(t, err)

	return history.Fork
}

func (lab crossingLab) switchAndWaitForSegment(t *testing.T, conn *pgx.Conn, directory string, timelineID uint32) {
	t.Helper()

	var currentLSN string
	require.NoError(t, conn.QueryRow(t.Context(), "SELECT pg_current_wal_lsn()::text").Scan(&currentLSN))
	lsn, err := walmath.ParseLSN(currentLSN)
	require.NoError(t, err)
	execStatements(t, conn, "SELECT pg_switch_wal()")

	segment := walmath.NewWalSegmentNo(lsn, crossingSegmentSize).GetFilename(timelineID, crossingSegmentSize)
	waitForCompleteSegment(t, directory, segment)
}

type crossingRestoreSpec struct {
	Backups          []string
	ArchiveFiles     []string
	TargetTimelineID int
}

type crossingReplay struct {
	Markers        []string
	RequestedFiles []string
}

// restore combines the backups on the host, ships the archive inside the cluster directory and
// replays it in a container that serves the directory, the way a restored bundle is served.
func (lab crossingLab) restore(t *testing.T, spec crossingRestoreSpec) crossingReplay {
	t.Helper()

	volume := filepath.Join(t.TempDir(), "volume")
	clusterDir := volume
	containerClusterDir := "/var/lib/postgresql/data"
	if major := containers.PostgresMajorVersion(lab.image); major >= 18 {
		clusterDir = filepath.Join(volume, strconv.Itoa(major), "docker")
		containerClusterDir = "/var/lib/postgresql/" + strconv.Itoa(major) + "/docker"
	}
	require.NoError(t, os.MkdirAll(filepath.Dir(clusterDir), 0o700))

	output, err := lab.runTool(
		t,
		pgshared.ExecutablePgCombinebackup,
		append([]string{"--output=" + clusterDir, "--no-sync"}, spec.Backups...)...,
	)
	require.NoError(t, err, "pg_combinebackup failed:\n%s", output)
	require.NoError(t, os.Chmod(clusterDir, 0o700))

	archiveDir := filepath.Join(clusterDir, "crossing_archive")
	require.NoError(t, os.Mkdir(archiveDir, 0o700))
	for _, archivedFile := range spec.ArchiveFiles {
		copyCrossingFile(t, archivedFile, filepath.Join(archiveDir, filepath.Base(archivedFile)))
	}

	recoverySettings := fmt.Sprintf(
		"restore_command = 'echo %%f >> %[1]s/crossing_requested.log; cp %[1]s/crossing_archive/%%f %%p'\n"+
			"recovery_target_timeline = '%[2]d'\n",
		containerClusterDir,
		spec.TargetTimelineID,
	)
	appendCrossingFile(t, filepath.Join(clusterDir, "postgresql.auto.conf"), recoverySettings)
	require.NoError(t, os.WriteFile(filepath.Join(clusterDir, "recovery.signal"), nil, 0o600))

	restored := testServer{
		endpoint: containers.StartPostgresWithBoundDataDir(t, lab.image, volume),
		image:    lab.image,
		version:  lab.version,
	}
	restoredConn := restored.connectAs(t, getAdminRole())

	requestedLog, err := os.ReadFile(filepath.Join(clusterDir, "crossing_requested.log"))
	require.NoError(t, err)

	return crossingReplay{
		Markers:        readCrossingMarkers(t, restoredConn),
		RequestedFiles: strings.Fields(string(requestedLog)),
	}
}

func createCrossingSlot(t *testing.T, conn *pgx.Conn) string {
	t.Helper()

	slotName := newSlotName()
	execStatements(t, conn, fmt.Sprintf("SELECT pg_create_physical_replication_slot('%s', true)", slotName))

	return slotName
}

func insertCrossingMarker(t *testing.T, conn *pgx.Conn, phase string) {
	t.Helper()

	_, err := conn.Exec(t.Context(), "INSERT INTO crossing_marker (phase) VALUES ($1)", phase)
	require.NoError(t, err)
}

func readCrossingMarkers(t *testing.T, conn *pgx.Conn) []string {
	t.Helper()

	rows, err := conn.Query(t.Context(), "SELECT phase FROM crossing_marker")
	require.NoError(t, err)
	phases, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)

	return phases
}

func waitForCrossingMarker(t *testing.T, conn *pgx.Conn, phase string) {
	t.Helper()

	testwait.Until(t, "the marker "+phase+" is replayed", func(ctx context.Context) error {
		var isReplayed bool
		if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM crossing_marker WHERE phase = $1)", phase).
			Scan(&isReplayed); err != nil {
			return err
		}

		if !isReplayed {
			return fmt.Errorf("marker %s is not replayed", phase)
		}

		return nil
	})
}

func promoteCrossingStandby(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	var isPromoted bool
	require.NoError(t, conn.QueryRow(t.Context(), "SELECT pg_promote(true, 60)").Scan(&isPromoted))
	require.True(t, isPromoted)
}

// detachCrossingStandby stops the standby's replication from the primary while the standby stays
// in recovery, so the primary's later writes never reach it.
func detachCrossingStandby(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	execStatements(t, conn, "ALTER SYSTEM SET primary_conninfo = ''", "SELECT pg_reload_conf()")
	testwait.Until(t, "the standby stops receiving wal", func(ctx context.Context) error {
		var isReceiving bool
		if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_stat_wal_receiver)").
			Scan(&isReceiving); err != nil {
			return err
		}

		if isReceiving {
			return fmt.Errorf("the standby still receives wal")
		}

		return nil
	})
}

// realignQueueToForkSegment leaves only a zero-filled partial copy of the fork segment on the
// former timeline. pg_receivewal resumes at the start of the newest partial segment, so the server
// sends the former timeline up to the fork point and continues on the promoted timeline.
func realignQueueToForkSegment(t *testing.T, queue, parentForkSegment string) {
	t.Helper()

	entries, err := os.ReadDir(queue)
	require.NoError(t, err)
	for _, entry := range entries {
		_, segmentNo, parseErr := walmath.ParseWALFilename(entry.Name(), crossingSegmentSize)
		_, forkSegmentNo, forkErr := walmath.ParseWALFilename(parentForkSegment, crossingSegmentSize)
		require.NoError(t, forkErr)
		if parseErr == nil && segmentNo >= forkSegmentNo {
			require.NoError(t, os.Remove(filepath.Join(queue, entry.Name())))
		}
	}

	require.NoError(
		t,
		os.WriteFile(
			filepath.Join(queue, parentForkSegment+".partial"),
			make([]byte, crossingSegmentSize),
			0o600,
		),
	)
}

func waitForCompleteSegment(t *testing.T, directory, segment string) {
	t.Helper()

	testwait.Until(t, "segment "+segment+" is archived", func(context.Context) error {
		if !IsCompleteSegmentFile(filepath.Join(directory, segment), int64(crossingSegmentSize)) {
			return fmt.Errorf("segment %s is not complete in %s", segment, directory)
		}

		return nil
	})
}

func waitForArchivedFile(t *testing.T, directory, filename string) {
	t.Helper()

	testwait.Until(t, filename+" is archived", func(context.Context) error {
		_, err := os.Stat(filepath.Join(directory, filename))

		return err
	})
}

func listCompleteArchiveFiles(t *testing.T, directory string) []string {
	t.Helper()

	entries, err := os.ReadDir(directory)
	require.NoError(t, err)

	var paths []string
	for _, entry := range entries {
		path := filepath.Join(directory, entry.Name())
		if walmath.IsWalFilename(entry.Name(), crossingSegmentSize) &&
			IsCompleteSegmentFile(path, int64(crossingSegmentSize)) {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)

	return paths
}

func copyCrossingFile(t *testing.T, source, destination string) {
	t.Helper()

	content, err := os.ReadFile(source)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(destination, content, 0o600))
}

func appendCrossingFile(t *testing.T, path, content string) {
	t.Helper()

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	require.NoError(t, err)
	_, err = io.WriteString(file, content)
	require.NoError(t, err)
	require.NoError(t, file.Close())
}

type lockedOutput struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (o *lockedOutput) Write(p []byte) (int, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	return o.buffer.Write(p)
}

func (o *lockedOutput) String() string {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	return o.buffer.String()
}
