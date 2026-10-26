package usecases_postgresql

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	restores_core "databasus-backend/internal/features/restores/core"
	io_utils "databasus-backend/internal/util/io"
)

const pgRestoreStubError = `pg_restore: error: could not execute query: ERROR:  relation "orders" already exists`

func newStubPgRestoreCommand(pgBin string) pgRestoreCommand {
	return pgRestoreCommand{
		source:      restores_core.SourceDatabase{},
		pgBin:       pgBin,
		credentials: &pgshared.CredentialTempFiles{},
		pgConfig:    &restores_core.PostgresqlTargetDatabase{},
		logger:      slog.New(slog.DiscardHandler),
	}
}

func newStubRestoreUsecase() *RestorePostgresqlBackupUsecase {
	return &RestorePostgresqlBackupUsecase{
		slog.New(slog.DiscardHandler),
		nil,
	}
}

func Test_ExecutePgRestore_WhenPgRestoreExitsRightAfterError_ReportsPgRestoreError(t *testing.T) {
	restoreErr := newStubRestoreUsecase().executePgRestore(
		t.Context(),
		newStubPgRestoreCommand(restores_core.WriteLateFailingToolStub(t, "pg_restore", pgRestoreStubError)),
	)

	require.Error(t, restoreErr)
	assert.Contains(t, restoreErr.Error(), `relation "orders" already exists`)
}

func Test_ExecutePgRestoreViaStdin_WhenPgRestoreExitsRightAfterError_ReportsPgRestoreError(
	t *testing.T,
) {
	backupReader := bytes.NewReader([]byte("PGDMP"))

	restoreErr := newStubRestoreUsecase().executePgRestoreViaStdin(
		t.Context(),
		newStubPgRestoreCommand(restores_core.WriteLateFailingToolStub(t, "pg_restore", pgRestoreStubError)),
		backupReader,
		io_utils.NewFailureTrackingReader(backupReader),
	)

	require.Error(t, restoreErr)
	assert.NotContains(t, restoreErr.Error(), "backup stream from storage failed mid-restore")
	assert.Contains(t, restoreErr.Error(), `relation "orders" already exists`)
}

func Test_ExecutePgRestoreViaStdin_WhenPgRestoreExitsBeforeReadingBackup_ReportsPgRestoreError(
	t *testing.T,
) {
	// The backup is far larger than a pipe buffer, so pg_restore exits while most of it is
	// still waiting to be written to its standard input.
	backupReader := bytes.NewReader(bytes.Repeat([]byte("PGDMP"), 4*1024*1024))
	pgRestoreStub := restores_core.WriteToolStub(
		t,
		"pg_restore",
		"#!/bin/sh\necho '"+pgRestoreStubError+"' >&2\nexit 1\n",
	)

	restoreErr := newStubRestoreUsecase().executePgRestoreViaStdin(
		t.Context(),
		newStubPgRestoreCommand(pgRestoreStub),
		backupReader,
		io_utils.NewFailureTrackingReader(backupReader),
	)

	require.Error(t, restoreErr)
	assert.NotContains(t, restoreErr.Error(), "failed to stream backup data to pg_restore")
	assert.Contains(t, restoreErr.Error(), `relation "orders" already exists`)
}
