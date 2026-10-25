package usecases_logical_postgresql

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backup_encryption "databasus-backend/internal/features/backups/backups/encryption"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	pgtypes "databasus-backend/internal/features/databases/databases/postgresql/logical"
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	storage_files "databasus-backend/internal/features/storages/files"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/namelist"
	"databasus-backend/internal/util/tools"
)

const (
	copyBufferSize           = 8 * 1024 * 1024
	progressReportIntervalMB = 1.0
	pgConnectTimeout         = 30
	compressionLevel         = 5
	exitCodeAccessViolation  = -1073741819
	exitCodeGenericError     = 1
	exitCodeConnectionError  = 2
)

type CreatePostgresqlBackupUsecase struct {
	logger         *slog.Logger
	masterKey      *crypto.MasterKey
	fieldEncryptor encryption.FieldEncryptor
}

type writeResult struct {
	bytesWritten int
	writeErr     error
}

func (uc *CreatePostgresqlBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	db *databases.Database,
	fileStore backups_core_logical.BackupFileStore,
	backupProgressListener func(
		completedMBs float64,
	),
) (*backups_core_logical.BackupArtifacts, error) {
	logger := uc.logger.With("database_id", db.ID, "storage_id", backup.StorageID, "backup_id", backup.ID)

	logger.InfoContext(ctx, "creating postgresql backup via pg_dump custom format")

	if db.PostgresqlLogical == nil {
		return nil, fmt.Errorf("postgresql database configuration is required for pg_dump backups")
	}

	tunneledDatabase, err := databases.OpenTunnel(ctx, databases.OpenTunnelSpec{
		Database:  db,
		Logger:    logger,
		Encryptor: uc.fieldEncryptor,
	})
	if err != nil {
		return nil, err
	}

	defer tunneledDatabase.Close()

	databaseThroughTunnel := tunneledDatabase.GetDatabaseThroughTunnel()
	pg := databaseThroughTunnel.PostgresqlLogical

	if pg.Database == nil || *pg.Database == "" {
		return nil, fmt.Errorf("database name is required for pg_dump backups")
	}

	args := uc.buildPgDumpArgs(pg)

	decryptedPassword, err := uc.fieldEncryptor.Decrypt(pg.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt database password: %w", err)
	}

	rawSizeMB, err := pg.GetRawDbSizeMb(ctx, logger, uc.fieldEncryptor)
	if err != nil {
		logger.WarnContext(ctx, "failed to fetch raw db size before backup", "error", err)
	} else {
		backup.BackupRawDbSizeMb = rawSizeMB
	}

	timescaledbVersion, err := pg.GetTimescaleDBVersion(ctx, uc.fieldEncryptor)
	if err != nil {
		logger.WarnContext(ctx, "failed to detect timescaledb extension before backup", "error", err)
	} else {
		backup.TimescaledbVersion = timescaledbVersion
	}

	return uc.streamToStorage(
		ctx,
		backup,
		backupConfig,
		tools.GetPostgresqlExecutable(pg.Version, "pg_dump"),
		args,
		decryptedPassword,
		fileStore,
		databaseThroughTunnel,
		backupProgressListener,
	)
}

func (uc *CreatePostgresqlBackupUsecase) streamToStorage(
	parentCtx context.Context,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	pgBin string,
	args []string,
	password string,
	fileStore backups_core_logical.BackupFileStore,
	db *databases.Database,
	backupProgressListener func(completedMBs float64),
) (*backups_core_logical.BackupArtifacts, error) {
	uc.logger.InfoContext(parentCtx, "streaming PostgreSQL backup to storage", "pg_bin", pgBin, "args", args)

	ctx, cancel := backups_core_logical.NewBackupContext(parentCtx, config.GetShutdownSignal())
	defer cancel(nil)

	credentials, err := postgresql_shared.WriteCredentialFilesToTempDir(
		db.PostgresqlLogical.CredentialSpec(), password, uc.fieldEncryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential files: %w", err)
	}
	defer credentials.Remove()

	// The binary is a database client from the bundled tools directory, and the arguments go to it as a list, not through a shell.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, pgBin, args...)
	uc.logger.InfoContext(parentCtx, "executing PostgreSQL backup command", "command", cmd.String())

	if err := uc.setupPgEnvironment(
		cmd,
		credentials,
		db.PostgresqlLogical.CredentialSpec(),
		password,
		db.PostgresqlLogical.CpuCount,
		pgBin,
	); err != nil {
		return nil, err
	}

	pgStdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	var stderrOutput bytes.Buffer
	cmd.Stderr = &stderrOutput
	cmd.WaitDelay = backups_core_logical.DumpToolOutputWaitDelay

	storageReader, storageWriter := io.Pipe()

	finalWriter, encryptionWriter, backupMetadata, err := uc.setupBackupEncryption(
		backup.ID,
		backupConfig,
		storageWriter,
	)
	if err != nil {
		return nil, err
	}

	// The dump tool starts before the storage write, so a storage that fails at once cannot
	// cancel the context the tool is started with and hide its own error.
	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", filepath.Base(pgBin), err)
	}

	fileWrite := storage_files.StartBackgroundWrite(
		ctx,
		fileStore,
		storage_files.StoredFileReference{StorageID: backup.StorageID, FileName: backup.FileName},
		storageReader,
		cancel,
	)

	copyResultCh := make(chan error, 1)
	bytesWrittenCh := make(chan int64, 1)
	go func() {
		bytesWritten, err := uc.copyWithShutdownCheck(
			ctx,
			finalWriter,
			pgStdout,
			backupProgressListener,
		)
		bytesWrittenCh <- bytesWritten
		copyResultCh <- err
	}()

	copyErr := <-copyResultCh
	bytesWritten := <-bytesWrittenCh
	waitErr := cmd.Wait()

	select {
	case earlySaveErr := <-fileWrite.Errors:
		if earlySaveErr != nil {
			_ = uc.closeWriters(encryptionWriter, storageWriter)
			return nil, fmt.Errorf("save to storage: %w", earlySaveErr)
		}
		fileWrite.Errors <- nil
	default:
	}

	select {
	case <-ctx.Done():
		uc.cleanupOnCancellation(encryptionWriter, storageWriter, fileWrite.Errors)
		return nil, backups_core_logical.DescribeBackupInterruption(parentCtx, ctx)
	default:
	}

	if err := uc.closeWriters(encryptionWriter, storageWriter); err != nil {
		<-fileWrite.Errors
		return nil, err
	}

	saveErr := <-fileWrite.Errors

	// Send final sizing after backup is completed
	if waitErr == nil && copyErr == nil && saveErr == nil && backupProgressListener != nil {
		sizeMB := float64(bytesWritten) / (1024 * 1024)
		backupProgressListener(sizeMB)
	}

	switch {
	case waitErr != nil:
		if ctx.Err() != nil {
			return nil, backups_core_logical.DescribeBackupInterruption(parentCtx, ctx)
		}
		return nil, uc.buildPgDumpErrorMessage(waitErr, stderrOutput.Bytes(), pgBin, args, password)
	case copyErr != nil:
		if ctx.Err() != nil {
			return nil, backups_core_logical.DescribeBackupInterruption(parentCtx, ctx)
		}
		return nil, fmt.Errorf("copy to storage: %w", copyErr)
	case saveErr != nil:
		if ctx.Err() != nil {
			return nil, backups_core_logical.DescribeBackupInterruption(parentCtx, ctx)
		}
		return nil, fmt.Errorf("save to storage: %w", saveErr)
	}

	return &backups_core_logical.BackupArtifacts{
		Metadata: &backupMetadata,
		Receipts: []storage_files.WriteReceipt{<-fileWrite.Receipts},
	}, nil
}

func (uc *CreatePostgresqlBackupUsecase) copyWithShutdownCheck(
	ctx context.Context,
	dst io.Writer,
	src io.Reader,
	backupProgressListener func(completedMBs float64),
) (int64, error) {
	buf := make([]byte, copyBufferSize)
	var totalBytesWritten int64
	var lastReportedMB float64

	for {
		select {
		case <-ctx.Done():
			return totalBytesWritten, fmt.Errorf("copy cancelled: %w", ctx.Err())
		default:
		}

		bytesRead, readErr := src.Read(buf)
		if bytesRead > 0 {
			writeResultCh := make(chan writeResult, 1)
			go func() {
				bytesWritten, writeErr := dst.Write(buf[0:bytesRead])
				writeResultCh <- writeResult{bytesWritten, writeErr}
			}()

			var bytesWritten int
			var writeErr error

			select {
			case <-ctx.Done():
				return totalBytesWritten, fmt.Errorf("copy cancelled during write: %w", ctx.Err())
			case result := <-writeResultCh:
				bytesWritten = result.bytesWritten
				writeErr = result.writeErr
			}

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

			if backupProgressListener != nil {
				currentSizeMB := float64(totalBytesWritten) / (1024 * 1024)
				if currentSizeMB >= lastReportedMB+progressReportIntervalMB {
					backupProgressListener(currentSizeMB)
					lastReportedMB = currentSizeMB
				}
			}
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

func (uc *CreatePostgresqlBackupUsecase) buildPgDumpArgs(pg *pgtypes.PostgresqlLogicalDatabase) []string {
	args := []string{
		"-Fc",
		"--no-password",
		"-h", pg.Host,
		"-p", strconv.Itoa(pg.Port),
		"-U", pg.Username,
		"-d", postgresql_shared.BuildDatabaseNameConninfo(*pg.Database),
		"--verbose",
	}

	for _, includedSchema := range namelist.NormalizeUniqueNames(pg.IncludeSchemas) {
		args = append(args, "-n", includedSchema)
	}

	for _, excludedTable := range namelist.NormalizeUniqueNames(pg.ExcludeTables) {
		args = append(args, "--exclude-table="+excludedTable)
	}

	for _, dataExcludedTable := range namelist.NormalizeUniqueNames(pg.ExcludeTableData) {
		args = append(args, "--exclude-table-data="+dataExcludedTable)
	}

	compressionArgs := uc.getCompressionArgs(pg.Version)
	return append(args, compressionArgs...)
}

func (uc *CreatePostgresqlBackupUsecase) getCompressionArgs(
	version tools.PostgresqlVersion,
) []string {
	if uc.isOlderPostgresVersion(version) {
		uc.logger.Info("using gzip compression level 5 (zstd not available)", "version", version)
		return []string{"-Z", strconv.Itoa(compressionLevel)}
	}

	uc.logger.Info("using zstd compression level 5", "version", version)
	return []string{fmt.Sprintf("--compress=zstd:%d", compressionLevel)}
}

func (uc *CreatePostgresqlBackupUsecase) isOlderPostgresVersion(
	version tools.PostgresqlVersion,
) bool {
	return version == tools.PostgresqlVersion12 ||
		version == tools.PostgresqlVersion13 ||
		version == tools.PostgresqlVersion14 ||
		version == tools.PostgresqlVersion15
}

func (uc *CreatePostgresqlBackupUsecase) setupPgEnvironment(
	cmd *exec.Cmd,
	credentials *postgresql_shared.CredentialTempFiles,
	credentialSpec postgresql_shared.CredentialSpec,
	password string,
	cpuCount int,
	pgBin string,
) error {
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "PGPASSFILE="+credentials.PgpassPath)
	cmd.Env = append(cmd.Env, postgresql_shared.GetPgHostAddrEnv(credentialSpec)...)

	uc.logger.Info("setting up PostgreSQL environment",
		"pg_bin", pgBin,
		"parallel_jobs", cpuCount,
	)

	cmd.Env = append(cmd.Env,
		"PGCLIENTENCODING=UTF8",
		"PGCONNECT_TIMEOUT="+strconv.Itoa(pgConnectTimeout),
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	)

	resolvedSslMode := credentialSpec.SslMode
	if resolvedSslMode == "" {
		resolvedSslMode = postgresql_shared.PostgresSslModeDisable
	}

	cmd.Env = append(cmd.Env,
		"PGSSLMODE="+string(resolvedSslMode),
		"PGSSLCERT="+credentials.ClientCertPath,
		"PGSSLKEY="+credentials.ClientKeyPath,
		"PGSSLROOTCERT="+credentials.RootCertPath,
		"PGSSLCRL=",
	)
	uc.logger.Info("using SSL mode", "ssl_mode", resolvedSslMode)

	if _, err := exec.LookPath(pgBin); err != nil {
		return fmt.Errorf("PostgreSQL executable not found or not accessible: %s - %w", pgBin, err)
	}

	return nil
}

func (uc *CreatePostgresqlBackupUsecase) setupBackupEncryption(
	backupID uuid.UUID,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	storageWriter io.WriteCloser,
) (io.Writer, *backup_encryption.EncryptionWriter, backups_core_logical.BackupMetadata, error) {
	metadata := backups_core_logical.BackupMetadata{
		BackupID: backupID,
	}

	if backupConfig.Encryption != backups_core_enums.BackupEncryptionEncrypted {
		metadata.Encryption = backups_core_enums.BackupEncryptionNone
		uc.logger.Info("encryption disabled for backup", "backup_id", backupID)
		return storageWriter, nil, metadata, nil
	}

	masterKey := uc.masterKey.GetSecret()

	encSetup, err := backup_encryption.SetupEncryptionWriter(storageWriter, masterKey, backupID)
	if err != nil {
		return nil, nil, metadata, err
	}

	metadata.EncryptionSalt = &encSetup.SaltBase64
	metadata.EncryptionIV = &encSetup.NonceBase64
	metadata.Encryption = backups_core_enums.BackupEncryptionEncrypted

	uc.logger.Info("encryption enabled for backup", "backup_id", backupID)
	return encSetup.Writer, encSetup.Writer, metadata, nil
}

func (uc *CreatePostgresqlBackupUsecase) cleanupOnCancellation(
	encryptionWriter *backup_encryption.EncryptionWriter,
	storageWriter io.WriteCloser,
	writeErrors chan error,
) {
	if encryptionWriter != nil {
		go func() {
			if closeErr := encryptionWriter.Close(); closeErr != nil {
				uc.logger.Error(
					"Failed to close encrypting writer during cancellation",
					"error",
					closeErr,
				)
			}
		}()
	}

	if err := storageWriter.Close(); err != nil {
		uc.logger.Error("failed to close pipe writer during cancellation", "error", err)
	}

	<-writeErrors
}

func (uc *CreatePostgresqlBackupUsecase) closeWriters(
	encryptionWriter *backup_encryption.EncryptionWriter,
	storageWriter io.WriteCloser,
) error {
	encryptionCloseErrCh := make(chan error, 1)
	if encryptionWriter != nil {
		go func() {
			closeErr := encryptionWriter.Close()
			if closeErr != nil {
				uc.logger.Error("failed to close encrypting writer", "error", closeErr)
			}
			encryptionCloseErrCh <- closeErr
		}()
	} else {
		encryptionCloseErrCh <- nil
	}

	encryptionCloseErr := <-encryptionCloseErrCh
	if encryptionCloseErr != nil {
		if err := storageWriter.Close(); err != nil {
			uc.logger.Error("failed to close pipe writer after encryption error", "error", err)
		}
		return fmt.Errorf("failed to close encryption writer: %w", encryptionCloseErr)
	}

	if err := storageWriter.Close(); err != nil {
		uc.logger.Error("failed to close pipe writer", "error", err)
		return err
	}

	return nil
}

func (uc *CreatePostgresqlBackupUsecase) buildPgDumpErrorMessage(
	waitErr error,
	stderrOutput []byte,
	pgBin string,
	args []string,
	password string,
) error {
	stderrStr := string(stderrOutput)
	errorMsg := fmt.Sprintf("%s failed: %v – stderr: %s", filepath.Base(pgBin), waitErr, stderrStr)

	var exitErr *exec.ExitError
	if !errors.As(waitErr, &exitErr) {
		return errors.New(errorMsg)
	}

	exitCode := exitErr.ExitCode()

	if exitCode == exitCodeGenericError && strings.TrimSpace(stderrStr) == "" {
		return uc.handleExitCode1NoStderr(pgBin, args)
	}

	if exitCode == exitCodeAccessViolation {
		return uc.handleAccessViolation(pgBin, stderrStr)
	}

	if exitCode == exitCodeGenericError || exitCode == exitCodeConnectionError {
		return uc.handleConnectionErrors(stderrStr, password)
	}

	return errors.New(errorMsg)
}

func (uc *CreatePostgresqlBackupUsecase) handleExitCode1NoStderr(
	pgBin string,
	args []string,
) error {
	uc.logger.Error("pg_dump failed with exit status 1 but no stderr output",
		"pg_bin", pgBin,
		"args", args,
		"env_vars", []string{
			"PGCLIENTENCODING=UTF8",
			"PGCONNECT_TIMEOUT=" + strconv.Itoa(pgConnectTimeout),
			"LC_ALL=C.UTF-8",
			"LANG=C.UTF-8",
		},
	)

	return fmt.Errorf(
		"%s failed with exit status 1 but provided no error details. "+
			"This often indicates: "+
			"1) Connection timeout or refused connection, "+
			"2) Authentication failure with incorrect credentials, "+
			"3) Database does not exist, "+
			"4) Network connectivity issues, "+
			"5) PostgreSQL server not running. "+
			"Command executed: %s %s",
		filepath.Base(pgBin),
		pgBin,
		strings.Join(args, " "),
	)
}

func (uc *CreatePostgresqlBackupUsecase) handleAccessViolation(
	pgBin string,
	stderrStr string,
) error {
	uc.logger.Error("postgreSQL tool crashed with access violation",
		"pg_bin", pgBin,
		"exit_code", "0xC0000005",
	)

	return fmt.Errorf(
		"%s crashed with access violation (0xC0000005). "+
			"This may indicate incompatible PostgreSQL version, corrupted installation, or connection issues. "+
			"stderr: %s",
		filepath.Base(pgBin),
		stderrStr,
	)
}

func (uc *CreatePostgresqlBackupUsecase) handleConnectionErrors(
	stderrStr string,
	password string,
) error {
	if containsIgnoreCase(stderrStr, "pg_hba.conf") {
		return fmt.Errorf(
			"PostgreSQL connection rejected by server configuration (pg_hba.conf). "+
				"The server may not allow connections from your IP address or may require different authentication settings. "+
				"stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "no password supplied") ||
		containsIgnoreCase(stderrStr, "fe_sendauth") {
		return fmt.Errorf(
			"PostgreSQL authentication failed - no password supplied. "+
				"PGPASSWORD environment variable may not be working correctly on this system. "+
				"Password length: %d, Password empty: %v. "+
				"Consider using a .pgpass file as an alternative. "+
				"stderr: %s",
			len(password),
			password == "",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "ssl") && containsIgnoreCase(stderrStr, "connection") {
		return fmt.Errorf(
			"PostgreSQL SSL connection failed. "+
				"The server may require SSL encryption or have SSL configuration issues. "+
				"stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "connection") && containsIgnoreCase(stderrStr, "refused") {
		return fmt.Errorf(
			"PostgreSQL connection refused. "+
				"Check if the server is running and accessible from your network. "+
				"stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "authentication") ||
		containsIgnoreCase(stderrStr, "password") {
		return fmt.Errorf(
			"PostgreSQL authentication failed. Check username and password. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "timeout") {
		return fmt.Errorf(
			"PostgreSQL connection timeout. The server may be unreachable or overloaded. stderr: %s",
			stderrStr,
		)
	}

	return fmt.Errorf("PostgreSQL connection or authentication error. stderr: %s", stderrStr)
}

func containsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}
