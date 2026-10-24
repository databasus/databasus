package usecases_logical_mariadb

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
	"github.com/klauspost/compress/zstd"

	"databasus-backend/internal/config"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backup_encryption "databasus-backend/internal/features/backups/backups/encryption"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	mariadbtypes "databasus-backend/internal/features/databases/databases/mariadb"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	storage_files "databasus-backend/internal/features/storages/files"
	"databasus-backend/internal/util/encryption"
	io_utils "databasus-backend/internal/util/io"
	"databasus-backend/internal/util/namelist"
	"databasus-backend/internal/util/tools"
)

const (
	copyBufferSize              = 8 * 1024 * 1024
	progressReportIntervalMB    = 1.0
	zstdStorageCompressionLevel = 5
	exitCodeGenericError        = 1
	exitCodeConnectionError     = 2
)

type CreateMariadbBackupUsecase struct {
	logger           *slog.Logger
	secretKeyService *encryption_secrets.SecretKeyService
	fieldEncryptor   encryption.FieldEncryptor
}

type writeResult struct {
	bytesWritten int
	writeErr     error
}

type compressedCopySpec struct {
	Destination            io.Writer
	Source                 io.Reader
	CompressedBytesCounter *io_utils.CountingWriter
	ProgressListener       func(completedMBs float64)
}

func (uc *CreateMariadbBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	db *databases.Database,
	fileStore backups_core_logical.BackupFileStore,
	backupProgressListener func(completedMBs float64),
) (*backups_core_logical.BackupArtifacts, error) {
	logger := uc.logger.With("database_id", db.ID, "storage_id", backup.StorageID)

	logger.InfoContext(ctx, "creating mariadb backup via mariadb-dump")

	tunneledDatabase, err := databases.OpenTunnel(ctx, databases.OpenTunnelSpec{
		Database:  db,
		Logger:    logger,
		Encryptor: uc.fieldEncryptor,
	})
	if err != nil {
		return nil, err
	}

	defer tunneledDatabase.Close()

	mariadbDatabase := tunneledDatabase.GetDatabaseThroughTunnel().Mariadb
	if mariadbDatabase == nil {
		return nil, fmt.Errorf("mariadb database configuration is required")
	}

	if mariadbDatabase.Database == nil || *mariadbDatabase.Database == "" {
		return nil, fmt.Errorf("database name is required for mariadb-dump backups")
	}

	decryptedPassword, err := uc.fieldEncryptor.Decrypt(mariadbDatabase.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt database password: %w", err)
	}

	rawSizeMB, err := mariadbDatabase.GetRawDbSizeMb(ctx, logger, uc.fieldEncryptor)
	if err != nil {
		logger.WarnContext(ctx, "failed to fetch raw db size before backup", "error", err)
	} else {
		backup.BackupRawDbSizeMb = rawSizeMB
	}

	args := uc.buildMariadbDumpArgs(mariadbDatabase)

	mariadbDumpBin, err := tools.GetMariadbExecutable(mariadbDatabase.Version, tools.MariadbExecutableMariadbDump)
	if err != nil {
		return nil, err
	}

	return uc.streamToStorage(
		ctx,
		backup,
		backupConfig,
		mariadbDumpBin,
		args,
		decryptedPassword,
		fileStore,
		backupProgressListener,
		mariadbDatabase,
	)
}

func (uc *CreateMariadbBackupUsecase) buildMariadbDumpArgs(
	mdb *mariadbtypes.MariadbDatabase,
) []string {
	args := []string{
		"--host=" + mdb.Host,
		"--port=" + strconv.Itoa(mdb.Port),
		"--user=" + mdb.Username,
		"--single-transaction",
		"--routines",
		"--quick",
		"--skip-add-locks",
		"--verbose",
		// Dumping tablespace definitions makes mariadb-dump read INFORMATION_SCHEMA.FILES, which costs
		// a global PROCESS privilege that a backup role has no other use for and that managed
		// providers often refuse to grant.
		"--no-tablespaces",
	}

	// Triggers are on by default, so a role without the privilege has to be opted out explicitly
	// or mariadb-dump fails on SHOW TRIGGERS instead of dumping without them.
	if mdb.HasPrivilege("TRIGGER") {
		args = append(args, "--triggers")
	} else {
		args = append(args, "--skip-triggers")
	}

	if mdb.HasPrivilege("EVENT") && !mdb.IsExcludeEvents {
		args = append(args, "--events")
	}

	if mdb.Database != nil && *mdb.Database != "" {
		for _, excludedTable := range namelist.NormalizeUniqueNames(mdb.ExcludeTables) {
			args = append(args, "--ignore-table="+*mdb.Database+"."+excludedTable)
		}
	}

	args = append(args, "--max-allowed-packet=1G")

	if mdb.IsHttps {
		args = append(args, "--ssl")
		args = append(args, "--skip-ssl-verify-server-cert")
	} else {
		args = append(args, "--skip-ssl")
	}

	if mdb.Database != nil && *mdb.Database != "" {
		args = append(args, *mdb.Database)
	}

	return args
}

func (uc *CreateMariadbBackupUsecase) streamToStorage(
	parentCtx context.Context,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	mariadbBin string,
	args []string,
	password string,
	fileStore backups_core_logical.BackupFileStore,
	backupProgressListener func(completedMBs float64),
	mdbConfig *mariadbtypes.MariadbDatabase,
) (*backups_core_logical.BackupArtifacts, error) {
	uc.logger.InfoContext(parentCtx, "streaming MariaDB backup to storage", "mariadb_bin", mariadbBin)

	ctx, cancel := backups_core_logical.NewBackupContext(parentCtx, config.GetShutdownSignal())
	defer cancel(nil)

	myCnfFile, err := uc.createTempMyCnfFile(mdbConfig, password)
	if err != nil {
		return nil, fmt.Errorf("failed to create .my.cnf: %w", err)
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(myCnfFile)) }()

	compressionArgs := uc.probeNetworkCompressionArgs(ctx, CompressionProbeSpec{
		MariadbDumpBin: mariadbBin,
		MyCnfFile:      myCnfFile,
		DatabaseName:   *mdbConfig.Database,
		DatabaseID:     backup.DatabaseID,
		IsHttps:        mdbConfig.IsHttps,
	})

	fullArgs := append([]string{"--defaults-file=" + myCnfFile}, compressionArgs...)
	fullArgs = append(fullArgs, args...)

	// The binary is a database client from the bundled tools directory, and the arguments go to it as a list, not through a shell.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, mariadbBin, fullArgs...)
	uc.logger.InfoContext(parentCtx, "executing MariaDB backup command", "command", cmd.String())

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
		"MYSQL_PWD=",
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	)

	dumpStdout, err := cmd.StdoutPipe()
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

	compressedBytesCounter := io_utils.NewCountingWriter(finalWriter)

	zstdWriter, err := zstd.NewWriter(compressedBytesCounter,
		zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(zstdStorageCompressionLevel)))
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd writer: %w", err)
	}

	// The dump tool starts before the storage write, so a storage that fails at once cannot
	// cancel the context the tool is started with and hide its own error.
	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", filepath.Base(mariadbBin), err)
	}

	fileWrite := storage_files.StartBackgroundWrite(
		ctx,
		fileStore,
		storage_files.StoredFileReference{StorageID: backup.StorageID, FileName: backup.FileName},
		storageReader,
		cancel,
	)

	copyResultCh := make(chan error, 1)
	go func() {
		copyResultCh <- uc.copyWithShutdownCheck(ctx, compressedCopySpec{
			Destination:            zstdWriter,
			Source:                 dumpStdout,
			CompressedBytesCounter: compressedBytesCounter,
			ProgressListener:       backupProgressListener,
		})
	}()

	copyErr := <-copyResultCh
	waitErr := cmd.Wait()

	select {
	case earlySaveErr := <-fileWrite.Errors:
		if earlySaveErr != nil {
			_ = zstdWriter.Close()
			_ = uc.closeWriters(encryptionWriter, storageWriter)
			return nil, fmt.Errorf("save to storage: %w", earlySaveErr)
		}
		fileWrite.Errors <- nil
	default:
	}

	select {
	case <-ctx.Done():
		uc.cleanupOnCancellation(zstdWriter, encryptionWriter, storageWriter, fileWrite.Errors)
		return nil, backups_core_logical.DescribeBackupInterruption(parentCtx, ctx)
	default:
	}

	if err := zstdWriter.Close(); err != nil {
		uc.logger.ErrorContext(parentCtx, "failed to close zstd writer", "error", err)
	}
	if err := uc.closeWriters(encryptionWriter, storageWriter); err != nil {
		<-fileWrite.Errors
		return nil, err
	}

	saveErr := <-fileWrite.Errors

	if waitErr == nil && copyErr == nil && saveErr == nil && backupProgressListener != nil {
		compressedSizeMB := float64(compressedBytesCounter.GetBytesWritten()) / (1024 * 1024)
		backupProgressListener(compressedSizeMB)
	}

	switch {
	case waitErr != nil:
		return nil, uc.buildMariadbDumpErrorMessage(waitErr, stderrOutput.Bytes(), mariadbBin)
	case copyErr != nil:
		return nil, fmt.Errorf("copy to storage: %w", copyErr)
	case saveErr != nil:
		return nil, fmt.Errorf("save to storage: %w", saveErr)
	}

	return &backups_core_logical.BackupArtifacts{
		Metadata: &backupMetadata,
		Receipts: []storage_files.WriteReceipt{<-fileWrite.Receipts},
	}, nil
}

func (uc *CreateMariadbBackupUsecase) createTempMyCnfFile(
	mdbConfig *mariadbtypes.MariadbDatabase,
	password string,
) (string, error) {
	// Credential files use OS temp dir (/tmp) because some filesystems
	// (e.g. ZFS on TrueNAS) ignore chmod, causing "group or world access" errors.
	tempDir, err := os.MkdirTemp(os.TempDir(), "mycnf_"+uuid.New().String())
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	if err := os.Chmod(tempDir, 0o700); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to set temp directory permissions: %w", err)
	}

	myCnfFile := filepath.Join(tempDir, ".my.cnf")

	content := fmt.Sprintf(`[client]
user=%s
password="%s"
host=%s
port=%d
`, mdbConfig.Username, tools.EscapeMariadbPassword(password), mdbConfig.Host, mdbConfig.Port)

	if mdbConfig.IsHttps {
		content += "ssl=true\n"
	} else {
		content += "ssl=false\n"
	}

	err = os.WriteFile(myCnfFile, []byte(content), 0o600)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to write .my.cnf: %w", err)
	}

	return myCnfFile, nil
}

func (uc *CreateMariadbBackupUsecase) copyWithShutdownCheck(
	ctx context.Context,
	copySpec compressedCopySpec,
) error {
	buf := make([]byte, copyBufferSize)
	var lastReportedMB float64

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("copy cancelled: %w", ctx.Err())
		default:
		}

		bytesRead, readErr := copySpec.Source.Read(buf)
		if bytesRead > 0 {
			writeResultCh := make(chan writeResult, 1)
			go func() {
				bytesWritten, writeErr := copySpec.Destination.Write(buf[0:bytesRead])
				writeResultCh <- writeResult{bytesWritten, writeErr}
			}()

			var bytesWritten int
			var writeErr error

			select {
			case <-ctx.Done():
				return fmt.Errorf("copy cancelled during write: %w", ctx.Err())
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
				return writeErr
			}

			if bytesRead != bytesWritten {
				return io.ErrShortWrite
			}

			if copySpec.ProgressListener != nil {
				compressedSizeMB := float64(
					copySpec.CompressedBytesCounter.GetBytesWritten(),
				) / (1024 * 1024)
				if compressedSizeMB >= lastReportedMB+progressReportIntervalMB {
					copySpec.ProgressListener(compressedSizeMB)
					lastReportedMB = compressedSizeMB
				}
			}
		}

		if readErr != nil {
			if readErr != io.EOF {
				return readErr
			}
			break
		}
	}

	return nil
}

func (uc *CreateMariadbBackupUsecase) setupBackupEncryption(
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

	masterKey, err := uc.secretKeyService.GetSecretKey()
	if err != nil {
		return nil, nil, metadata, fmt.Errorf("failed to get master key: %w", err)
	}

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

func (uc *CreateMariadbBackupUsecase) cleanupOnCancellation(
	zstdWriter *zstd.Encoder,
	encryptionWriter *backup_encryption.EncryptionWriter,
	storageWriter io.WriteCloser,
	writeErrors chan error,
) {
	if zstdWriter != nil {
		go func() {
			if closeErr := zstdWriter.Close(); closeErr != nil {
				uc.logger.Error(
					"Failed to close zstd writer during cancellation",
					"error",
					closeErr,
				)
			}
		}()
	}

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

func (uc *CreateMariadbBackupUsecase) closeWriters(
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

func (uc *CreateMariadbBackupUsecase) buildMariadbDumpErrorMessage(
	waitErr error,
	stderrOutput []byte,
	mariadbBin string,
) error {
	stderrStr := string(stderrOutput)
	errorMsg := fmt.Sprintf(
		"%s failed: %v – stderr: %s",
		filepath.Base(mariadbBin),
		waitErr,
		stderrStr,
	)

	var exitErr *exec.ExitError
	if !errors.As(waitErr, &exitErr) {
		return errors.New(errorMsg)
	}

	exitCode := exitErr.ExitCode()

	if exitCode == exitCodeGenericError || exitCode == exitCodeConnectionError {
		return uc.handleConnectionErrors(stderrStr)
	}

	return errors.New(errorMsg)
}

func (uc *CreateMariadbBackupUsecase) handleConnectionErrors(stderrStr string) error {
	if containsIgnoreCase(stderrStr, "access denied") {
		return fmt.Errorf(
			"MariaDB access denied. Check username and password. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "can't connect") ||
		containsIgnoreCase(stderrStr, "connection refused") {
		return fmt.Errorf(
			"MariaDB connection refused. Check if the server is running and accessible. stderr: %s",
			stderrStr,
		)
	}

	if isCompressionRejection(stderrStr) {
		return fmt.Errorf(
			"MariaDB rejected the network compression algorithm that had just passed the "+
				"pre-flight handshake probe. Compression is re-probed on every run, so retry "+
				"the backup. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "unknown database") {
		return fmt.Errorf(
			"MariaDB database does not exist. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "ssl") {
		return fmt.Errorf(
			"MariaDB SSL connection failed. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "timeout") {
		return fmt.Errorf(
			"MariaDB connection timeout. stderr: %s",
			stderrStr,
		)
	}

	return fmt.Errorf("MariaDB connection or authentication error. stderr: %s", stderrStr)
}

func containsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}
