package usecases_logical_mysql

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
	mysqltypes "databasus-backend/internal/features/databases/databases/mysql"
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

type CreateMysqlBackupUsecase struct {
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

func (uc *CreateMysqlBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	db *databases.Database,
	fileStore backups_core_logical.BackupFileStore,
	backupProgressListener func(completedMBs float64),
) (*backups_core_logical.BackupArtifacts, error) {
	logger := uc.logger.With("database_id", db.ID, "storage_id", backup.StorageID)

	logger.InfoContext(ctx, "creating mysql backup via mysqldump")

	tunneledDatabase, err := databases.OpenTunnel(ctx, databases.OpenTunnelSpec{
		Database:  db,
		Logger:    logger,
		Encryptor: uc.fieldEncryptor,
	})
	if err != nil {
		return nil, err
	}

	defer tunneledDatabase.Close()

	mysqlDatabase := tunneledDatabase.GetDatabaseThroughTunnel().Mysql
	if mysqlDatabase == nil {
		return nil, fmt.Errorf("mysql database configuration is required")
	}

	if mysqlDatabase.Database == nil || *mysqlDatabase.Database == "" {
		return nil, fmt.Errorf("database name is required for mysqldump backups")
	}

	decryptedPassword, err := uc.fieldEncryptor.Decrypt(mysqlDatabase.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt database password: %w", err)
	}

	rawSizeMB, err := mysqlDatabase.GetRawDbSizeMb(ctx, logger, uc.fieldEncryptor)
	if err != nil {
		logger.WarnContext(ctx, "failed to fetch raw db size before backup", "error", err)
	} else {
		backup.BackupRawDbSizeMb = rawSizeMB
	}

	args := uc.buildMysqldumpArgs(mysqlDatabase)

	mysqldumpBin, err := tools.GetMysqlExecutable(mysqlDatabase.Version, tools.MysqlExecutableMysqldump)
	if err != nil {
		return nil, err
	}

	return uc.streamToStorage(
		ctx,
		backup,
		backupConfig,
		mysqldumpBin,
		args,
		decryptedPassword,
		fileStore,
		backupProgressListener,
		mysqlDatabase,
	)
}

func (uc *CreateMysqlBackupUsecase) buildMysqldumpArgs(my *mysqltypes.MysqlDatabase) []string {
	args := []string{
		"--host=" + my.Host,
		"--port=" + strconv.Itoa(my.Port),
		"--user=" + my.Username,
		"--single-transaction",
		"--routines",
		"--set-gtid-purged=OFF",
		"--quick",
		"--skip-add-locks",
		"--verbose",
		// Dumping tablespace definitions makes mysqldump read INFORMATION_SCHEMA.FILES, which costs
		// a global PROCESS privilege that a backup role has no other use for and that managed
		// providers often refuse to grant.
		"--no-tablespaces",
	}

	// Triggers are on by default, so a role without the privilege has to be opted out explicitly
	// or mysqldump fails on SHOW TRIGGERS instead of dumping without them.
	if my.HasPrivilege("TRIGGER") {
		args = append(args, "--triggers")
	} else {
		args = append(args, "--skip-triggers")
	}

	if my.HasPrivilege("EVENT") {
		args = append(args, "--events")
	}

	if my.Database != nil && *my.Database != "" {
		for _, excludedTable := range namelist.NormalizeUniqueNames(my.ExcludeTables) {
			args = append(args, "--ignore-table="+*my.Database+"."+excludedTable)
		}
	}

	args = append(args, "--max-allowed-packet=1G")

	if my.IsHttps {
		args = append(args, "--ssl-mode=REQUIRED")
	} else {
		args = append(args, "--ssl-mode=DISABLED")
	}

	if my.Database != nil && *my.Database != "" {
		args = append(args, *my.Database)
	}

	return args
}

func (uc *CreateMysqlBackupUsecase) streamToStorage(
	parentCtx context.Context,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	mysqlBin string,
	args []string,
	password string,
	fileStore backups_core_logical.BackupFileStore,
	backupProgressListener func(completedMBs float64),
	myConfig *mysqltypes.MysqlDatabase,
) (*backups_core_logical.BackupArtifacts, error) {
	uc.logger.InfoContext(parentCtx, "streaming MySQL backup to storage", "mysql_bin", mysqlBin)

	ctx, cancel := backups_core_logical.NewBackupContext(parentCtx, config.GetShutdownSignal())
	defer cancel(nil)

	myCnfFile, err := uc.createTempMyCnfFile(myConfig, password)
	if err != nil {
		return nil, fmt.Errorf("failed to create .my.cnf: %w", err)
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(myCnfFile)) }()

	compressionArgs := uc.probeNetworkCompressionArgs(ctx, CompressionProbeSpec{
		MysqldumpBin: mysqlBin,
		MyCnfFile:    myCnfFile,
		DatabaseName: *myConfig.Database,
		DatabaseID:   backup.DatabaseID,
	}, myConfig.Version)

	fullArgs := append([]string{"--defaults-file=" + myCnfFile}, compressionArgs...)
	fullArgs = append(fullArgs, args...)

	// The binary is a database client from the bundled tools directory, and the arguments go to it as a list, not through a shell.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, mysqlBin, fullArgs...)
	uc.logger.InfoContext(parentCtx, "executing MySQL backup command", "command", cmd.String())

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
		return nil, fmt.Errorf("start %s: %w", filepath.Base(mysqlBin), err)
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
		return nil, uc.buildMysqldumpErrorMessage(waitErr, stderrOutput.Bytes(), mysqlBin)
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

func (uc *CreateMysqlBackupUsecase) createTempMyCnfFile(
	myConfig *mysqltypes.MysqlDatabase,
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
`, myConfig.Username, tools.EscapeMysqlPassword(password), myConfig.Host, myConfig.Port)

	if myConfig.IsHttps {
		content += "ssl-mode=REQUIRED\n"
	} else {
		content += "ssl-mode=DISABLED\n"
	}

	err = os.WriteFile(myCnfFile, []byte(content), 0o600)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to write .my.cnf: %w", err)
	}

	return myCnfFile, nil
}

func (uc *CreateMysqlBackupUsecase) copyWithShutdownCheck(
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

func (uc *CreateMysqlBackupUsecase) setupBackupEncryption(
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

func (uc *CreateMysqlBackupUsecase) cleanupOnCancellation(
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

func (uc *CreateMysqlBackupUsecase) closeWriters(
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

func (uc *CreateMysqlBackupUsecase) buildMysqldumpErrorMessage(
	waitErr error,
	stderrOutput []byte,
	mysqlBin string,
) error {
	stderrStr := string(stderrOutput)
	errorMsg := fmt.Sprintf(
		"%s failed: %v – stderr: %s",
		filepath.Base(mysqlBin),
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

func (uc *CreateMysqlBackupUsecase) handleConnectionErrors(stderrStr string) error {
	if containsIgnoreCase(stderrStr, "access denied") {
		return fmt.Errorf(
			"MySQL access denied. Check username and password. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "can't connect") ||
		containsIgnoreCase(stderrStr, "connection refused") {
		return fmt.Errorf(
			"MySQL connection refused. Check if the server is running and accessible. stderr: %s",
			stderrStr,
		)
	}

	if isCompressionRejection(stderrStr) {
		return fmt.Errorf(
			"MySQL rejected the network compression algorithm that had just passed the "+
				"pre-flight handshake probe. Compression is re-probed on every run, so retry "+
				"the backup. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "unknown database") {
		return fmt.Errorf(
			"MySQL database does not exist. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "ssl") {
		return fmt.Errorf(
			"MySQL SSL connection failed. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "timeout") {
		return fmt.Errorf(
			"MySQL connection timeout. stderr: %s",
			stderrStr,
		)
	}

	return fmt.Errorf("MySQL connection or authentication error. stderr: %s", stderrStr)
}

func containsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}
