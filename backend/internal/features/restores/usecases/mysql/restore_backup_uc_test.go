package usecases_mysql

import (
	"bytes"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	restores_core "databasus-backend/internal/features/restores/core"
)

func writeFailingMysqlClientStub(t *testing.T) string {
	t.Helper()

	return restores_core.WriteLateFailingToolStub(
		t, "mysql-stub", "ERROR 1064 (42000) at line 53465: You have an error in your SQL syntax",
	)
}

// A truncated stream makes the client fail on a half-written statement, and that syntax error must
// not mask the stream failure.
func Test_ExecuteMysqlRestore_WhenBackupStreamTruncates_ReportsStreamFailureNotSyntaxError(
	t *testing.T,
) {
	dumpStatements := bytes.Repeat(
		[]byte("INSERT INTO orders VALUES ('2020-07-13 18:49:41');\n"),
		200_000,
	)
	compressedDump := restores_core.CompressToZstd(t, dumpStatements)
	truncatedDump := compressedDump[:len(compressedDump)*3/4]

	restoreUsecase := &RestoreMysqlBackupUsecase{
		slog.New(slog.DiscardHandler),
		nil,
	}

	restoreErr := restoreUsecase.executeMysqlRestore(
		t.Context(),
		restores_core.SourceDatabase{},
		writeFailingMysqlClientStub(t),
		nil,
		filepath.Join(t.TempDir(), ".my.cnf"),
		io.NopCloser(bytes.NewReader(truncatedDump)),
		&backups_core_logical.LogicalBackup{},
	)

	require.Error(t, restoreErr)
	assert.Contains(t, restoreErr.Error(), "backup stream from storage failed mid-restore")
	assert.NotContains(t, restoreErr.Error(), "SQL syntax")
}

func Test_ExecuteMysqlRestore_WhenBackupStreamIsComplete_ReportsClientExitError(t *testing.T) {
	compressedDump := restores_core.CompressToZstd(t, []byte("INSERT INTO orders VALUES (1);\n"))

	restoreUsecase := &RestoreMysqlBackupUsecase{
		slog.New(slog.DiscardHandler),
		nil,
	}

	restoreErr := restoreUsecase.executeMysqlRestore(
		t.Context(),
		restores_core.SourceDatabase{},
		writeFailingMysqlClientStub(t),
		nil,
		filepath.Join(t.TempDir(), ".my.cnf"),
		io.NopCloser(bytes.NewReader(compressedDump)),
		&backups_core_logical.LogicalBackup{},
	)

	require.Error(t, restoreErr)
	assert.NotContains(t, restoreErr.Error(), "backup stream from storage failed mid-restore")
	assert.Contains(t, restoreErr.Error(), "SQL syntax")
}
