package logical

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"databasus-backend/internal/engines/postgresql/logical/pgrestore"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
)

type RestoreSpec struct {
	NewCleanupContext  func(context.Context) (context.Context, context.CancelFunc)
	ClientTools        pgshared.ClientTools
	Version            pgshared.Version
	SourceDatabaseName string
	// ParallelJobs above 1 restores with pg_restore -j, capped at 8.
	ParallelJobs        int
	IsTimescaleDB       bool
	IsRestoreOwnership  bool
	IsRestorePrivileges bool
	IsExcludeExtensions bool
	IsSkipUserMappings  bool
	// TempDir holds the archive of a restore that needs it as a file.
	TempDir string
	Logger  *slog.Logger
}

// NeedsArchiveFile reports whether pg_restore must read the archive more than once: -j seeks in it,
// and a filtered restore lists it with -l before restoring the filtered list with -L.
func (spec RestoreSpec) NeedsArchiveFile() bool {
	return spec.IsExcludeExtensions || spec.IsSkipUserMappings || spec.ParallelJobs > 1
}

// Restore restores the pg_dump custom-format archive read from input into the source's database.
// It streams input into pg_restore, or first writes it into a file in spec.TempDir when the restore
// needs the archive as a file. A failure whose only error is the existing public schema counts as
// success.
func (s *Source) Restore(ctx context.Context, spec RestoreSpec, input io.Reader) error {
	if s.settings.Database == "" {
		return ErrRestoreDatabaseNameRequired
	}

	if spec.IsTimescaleDB {
		return s.restoreWithTimescaleDBHooks(ctx, spec, input)
	}

	return s.restoreArchive(ctx, spec, input)
}

func (s *Source) restoreArchive(ctx context.Context, spec RestoreSpec, input io.Reader) error {
	credentials, err := s.writeCredentialFiles()
	if err != nil {
		return fmt.Errorf("failed to create credential files: %w", err)
	}
	defer credentials.Remove()

	pgRestore := spec.ClientTools.GetExecutable(spec.Version, pgshared.ExecutablePgRestore)

	if !spec.NeedsArchiveFile() {
		spec.Logger.InfoContext(ctx, "restoring via stdin streaming (CPU=1)")

		return s.runPgRestore(ctx, spec, clientToolRun{
			path:        pgRestore,
			args:        s.buildRestoreArgs(spec, restoreArchivePaths{}),
			credentials: credentials,
			stdin:       input,
			stdout:      nil,
			stderr:      nil,
		})
	}

	spec.Logger.InfoContext(ctx, "restoring via file with parallel jobs",
		slog.Int("parallel_jobs", pgrestore.GetParallelJobs(spec.ParallelJobs, spec.IsTimescaleDB)),
		slog.Bool("is_exclude_extensions", spec.IsExcludeExtensions),
		slog.Bool("is_skip_user_mappings", spec.IsSkipUserMappings),
	)

	tempDir, err := os.MkdirTemp(spec.TempDir, "restore_*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	archivePath := filepath.Join(tempDir, "backup.dump")

	err = writeArchiveFile(ctx, archivePath, input)
	if err != nil {
		return fmt.Errorf("failed to download backup to temporary file: %w", err)
	}

	spec.Logger.InfoContext(ctx, "backup file written to temporary location", slog.String("temp_file", archivePath))

	paths := restoreArchivePaths{archive: archivePath}

	if spec.IsExcludeExtensions || spec.IsSkipUserMappings {
		tocListPath, err := s.writeFilteredTocList(ctx, spec, tocListRun{
			pgRestore:   pgRestore,
			archivePath: archivePath,
			tempDir:     tempDir,
			credentials: credentials,
		})
		if err != nil {
			return fmt.Errorf("failed to generate filtered TOC list: %w", err)
		}

		paths.tocList = tocListPath
	}

	return s.runPgRestore(ctx, spec, clientToolRun{
		path:        pgRestore,
		args:        s.buildRestoreArgs(spec, paths),
		credentials: credentials,
		stdin:       nil,
		stdout:      nil,
		stderr:      nil,
	})
}

func (s *Source) runPgRestore(ctx context.Context, spec RestoreSpec, run clientToolRun) error {
	spec.Logger.InfoContext(
		ctx,
		"executing pg_restore command",
		slog.String("pg_bin", run.path),
		slog.Any("args", run.args),
	)

	var stderr bytes.Buffer
	run.stderr = &stderr

	err := s.runClientTool(ctx, run)
	if err == nil {
		return nil
	}

	// A missing extension in a real restore means missing data, so only the existing public schema is ignorable.
	if pgrestore.HasOnlyIgnorableErrors(stderr.String(), pgrestore.PreexistingPublicSchema) {
		spec.Logger.WarnContext(ctx,
			"pg_restore ignored a pre-existing public schema; treating the restore as successful",
		)

		return nil
	}

	return classifyRestoreFailure(clientToolFailure{
		toolPath: run.path,
		args:     run.args,
		runErr:   err,
		stderr:   stderr.String(),
		logger:   spec.Logger,
	}, restoreDatabaseNames{source: spec.SourceDatabaseName, target: s.settings.Database})
}

type restoreArchivePaths struct {
	archive string
	tocList string
}

func (s *Source) buildRestoreArgs(spec RestoreSpec, paths restoreArchivePaths) []string {
	return pgrestore.BuildArgs(pgrestore.ArgsSpec{
		Host:                s.settings.Host,
		Port:                s.settings.Port,
		User:                s.settings.Username,
		Database:            s.settings.Database,
		ParallelJobs:        pgrestore.GetParallelJobs(spec.ParallelJobs, spec.IsTimescaleDB),
		IsTimescaleDB:       spec.IsTimescaleDB,
		IsRestoreOwnership:  spec.IsRestoreOwnership,
		IsRestorePrivileges: spec.IsRestorePrivileges,
		IsVerbose:           true,
		TocListPath:         paths.tocList,
		ArchivePath:         paths.archive,
	})
}

func writeArchiveFile(ctx context.Context, archivePath string, input io.Reader) error {
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create temporary backup file: %w", err)
	}

	_, err = io.Copy(archiveFile, contextCheckingReader{ctx: ctx, reader: input})

	closeErr := archiveFile.Close()
	if err != nil {
		return fmt.Errorf("failed to write backup to temporary file: %w", err)
	}

	if closeErr != nil {
		return fmt.Errorf("failed to write backup to temporary file: %w", closeErr)
	}

	return nil
}

// contextCheckingReader stops a copy at the next read once ctx ended. A read blocked inside the
// reader ends with the reader's own context.
type contextCheckingReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextCheckingReader) Read(p []byte) (int, error) {
	err := context.Cause(r.ctx)
	if err != nil {
		return 0, fmt.Errorf("copy cancelled: %w", err)
	}

	return r.reader.Read(p)
}
