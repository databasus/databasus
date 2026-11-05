package backups_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

//nolint:tparallel // The subtests share one source server.
func TestCancelRestore(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("returns 202 and leaves no restore tool process when the restore runs", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		runningRestore, targetSpec := startBlockedRestore(t, tm.owner, server, database.backup.Id)
		require.NotEmpty(t, listRestoreToolProcesses(t, targetSpec.databaseName))
		waitForRestoreLogRecord(
			t,
			tm,
			restoreLogRecord{message: "executing pg_restore command", restoreID: runningRestore.Id},
		)
		require.True(t, hasRestoreToolStartRecord(t, tm, targetSpec.databaseName))

		response := cancelRestore(t, tm.owner, runningRestore.Id)

		require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))
		waitForRestoreStatus(t, tm.owner, runningRestore.Id, apiclient.RestoreStatusCANCELED)
		assert.Empty(t, listRestoreToolProcesses(t, targetSpec.databaseName))
	})

	t.Run("returns 202 and starts no restore tool when the archive still downloads", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		slowBackupFile := replaceBackupFileWithSlowFile(t, tm, database)
		archiveDirsBefore := listRestoreArchiveDirs(t, tm)
		targetSpec := server.newTargetSpec(t)
		targetSpec.cpuCount = 2

		downloadingRestore := requireStartedRestore(
			t,
			tm.owner,
			database.backup.Id,
			server.newRestoreTarget(t, targetSpec),
		)
		archiveDir := waitForArchiveCopy(t, archiveCopySpec{
			team: tm, archiveDirsBefore: archiveDirsBefore, servedBytes: slowBackupFile.servedBytes,
		})

		response := cancelRestore(t, tm.owner, downloadingRestore.Id)
		require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))
		slowBackupFile.serveRest(t)

		waitForRestoreStatus(t, tm.owner, downloadingRestore.Id, apiclient.RestoreStatusCANCELED)
		waitForRestoreLogRecord(t, tm, restoreLogRecord{
			message: "logical restore cancelled on request", restoreID: downloadingRestore.Id,
		})
		assert.NoDirExists(t, archiveDir)
		assert.False(t, hasRestoreToolStartRecord(t, tm, targetSpec.databaseName),
			"the log records a start of pg_restore into %s", targetSpec.databaseName)
		assert.False(t, server.hasItemsTable(t, targetSpec.databaseName))
	})

	t.Run("returns 409 not_running when the restore completed", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		completedRestore := requireStartedRestore(t, tm.owner, database.backup.Id,
			server.newRestoreTarget(t, server.newTargetSpec(t)))
		waitForRestoreStatus(t, tm.owner, completedRestore.Id, apiclient.RestoreStatusCOMPLETED)

		response := cancelRestore(t, tm.owner, completedRestore.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeNotRunning)
		assert.Equal(t, apiclient.RestoreStatusCOMPLETED, getRestore(t, tm.owner, completedRestore.Id).Status)
	})
}

// slowBackupFile stands in the local storage for a backup's file. It is a named pipe that serves the
// first half of the file at once and the rest only when the test calls serveRest, so a restore that
// reads it stays in the middle of its download.
type slowBackupFile struct {
	pipe        *os.File
	servedBytes []byte
	restBytes   []byte
}

func replaceBackupFileWithSlowFile(t *testing.T, tm team, database backedUpDatabase) *slowBackupFile {
	t.Helper()

	backupFilePath := getBackupFilePath(t, tm, database.backup.Id)
	backupContent, err := os.ReadFile(backupFilePath)
	require.NoError(t, err)
	require.Greater(t, len(backupContent), 1)

	require.NoError(t, os.Remove(backupFilePath))
	require.NoError(t, syscall.Mkfifo(backupFilePath, 0o600))

	// Opened for reading and writing, the pipe opens at once and never reports the end of the file to
	// its reader while the test holds it.
	pipe, err := os.OpenFile(backupFilePath, os.O_RDWR, 0)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pipe.Close() })

	halfLength := len(backupContent) / 2
	_, err = pipe.Write(backupContent[:halfLength])
	require.NoError(t, err)

	return &slowBackupFile{pipe: pipe, servedBytes: backupContent[:halfLength], restBytes: backupContent[halfLength:]}
}

func (f *slowBackupFile) serveRest(t *testing.T) {
	t.Helper()

	_, err := f.pipe.Write(f.restBytes)
	require.NoError(t, err)
}

type archiveCopySpec struct {
	team              team
	archiveDirsBefore []string
	servedBytes       []byte
}

func waitForArchiveCopy(t *testing.T, spec archiveCopySpec) string {
	t.Helper()

	var archiveDir string

	testwait.Until(t, "the restore copies the served part of the archive", func(context.Context) error {
		for _, listedDir := range listRestoreArchiveDirs(t, spec.team) {
			if slices.Contains(spec.archiveDirsBefore, listedDir) {
				continue
			}

			archiveInfo, err := os.Stat(filepath.Join(listedDir, "backup.dump"))
			if err != nil {
				return fmt.Errorf("%w: %w", errRestoreArchiveNotCopied, err)
			}

			if archiveInfo.Size() != int64(len(spec.servedBytes)) {
				return fmt.Errorf(
					"%w: %d of %d bytes",
					errRestoreArchiveNotCopied,
					archiveInfo.Size(),
					len(spec.servedBytes),
				)
			}

			archiveDir = listedDir

			return nil
		}

		return errRestoreArchiveNotCopied
	})

	return archiveDir
}

// hasRestoreToolStartRecord tells whether the log records a start of pg_restore into the database. The
// restore logs every start of the tool with its arguments, the target database among them.
func hasRestoreToolStartRecord(t *testing.T, tm team, databaseName string) bool {
	t.Helper()

	for line := range strings.SplitSeq(readApplicationLogs(t, tm), "\n") {
		if strings.Contains(line, "executing pg_restore command") && strings.Contains(line, databaseName) {
			return true
		}
	}

	return false
}
