package usecases_mariadb

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

func Test_ExecuteMariadbRestore_WhenClientExitsRightAfterError_ReportsClientError(t *testing.T) {
	compressedDump := restores_core.CompressToZstd(t, []byte("INSERT INTO orders VALUES (1);\n"))

	restoreUsecase := &RestoreMariadbBackupUsecase{
		slog.New(slog.DiscardHandler),
		nil,
	}

	restoreErr := restoreUsecase.executeMariadbRestore(
		t.Context(),
		restores_core.SourceDatabase{},
		restores_core.WriteLateFailingToolStub(
			t, "mariadb-stub", "ERROR 1064 (42000) at line 1: You have an error in your SQL syntax",
		),
		nil,
		filepath.Join(t.TempDir(), ".my.cnf"),
		io.NopCloser(bytes.NewReader(compressedDump)),
		&backups_core_logical.LogicalBackup{},
	)

	require.Error(t, restoreErr)
	assert.NotContains(t, restoreErr.Error(), "backup stream from storage failed mid-restore")
	assert.Contains(t, restoreErr.Error(), "SQL syntax")
}
