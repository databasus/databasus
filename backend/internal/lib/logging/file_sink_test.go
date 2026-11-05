package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func Test_RotatingFileWriter_OnShutdown_DrainsQueuedLinesToDisk(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "logs", "databasus.log")

	writer, err := newRotatingFileWriter(rotatingFileSpec{path: logPath, maxSizeMB: 5, maxBackups: 3})
	require.NoError(t, err)

	fileLogger := slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: slog.LevelDebug}))
	fileLogger.InfoContext(t.Context(), "backup finished", slog.String("database_id", "db-42"))

	require.NoError(t, writer.Shutdown(t.Context()))

	writtenLog, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(writtenLog), "backup finished")
	assert.Contains(t, string(writtenLog), "db-42")
}

func Test_RotatingFileWriter_WhenSizeExceeded_KeepsAtMostMaxBackupsPlusActive(t *testing.T) {
	logDirectory := t.TempDir()
	logPath := filepath.Join(logDirectory, "databasus.log")

	writer, err := newRotatingFileWriter(rotatingFileSpec{path: logPath, maxSizeMB: 1, maxBackups: 3})
	require.NoError(t, err)

	line := append([]byte(strings.Repeat("x", 8*1024)), '\n')
	for range 700 {
		_, writeErr := writer.Write(line)
		require.NoError(t, writeErr)
	}

	require.NoError(t, writer.Shutdown(t.Context()))

	globLogFiles := func() []string {
		logFiles, globErr := filepath.Glob(filepath.Join(logDirectory, "databasus*.log"))
		require.NoError(t, globErr)

		return logFiles
	}

	// lumberjack deletes surplus backups in a background goroutine that Close does not wait for,
	// so the cap holds only once that cleanup has caught up with the last rotation.
	testwait.Until(t, "the active file plus at most 3 backups", func(context.Context) error {
		logFileCount := len(globLogFiles())
		if logFileCount > 4 {
			return fmt.Errorf("%d log files remain", logFileCount)
		}

		return nil
	})

	logFiles := globLogFiles()
	// Both bounds matter: >1 proves rotation happened at all, <=4 proves it was capped. Asserting
	// only the upper bound would pass if rotation never ran.
	assert.Greater(t, len(logFiles), 1, "5.6 MB through a 1 MB limit must rotate")

	for _, logFile := range logFiles {
		info, statErr := os.Stat(logFile)
		require.NoError(t, statErr)
		assert.LessOrEqual(t, info.Size(), int64(2*1024*1024), "file %s", logFile)
	}
}

func Test_RotatingFileWriter_AfterShutdown_WriteDoesNotBlockOrPanic(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "databasus.log")

	writer, err := newRotatingFileWriter(rotatingFileSpec{path: logPath, maxSizeMB: 5, maxBackups: 3})
	require.NoError(t, err)
	require.NoError(t, writer.Shutdown(t.Context()))

	written, err := writer.Write([]byte("after shutdown\n"))

	require.NoError(t, err)
	assert.Equal(t, len("after shutdown\n"), written)
}

func Test_NewRotatingFileWriter_WhenDirectoryIsUnwritable_ReturnsError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}

	blockedRoot := filepath.Join(t.TempDir(), "blocked")
	require.NoError(t, os.Mkdir(blockedRoot, 0o500))

	_, err := newRotatingFileWriter(rotatingFileSpec{
		path:       filepath.Join(blockedRoot, "nested", "databasus.log"),
		maxSizeMB:  5,
		maxBackups: 3,
	})

	require.Error(t, err)
}
