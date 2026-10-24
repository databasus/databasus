package usecases_postgresql

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/features/backups/backups/encryption"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	pgtypes "databasus-backend/internal/features/databases/databases/postgresql/logical"
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	util_encryption "databasus-backend/internal/util/encryption"
	io_utils "databasus-backend/internal/util/io"
	"databasus-backend/internal/util/tools"
)

type RestorePostgresqlBackupUsecase struct {
	logger           *slog.Logger
	secretKeyService *encryption_secrets.SecretKeyService
}

func (uc *RestorePostgresqlBackupUsecase) Execute(
	parentCtx context.Context,
	originalDB *databases.Database,
	restoringToDB *databases.Database,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	restore restores_core.Restore,
	backup *backups_core_logical.LogicalBackup,
	storage *storages.Storage,
	options restores_core.RestoreOptions,
) error {
	if originalDB.Type != databases.DatabaseTypePostgresLogical {
		return errors.New("database type not supported")
	}

	uc.logger.InfoContext(parentCtx,
		"restoring PostgreSQL backup via pg_restore",
		"restore_id",
		restore.ID,
		"backup_id",
		backup.ID,
	)

	pg := restoringToDB.PostgresqlLogical
	if pg == nil {
		return fmt.Errorf("postgresql configuration is required for restore")
	}

	if pg.Database == nil || *pg.Database == "" {
		return fmt.Errorf("target database name is required for pg_restore")
	}

	pgBin := tools.GetPostgresqlExecutable(pg.Version, "pg_restore")

	// All PostgreSQL backups are now custom format (-Fc)
	return uc.restoreCustomType(
		parentCtx,
		originalDB,
		pgBin,
		backup,
		storage,
		pg,
		options,
	)
}

// restoreCustomType restores a backup in custom type (-Fc)
func (uc *RestorePostgresqlBackupUsecase) restoreCustomType(
	parentCtx context.Context,
	originalDB *databases.Database,
	pgBin string,
	backup *backups_core_logical.LogicalBackup,
	storage *storages.Storage,
	pg *pgtypes.PostgresqlLogicalDatabase,
	options restores_core.RestoreOptions,
) error {
	uc.logger.InfoContext(parentCtx,
		"restoring backup in custom type (-Fc)",
		"backup_id",
		backup.ID,
		"cpuCount",
		pg.CpuCount,
	)

	// File-based restore for parallel jobs (multiple CPUs) or any TOC filtering (extension exclusion
	// or skipping user mappings needs a TOC file); otherwise stream directly via stdin. A timescaledb
	// backup uses whichever path applies — the pre/post hooks wrap it the same way (they only set a
	// database-level GUC, so streaming is kept).
	runRestore := func() error {
		if options.IsExcludeExtensions || options.IsSkipUserMappings || pg.CpuCount > 1 {
			return uc.restoreViaFile(parentCtx, originalDB, pgBin, backup, storage, pg, options)
		}

		return uc.restoreViaStdin(parentCtx, originalDB, pgBin, backup, storage, pg)
	}

	if backup.TimescaledbVersion != "" {
		return uc.withTimescaleHooks(parentCtx, pg, runRestore)
	}

	return runRestore()
}

// withTimescaleHooks wraps a restore in the TimescaleDB procedure: ensure the extension exists and
// enter restoring mode before pg_restore, then leave restoring mode after — unconditionally, so a
// failed restore never leaves the target stuck in restoring mode. See timescaledb.go for why the
// database-level GUC set here is inherited by the separate pg_restore process.
func (uc *RestorePostgresqlBackupUsecase) withTimescaleHooks(
	ctx context.Context,
	pg *pgtypes.PostgresqlLogicalDatabase,
	runRestore func() error,
) (err error) {
	encryptor := util_encryption.GetFieldEncryptor()

	if err := pg.RunTimescaleDBPreRestore(ctx, encryptor); err != nil {
		return fmt.Errorf("timescaledb_pre_restore failed: %w", err)
	}

	uc.logger.InfoContext(ctx, "entered timescaledb restoring mode")

	defer func() {
		if postErr := pg.RunTimescaleDBPostRestore(ctx, encryptor); postErr != nil {
			uc.logger.ErrorContext(
				ctx,
				"timescaledb_post_restore failed; target database may be left in restoring mode",
				"error", postErr,
			)

			if err == nil {
				err = fmt.Errorf("timescaledb_post_restore failed: %w", postErr)
			}
		}
	}()

	return runRestore()
}

func (uc *RestorePostgresqlBackupUsecase) restoreViaStdin(
	parentCtx context.Context,
	originalDB *databases.Database,
	pgBin string,
	backup *backups_core_logical.LogicalBackup,
	storage *storages.Storage,
	pg *pgtypes.PostgresqlLogicalDatabase,
) error {
	logger := uc.logger.With("backup_id", backup.ID)

	logger.InfoContext(parentCtx, "restoring via stdin streaming (CPU=1)")

	args := []string{
		"-Fc", // expect custom type
		"--no-password",
		"-h", pg.Host,
		"-p", strconv.Itoa(pg.Port),
		"-U", pg.Username,
		"-d", *pg.Database,
		"--verbose",
	}
	// --clean would DROP EXTENSION timescaledb, taking the catalog tables that pre_restore needs
	// with it; TimescaleDB restores into a clean target without it. Non-timescaledb keeps --clean.
	if backup.TimescaledbVersion == "" {
		args = append(args, "--clean", "--if-exists")
	}
	if !pg.IsRestoreOwnership {
		args = append(args, "--no-owner")
	}
	if !pg.IsRestorePrivileges {
		args = append(args, "--no-acl")
	}

	ctx, cancel := context.WithTimeout(parentCtx, 23*time.Hour)
	defer cancel()

	// Materialize connection credentials (.pgpass + optional client certificates)
	fieldEncryptor := util_encryption.GetFieldEncryptor()
	decryptedPassword, err := fieldEncryptor.Decrypt(pg.Password)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	credentials, err := postgresql_shared.WriteCredentialFilesToTempDir(
		pg.CredentialSpec(), decryptedPassword, fieldEncryptor)
	if err != nil {
		return fmt.Errorf("failed to create credential files: %w", err)
	}
	defer credentials.Remove()

	rawReader, err := storage.GetFile(ctx, fieldEncryptor, logger, backup.FileName)
	if err != nil {
		return fmt.Errorf("failed to get backup file from storage: %w", err)
	}
	defer func() {
		if err := rawReader.Close(); err != nil {
			logger.ErrorContext(parentCtx, "failed to close backup reader", "error", err)
		}
	}()

	storageReadFailureTracker := io_utils.NewFailureTrackingReader(rawReader)

	var backupReader io.Reader = storageReadFailureTracker
	if backup.Encryption == backups_core_enums.BackupEncryptionEncrypted {
		if backup.EncryptionSalt == nil || backup.EncryptionIV == nil {
			return fmt.Errorf("backup is encrypted but missing encryption metadata")
		}

		masterKey, err := uc.secretKeyService.GetSecretKey()
		if err != nil {
			return fmt.Errorf("failed to get master key for decryption: %w", err)
		}

		salt, err := base64.StdEncoding.DecodeString(*backup.EncryptionSalt)
		if err != nil {
			return fmt.Errorf("failed to decode encryption salt: %w", err)
		}

		iv, err := base64.StdEncoding.DecodeString(*backup.EncryptionIV)
		if err != nil {
			return fmt.Errorf("failed to decode encryption IV: %w", err)
		}

		decryptReader, err := encryption.NewDecryptionReader(
			storageReadFailureTracker,
			masterKey,
			backup.ID,
			salt,
			iv,
		)
		if err != nil {
			return fmt.Errorf("failed to create decryption reader: %w", err)
		}

		backupReader = decryptReader
		logger.InfoContext(parentCtx, "using decryption for encrypted backup")
	}

	return uc.executePgRestoreViaStdin(ctx, pgRestoreCommand{
		database:    originalDB,
		pgBin:       pgBin,
		args:        args,
		credentials: credentials,
		pgConfig:    pg,
		logger:      logger,
	}, backupReader, storageReadFailureTracker)
}

// executePgRestoreViaStdin pumps the decoded backup into pg_restore's standard input.
func (uc *RestorePostgresqlBackupUsecase) executePgRestoreViaStdin(
	ctx context.Context,
	command pgRestoreCommand,
	backupReader io.Reader,
	storageReadFailureTracker *io_utils.FailureTrackingReader,
) error {
	pgBin := command.pgBin

	// The binary is a database client from the bundled tools directory, and the arguments go to it as a list, not through a shell.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, pgBin, command.args...)
	command.logger.InfoContext(ctx, "executing PostgreSQL restore command via stdin", "command", cmd.String())

	// Setup environment variables
	uc.setupPgRestoreEnvironment(cmd, command.credentials, command.pgConfig)

	// Verify executable exists and is accessible
	if _, err := exec.LookPath(pgBin); err != nil {
		return fmt.Errorf(
			"PostgreSQL executable not found or not accessible: %s - %w",
			pgBin,
			err,
		)
	}

	// Create stdin pipe for explicit data pumping
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	var stderrOutput bytes.Buffer
	cmd.Stderr = &stderrOutput
	cmd.WaitDelay = restores_core.ToolOutputWaitDelay

	// Start pg_restore
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", filepath.Base(pgBin), err)
	}

	decodedStreamFailureTracker := io_utils.NewFailureTrackingReader(backupReader)

	copyErrCh := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(stdinPipe, decodedStreamFailureTracker)
		// Close stdin pipe to signal EOF to pg_restore - critical for proper termination
		closeErr := stdinPipe.Close()
		switch {
		case copyErr != nil:
			copyErrCh <- fmt.Errorf("copy to stdin: %w", copyErr)
		case closeErr != nil:
			copyErrCh <- fmt.Errorf("close stdin: %w", closeErr)
		default:
			copyErrCh <- nil
		}
	}()

	// Wait for the restore to finish
	waitErr := cmd.Wait()
	copyErr := <-copyErrCh

	// Check for cancellation
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.Canceled) {
			return fmt.Errorf("restore cancelled")
		}
	default:
	}

	if streamErr := restores_core.GetBackupStreamFailure(restores_core.BackupStreamTrackers{
		StorageRead:   storageReadFailureTracker,
		DecodedStream: decodedStreamFailureTracker,
	}); streamErr != nil {
		return streamErr
	}

	if waitErr != nil {
		// Check for cancellation again
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				return fmt.Errorf("restore cancelled")
			}
		default:
		}

		return uc.resolvePgRestoreOutcome(pgRestoreFailure{
			database:     command.database,
			waitErr:      waitErr,
			stderrOutput: stderrOutput.Bytes(),
			pgBin:        command.pgBin,
			args:         command.args,
			pgConfig:     command.pgConfig,
			logger:       command.logger,
		})
	}

	// pg_restore exiting early breaks the pipe, so its own exit error is the cause and the copy
	// error counts only when pg_restore succeeded.
	if copyErr != nil {
		return fmt.Errorf("failed to stream backup data to pg_restore: %w", copyErr)
	}

	return nil
}

// restoreViaFile downloads backup and uses parallel jobs for multi-CPU restore
func (uc *RestorePostgresqlBackupUsecase) restoreViaFile(
	parentCtx context.Context,
	originalDB *databases.Database,
	pgBin string,
	backup *backups_core_logical.LogicalBackup,
	storage *storages.Storage,
	pg *pgtypes.PostgresqlLogicalDatabase,
	options restores_core.RestoreOptions,
) error {
	uc.logger.InfoContext(parentCtx,
		"restoring via file with parallel jobs",
		"backup_id",
		backup.ID,
		"cpuCount",
		pg.CpuCount,
	)

	isTimescale := backup.TimescaledbVersion != ""

	// TimescaleDB restores single-threaded: with -j the parallel loop loads dependent
	// _timescaledb_catalog rows (chunk_constraint, chunk_index) before the chunk rows they reference,
	// tripping the catalog's own foreign keys. Otherwise cap between 1 and 8.
	parallelJobs := max(1, min(pg.CpuCount, 8))
	if isTimescale {
		parallelJobs = 1
	}

	args := []string{
		"-Fc",                            // expect custom type
		"-j", strconv.Itoa(parallelJobs), // parallel jobs based on CPU count
		"--no-password",
		"-h", pg.Host,
		"-p", strconv.Itoa(pg.Port),
		"-U", pg.Username,
		"-d", *pg.Database,
		"--verbose",
	}
	// --clean would DROP EXTENSION timescaledb, taking the catalog tables that pre_restore needs
	// with it; TimescaleDB restores into a clean target without it. Non-timescaledb keeps --clean.
	if !isTimescale {
		args = append(args, "--clean", "--if-exists")
	}
	if !pg.IsRestoreOwnership {
		args = append(args, "--no-owner")
	}
	if !pg.IsRestorePrivileges {
		args = append(args, "--no-acl")
	}

	return uc.restoreFromStorage(
		parentCtx,
		originalDB,
		pgBin,
		args,
		pg.Password,
		backup,
		storage,
		pg,
		options,
	)
}

// restoreFromStorage restores backup data from storage using pg_restore
func (uc *RestorePostgresqlBackupUsecase) restoreFromStorage(
	parentCtx context.Context,
	database *databases.Database,
	pgBin string,
	args []string,
	password string,
	backup *backups_core_logical.LogicalBackup,
	storage *storages.Storage,
	pgConfig *pgtypes.PostgresqlLogicalDatabase,
	options restores_core.RestoreOptions,
) error {
	logger := uc.logger.With("backup_id", backup.ID)

	logger.InfoContext(parentCtx,
		"restoring backup from storage via temporary file",
		"pg_bin",
		pgBin,
		"args",
		args,
		"is_exclude_extensions",
		options.IsExcludeExtensions,
		"is_skip_user_mappings",
		options.IsSkipUserMappings,
	)

	ctx, cancel := context.WithTimeout(parentCtx, 23*time.Hour)
	defer cancel()

	// Materialize connection credentials (.pgpass + optional client certificates)
	credentials, err := postgresql_shared.WriteCredentialFilesToTempDir(
		pgConfig.CredentialSpec(),
		password,
		util_encryption.GetFieldEncryptor(),
	)
	if err != nil {
		return fmt.Errorf("failed to create credential files: %w", err)
	}
	defer credentials.Remove()

	// Download backup to temporary file
	tempBackupFile, cleanupFunc, err := uc.downloadBackupToTempFile(ctx, backup, storage)
	if err != nil {
		return fmt.Errorf("failed to download backup to temporary file: %w", err)
	}
	defer cleanupFunc()

	if options.IsExcludeExtensions || options.IsSkipUserMappings {
		tocListFile, err := uc.generateFilteredTocList(
			ctx,
			pgBin,
			tempBackupFile,
			credentials,
			pgConfig,
			options,
		)
		if err != nil {
			return fmt.Errorf("failed to generate filtered TOC list: %w", err)
		}
		defer func() {
			_ = os.Remove(tocListFile)
		}()

		// Add -L flag to use the filtered list
		args = append(args, "-L", tocListFile)
	}

	// Add the temporary backup file as the last argument to pg_restore
	args = append(args, tempBackupFile)

	return uc.executePgRestore(ctx, pgRestoreCommand{
		database:    database,
		pgBin:       pgBin,
		args:        args,
		credentials: credentials,
		pgConfig:    pgConfig,
		logger:      logger,
	})
}

func (uc *RestorePostgresqlBackupUsecase) downloadBackupToTempFile(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	storage *storages.Storage,
) (string, func(), error) {
	// Create temporary directory for backup data
	tempDir, err := os.MkdirTemp(config.GetEnv().TempFolder, "restore_"+uuid.New().String())
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	logger := uc.logger.With("backup_id", backup.ID)

	cleanupFunc := func() {
		_ = os.RemoveAll(tempDir)
	}

	tempBackupFile := filepath.Join(tempDir, "backup.dump")

	logger.InfoContext(
		ctx,
		"downloading backup file from storage to temporary file",
		"temp_file",
		tempBackupFile,
		"encrypted",
		backup.Encryption == backups_core_enums.BackupEncryptionEncrypted,
	)

	fieldEncryptor := util_encryption.GetFieldEncryptor()
	rawReader, err := storage.GetFile(ctx, fieldEncryptor, logger, backup.FileName)
	if err != nil {
		cleanupFunc()
		return "", nil, fmt.Errorf("failed to get backup file from storage: %w", err)
	}

	defer func() {
		if err := rawReader.Close(); err != nil {
			logger.ErrorContext(ctx, "failed to close backup reader", "error", err)
		}
	}()

	storageReadFailureTracker := io_utils.NewFailureTrackingReader(rawReader)

	var backupReader io.Reader = storageReadFailureTracker
	if backup.Encryption == backups_core_enums.BackupEncryptionEncrypted {
		if backup.EncryptionSalt == nil || backup.EncryptionIV == nil {
			cleanupFunc()
			return "", nil, fmt.Errorf("backup is encrypted but missing encryption metadata")
		}

		masterKey, err := uc.secretKeyService.GetSecretKey()
		if err != nil {
			cleanupFunc()
			return "", nil, fmt.Errorf("failed to get master key for decryption: %w", err)
		}

		salt, err := base64.StdEncoding.DecodeString(*backup.EncryptionSalt)
		if err != nil {
			cleanupFunc()
			return "", nil, fmt.Errorf("failed to decode encryption salt: %w", err)
		}

		iv, err := base64.StdEncoding.DecodeString(*backup.EncryptionIV)
		if err != nil {
			cleanupFunc()
			return "", nil, fmt.Errorf("failed to decode encryption IV: %w", err)
		}

		decryptReader, err := encryption.NewDecryptionReader(
			storageReadFailureTracker,
			masterKey,
			backup.ID,
			salt,
			iv,
		)
		if err != nil {
			cleanupFunc()
			return "", nil, fmt.Errorf("failed to create decryption reader: %w", err)
		}

		backupReader = decryptReader
		logger.InfoContext(ctx, "using decryption for encrypted backup")
	}

	// Create temporary backup file
	tempFile, err := os.Create(tempBackupFile)
	if err != nil {
		cleanupFunc()
		return "", nil, fmt.Errorf("failed to create temporary backup file: %w", err)
	}
	defer func() {
		if err := tempFile.Close(); err != nil {
			logger.ErrorContext(ctx, "failed to close temporary file", "error", err)
		}
	}()

	// Copy backup data to temporary file with shutdown checks
	_, err = uc.copyWithShutdownCheck(ctx, tempFile, backupReader)
	if err != nil {
		cleanupFunc()
		return "", nil, fmt.Errorf("failed to write backup to temporary file: %w", err)
	}

	logger.InfoContext(ctx, "backup file written to temporary location", "temp_file", tempBackupFile)

	return tempBackupFile, cleanupFunc, nil
}

type pgRestoreCommand struct {
	database    *databases.Database
	pgBin       string
	args        []string
	credentials *postgresql_shared.CredentialTempFiles
	pgConfig    *pgtypes.PostgresqlLogicalDatabase
	logger      *slog.Logger
}

func (uc *RestorePostgresqlBackupUsecase) executePgRestore(
	ctx context.Context,
	command pgRestoreCommand,
) error {
	pgBin := command.pgBin

	// The binary is a database client from the bundled tools directory, and the arguments go to it as a list, not through a shell.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, pgBin, command.args...)
	command.logger.InfoContext(ctx, "executing pg_restore command", "command", cmd.String())

	// Setup environment variables
	uc.setupPgRestoreEnvironment(cmd, command.credentials, command.pgConfig)

	// Verify executable exists and is accessible
	if _, err := exec.LookPath(pgBin); err != nil {
		return fmt.Errorf(
			"PostgreSQL executable not found or not accessible: %s - %w",
			pgBin,
			err,
		)
	}

	var stderrOutput bytes.Buffer
	cmd.Stderr = &stderrOutput
	cmd.WaitDelay = restores_core.ToolOutputWaitDelay

	// Start pg_restore
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", filepath.Base(pgBin), err)
	}

	// Wait for the restore to finish
	waitErr := cmd.Wait()

	// Check for cancellation
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.Canceled) {
			return fmt.Errorf("restore cancelled")
		}
	default:
	}

	if waitErr != nil {
		// Check for cancellation again
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				return fmt.Errorf("restore cancelled")
			}
		default:
		}

		return uc.resolvePgRestoreOutcome(pgRestoreFailure{
			database:     command.database,
			waitErr:      waitErr,
			stderrOutput: stderrOutput.Bytes(),
			pgBin:        command.pgBin,
			args:         command.args,
			pgConfig:     command.pgConfig,
			logger:       command.logger,
		})
	}

	return nil
}

func (uc *RestorePostgresqlBackupUsecase) setupPgRestoreEnvironment(
	cmd *exec.Cmd,
	credentials *postgresql_shared.CredentialTempFiles,
	pgConfig *pgtypes.PostgresqlLogicalDatabase,
) {
	cmd.Env = os.Environ()

	cmd.Env = append(cmd.Env, "PGPASSFILE="+credentials.PgpassPath)
	cmd.Env = append(cmd.Env, postgresql_shared.GetPgHostAddrEnv(pgConfig.CredentialSpec())...)
	uc.logger.Info(
		"Using temporary .pgpass file for authentication",
		"pgpass_file", credentials.PgpassPath,
	)

	cmd.Env = append(cmd.Env,
		"PGCLIENTENCODING=UTF8",
		"PGCONNECT_TIMEOUT=30",
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	)

	sslMode := pgConfig.SslMode
	if sslMode == "" {
		sslMode = postgresql_shared.PostgresSslModeDisable
	}

	cmd.Env = append(cmd.Env,
		"PGSSLMODE="+string(sslMode),
		"PGSSLCERT="+credentials.ClientCertPath,
		"PGSSLKEY="+credentials.ClientKeyPath,
		"PGSSLROOTCERT="+credentials.RootCertPath,
		"PGSSLCRL=",
	)
	uc.logger.Info("using SSL mode", "ssl_mode", sslMode)
}

type pgRestoreFailure struct {
	database     *databases.Database
	waitErr      error
	stderrOutput []byte
	pgBin        string
	args         []string
	pgConfig     *pgtypes.PostgresqlLogicalDatabase
	logger       *slog.Logger
}

func (uc *RestorePostgresqlBackupUsecase) resolvePgRestoreOutcome(failure pgRestoreFailure) error {
	pgBin := failure.pgBin
	waitErr := failure.waitErr
	stderr := string(failure.stderrOutput)

	if IsPreexistingPublicSchemaOnly(stderr) {
		failure.logger.Warn(
			"pg_restore ignored a pre-existing public schema; treating the restore as successful",
		)

		return nil
	}

	errorMsg := fmt.Sprintf(
		"%s failed: %v – stderr: %s",
		filepath.Base(pgBin),
		waitErr,
		stderr,
	)

	// Check for specific PostgreSQL error patterns
	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		exitCode := exitErr.ExitCode()

		switch {
		case exitCode == 1 && strings.TrimSpace(stderr) == "":
			errorMsg = fmt.Sprintf(
				"%s failed with exit status 1 but provided no error details. "+
					"This often indicates: "+
					"1) Connection timeout or refused connection, "+
					"2) Authentication failure with incorrect credentials, "+
					"3) Database does not exist, "+
					"4) Network connectivity issues, "+
					"5) PostgreSQL server not running, "+
					"6) Backup file is corrupted or incompatible. "+
					"Command executed: %s %s",
				filepath.Base(pgBin),
				pgBin,
				strings.Join(failure.args, " "),
			)
		case exitCode == -1073741819: // 0xC0000005 in decimal
			errorMsg = fmt.Sprintf(
				"%s crashed with access violation (0xC0000005). This may indicate incompatible PostgreSQL version, corrupted installation, or connection issues. stderr: %s",
				filepath.Base(pgBin),
				stderr,
			)
		case exitCode == 1 || exitCode == 2:
			// Check for common connection and authentication issues
			switch {
			case containsIgnoreCase(stderr, "pg_hba.conf"):
				errorMsg = fmt.Sprintf(
					"PostgreSQL connection rejected by server configuration (pg_hba.conf). stderr: %s",
					stderr,
				)
			case containsIgnoreCase(stderr, "no password supplied") || containsIgnoreCase(stderr, "fe_sendauth"):
				errorMsg = fmt.Sprintf(
					"PostgreSQL authentication failed - no password supplied. stderr: %s",
					stderr,
				)
			case containsIgnoreCase(stderr, "ssl") && containsIgnoreCase(stderr, "connection"):
				errorMsg = fmt.Sprintf(
					"PostgreSQL SSL connection failed. stderr: %s",
					stderr,
				)
			case containsIgnoreCase(stderr, "connection") && containsIgnoreCase(stderr, "refused"):
				errorMsg = fmt.Sprintf(
					"PostgreSQL connection refused. Check if the server is running and accessible. stderr: %s",
					stderr,
				)
			case containsIgnoreCase(stderr, "authentication") || containsIgnoreCase(stderr, "password"):
				errorMsg = fmt.Sprintf(
					"PostgreSQL authentication failed. Check username and password. stderr: %s",
					stderr,
				)
			case containsIgnoreCase(stderr, "timeout"):
				errorMsg = fmt.Sprintf(
					"PostgreSQL connection timeout. stderr: %s",
					stderr,
				)
			case containsIgnoreCase(stderr, "database") && containsIgnoreCase(stderr, "does not exist"):
				sourceDatabase := failure.database.PostgresqlLogical

				backupDbName := "unknown"
				if sourceDatabase != nil && sourceDatabase.Database != nil {
					backupDbName = *sourceDatabase.Database
				}

				targetDbName := "unknown"
				if failure.pgConfig.Database != nil {
					targetDbName = *failure.pgConfig.Database
				}

				errorMsg = fmt.Sprintf(
					"Target database does not exist (backup db %s, not found %s). Create the database before restoring. stderr: %s",
					backupDbName,
					targetDbName,
					stderr,
				)
			}
		}
	}

	return errors.New(errorMsg)
}

// copyWithShutdownCheck copies data from src to dst while checking for shutdown
func (uc *RestorePostgresqlBackupUsecase) copyWithShutdownCheck(
	ctx context.Context,
	dst io.Writer,
	src io.Reader,
) (int64, error) {
	buf := make([]byte, 16*1024*1024) // 16MB buffer
	var totalBytesWritten int64

	for {
		select {
		case <-ctx.Done():
			return totalBytesWritten, fmt.Errorf("copy cancelled: %w", ctx.Err())
		default:
		}

		bytesRead, readErr := src.Read(buf)
		if bytesRead > 0 {
			bytesWritten, writeErr := dst.Write(buf[0:bytesRead])
			if bytesWritten < 0 || bytesRead < bytesWritten {
				bytesWritten = 0
				if writeErr == nil {
					writeErr = fmt.Errorf("invalid write result")
				}
			}

			if writeErr != nil {
				return totalBytesWritten, writeErr
			}

			if bytesRead != bytesWritten {
				return totalBytesWritten, io.ErrShortWrite
			}

			totalBytesWritten += int64(bytesWritten)
		}

		if readErr != nil {
			if readErr != io.EOF {
				return totalBytesWritten, readErr
			}

			break
		}
	}

	return totalBytesWritten, nil
}

// containsIgnoreCase checks if a string contains a substring, ignoring case
func containsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}

// generateFilteredTocList writes a pg_restore TOC list (for -L) with the object classes selected
// by options dropped, so pg_restore skips them.
func (uc *RestorePostgresqlBackupUsecase) generateFilteredTocList(
	ctx context.Context,
	pgBin string,
	backupFile string,
	credentials *postgresql_shared.CredentialTempFiles,
	pgConfig *pgtypes.PostgresqlLogicalDatabase,
	options restores_core.RestoreOptions,
) (string, error) {
	uc.logger.InfoContext(ctx, "generating filtered TOC list", "backup_file", backupFile)

	// Run pg_restore -l to get the TOC list
	// The binary is a database client from the bundled tools directory, and the arguments go to it as a list, not through a shell.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	listCmd := exec.CommandContext(ctx, pgBin, "-l", backupFile)
	uc.setupPgRestoreEnvironment(listCmd, credentials, pgConfig)

	tocOutput, err := listCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to generate TOC list: %w", err)
	}

	// " EXTENSION " catches both CREATE EXTENSION ("3420; 0 0 EXTENSION - uuid-ossp") and
	// COMMENT ON EXTENSION ("3462; 0 0 COMMENT - EXTENSION "uuid-ossp"") entries.
	isExtensionEntry := func(upperLine string) bool {
		return strings.Contains(upperLine, " EXTENSION ")
	}
	// " USER MAPPING " catches CREATE USER MAPPING entries
	// ("18239; 0 0 USER MAPPING - platform SERVER oracle_tb_server").
	isUserMappingEntry := func(upperLine string) bool {
		return strings.Contains(upperLine, " USER MAPPING ")
	}

	var filteredLines []string
	for line := range strings.SplitSeq(string(tocOutput), "\n") {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		upperLine := strings.ToUpper(trimmedLine)

		if options.IsExcludeExtensions && isExtensionEntry(upperLine) {
			uc.logger.InfoContext(ctx, "excluding extension-related entry from restore", "toc_line", trimmedLine)
			continue
		}

		if options.IsSkipUserMappings && isUserMappingEntry(upperLine) {
			uc.logger.InfoContext(ctx, "excluding user mapping entry from restore", "toc_line", trimmedLine)
			continue
		}

		filteredLines = append(filteredLines, line)
	}

	// Write filtered TOC to temporary file
	tocFile, err := os.CreateTemp(config.GetEnv().TempFolder, "pg_restore_toc_*.list")
	if err != nil {
		return "", fmt.Errorf("failed to create TOC list file: %w", err)
	}
	tocFilePath := tocFile.Name()

	filteredContent := strings.Join(filteredLines, "\n")
	if _, err := tocFile.WriteString(filteredContent); err != nil {
		_ = tocFile.Close()
		_ = os.Remove(tocFilePath)
		return "", fmt.Errorf("failed to write TOC list file: %w", err)
	}

	if err := tocFile.Close(); err != nil {
		_ = os.Remove(tocFilePath)
		return "", fmt.Errorf("failed to close TOC list file: %w", err)
	}

	uc.logger.InfoContext(ctx, "generated filtered TOC list file",
		"toc_file", tocFilePath,
		"original_lines", len(strings.Split(string(tocOutput), "\n")),
		"filtered_lines", len(filteredLines),
	)

	return tocFilePath, nil
}
