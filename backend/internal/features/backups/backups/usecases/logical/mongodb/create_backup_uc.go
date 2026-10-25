package usecases_logical_mongodb

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

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backup_encryption "databasus-backend/internal/features/backups/backups/encryption"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	mongodbtypes "databasus-backend/internal/features/databases/databases/mongodb"
	storage_files "databasus-backend/internal/features/storages/files"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/namelist"
	"databasus-backend/internal/util/tools"
)

const (
	copyBufferSize           = 8 * 1024 * 1024
	progressReportIntervalMB = 1.0
)

type CreateMongodbBackupUsecase struct {
	logger         *slog.Logger
	masterKey      *crypto.MasterKey
	fieldEncryptor encryption.FieldEncryptor
}

type writeResult struct {
	bytesWritten int
	writeErr     error
}

func (uc *CreateMongodbBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	db *databases.Database,
	fileStore backups_core_logical.BackupFileStore,
	backupProgressListener func(completedMBs float64),
) (*backups_core_logical.BackupArtifacts, error) {
	logger := uc.logger.With("database_id", db.ID, "storage_id", backup.StorageID)

	logger.InfoContext(ctx, "creating mongodb backup via mongodump")

	tunneledDatabase, err := databases.OpenTunnel(ctx, databases.OpenTunnelSpec{
		Database:  db,
		Logger:    logger,
		Encryptor: uc.fieldEncryptor,
	})
	if err != nil {
		return nil, err
	}

	defer tunneledDatabase.Close()

	mongodbDatabase := tunneledDatabase.GetDatabaseThroughTunnel().Mongodb
	if mongodbDatabase == nil {
		return nil, fmt.Errorf("mongodb database configuration is required")
	}

	if mongodbDatabase.Database == "" {
		return nil, fmt.Errorf("database name is required for mongodump backups")
	}

	decryptedPassword, err := uc.fieldEncryptor.Decrypt(mongodbDatabase.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt database password: %w", err)
	}

	rawSizeMB, err := mongodbDatabase.GetRawDbSizeMb(ctx, logger, uc.fieldEncryptor)
	if err != nil {
		logger.WarnContext(ctx, "failed to fetch raw db size before backup", "error", err)
	} else {
		backup.BackupRawDbSizeMb = rawSizeMB
	}

	args := uc.buildMongodumpArgs(mongodbDatabase, decryptedPassword)

	return uc.streamToStorage(
		ctx,
		backup,
		backupConfig,
		tools.GetMongodbExecutable(tools.MongodbExecutableMongodump),
		args,
		fileStore,
		backupProgressListener,
	)
}

func (uc *CreateMongodbBackupUsecase) buildMongodumpArgs(
	mdb *mongodbtypes.MongodbDatabase,
	password string,
) []string {
	uri := mdb.BuildConnectionURI(password)

	args := []string{
		"--uri=" + uri,
		"--archive",
		"--gzip",
	}

	for _, excludedCollection := range namelist.NormalizeUniqueNames(mdb.ExcludeCollections) {
		args = append(args, "--excludeCollection="+excludedCollection)
	}

	// Use numParallelCollections based on CPU count
	// Cap between 1 and 16 to balance performance and resource usage
	parallelCollections := max(1, min(mdb.CpuCount, 16))
	if parallelCollections > 1 {
		args = append(args, "--numParallelCollections="+fmt.Sprintf("%d", parallelCollections))
	}

	return args
}

func (uc *CreateMongodbBackupUsecase) streamToStorage(
	parentCtx context.Context,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	mongodumpBin string,
	args []string,
	fileStore backups_core_logical.BackupFileStore,
	backupProgressListener func(completedMBs float64),
) (*backups_core_logical.BackupArtifacts, error) {
	uc.logger.InfoContext(parentCtx, "streaming MongoDB backup to storage", "mongodump_bin", mongodumpBin)

	ctx, cancel := backups_core_logical.NewBackupContext(parentCtx, config.GetShutdownSignal())
	defer cancel(nil)

	// The binary is a database client from the bundled tools directory, and the arguments go to it as a list, not through a shell.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, mongodumpBin, args...)

	safeArgs := make([]string, len(args))
	for i, arg := range args {
		if len(arg) > 6 && arg[:6] == "--uri=" {
			safeArgs[i] = "--uri=mongodb://***:***@***"
		} else {
			safeArgs[i] = arg
		}
	}
	uc.logger.InfoContext(parentCtx, "executing MongoDB backup command", "command", mongodumpBin, "args", safeArgs)

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
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

	// The dump tool starts before the storage write, so a storage that fails at once cannot
	// cancel the context the tool is started with and hide its own error.
	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", filepath.Base(mongodumpBin), err)
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
		bytesWritten, copyErr := uc.copyWithShutdownCheck(
			ctx,
			finalWriter,
			dumpStdout,
			backupProgressListener,
		)
		bytesWrittenCh <- bytesWritten
		copyResultCh <- copyErr
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

	if waitErr == nil && copyErr == nil && saveErr == nil && backupProgressListener != nil {
		sizeMB := float64(bytesWritten) / (1024 * 1024)
		backupProgressListener(sizeMB)
	}

	switch {
	case waitErr != nil:
		return nil, uc.buildMongodumpErrorMessage(waitErr, stderrOutput.Bytes(), mongodumpBin)
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

func (uc *CreateMongodbBackupUsecase) setupBackupEncryption(
	backupID uuid.UUID,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	storageWriter io.WriteCloser,
) (io.Writer, *backup_encryption.EncryptionWriter, backups_core_logical.BackupMetadata, error) {
	backupMetadata := backups_core_logical.BackupMetadata{
		BackupID:   backupID,
		Encryption: backups_core_enums.BackupEncryptionNone,
	}

	if backupConfig.Encryption != backups_core_enums.BackupEncryptionEncrypted {
		return storageWriter, nil, backupMetadata, nil
	}

	masterKey := uc.masterKey.GetSecret()

	encSetup, err := backup_encryption.SetupEncryptionWriter(storageWriter, masterKey, backupID)
	if err != nil {
		return nil, nil, backupMetadata, err
	}

	backupMetadata.Encryption = backups_core_enums.BackupEncryptionEncrypted
	backupMetadata.EncryptionSalt = &encSetup.SaltBase64
	backupMetadata.EncryptionIV = &encSetup.NonceBase64

	return encSetup.Writer, encSetup.Writer, backupMetadata, nil
}

func (uc *CreateMongodbBackupUsecase) copyWithShutdownCheck(
	ctx context.Context,
	dst io.Writer,
	src io.Reader,
	backupProgressListener func(completedMBs float64),
) (int64, error) {
	buf := make([]byte, copyBufferSize)
	var totalWritten int64
	var lastReportedMB float64

	for {
		select {
		case <-ctx.Done():
			return totalWritten, ctx.Err()
		default:
		}

		nr, readErr := src.Read(buf)
		if nr > 0 {
			writeResultCh := make(chan writeResult, 1)
			go func() {
				nw, writeErr := dst.Write(buf[:nr])
				writeResultCh <- writeResult{nw, writeErr}
			}()

			var nw int
			var writeErr error

			select {
			case <-ctx.Done():
				return totalWritten, fmt.Errorf("copy cancelled during write: %w", ctx.Err())
			case result := <-writeResultCh:
				nw = result.bytesWritten
				writeErr = result.writeErr
			}

			if nw < 0 || nr < nw {
				nw = 0
				if writeErr == nil {
					writeErr = fmt.Errorf("invalid write result")
				}
			}

			if writeErr != nil {
				return totalWritten, writeErr
			}
			if nr != nw {
				return totalWritten, io.ErrShortWrite
			}
			totalWritten += int64(nw)

			if backupProgressListener != nil {
				currentMB := float64(totalWritten) / (1024 * 1024)
				if currentMB-lastReportedMB >= progressReportIntervalMB {
					backupProgressListener(currentMB)
					lastReportedMB = currentMB
				}
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				return totalWritten, nil
			}
			return totalWritten, readErr
		}
	}
}

func (uc *CreateMongodbBackupUsecase) cleanupOnCancellation(
	encryptionWriter *backup_encryption.EncryptionWriter,
	storageWriter *io.PipeWriter,
	writeErrors chan error,
) {
	if encryptionWriter != nil {
		_ = encryptionWriter.Close()
	}
	_ = storageWriter.CloseWithError(errors.New("backup cancelled"))
	<-writeErrors
}

func (uc *CreateMongodbBackupUsecase) closeWriters(
	encryptionWriter *backup_encryption.EncryptionWriter,
	storageWriter *io.PipeWriter,
) error {
	if encryptionWriter != nil {
		if err := encryptionWriter.Close(); err != nil {
			uc.logger.Error("failed to close encryption writer", "error", err)
			return fmt.Errorf("failed to close encryption writer: %w", err)
		}
	}
	if err := storageWriter.Close(); err != nil {
		uc.logger.Error("failed to close storage writer", "error", err)
		return fmt.Errorf("failed to close storage writer: %w", err)
	}
	return nil
}

func (uc *CreateMongodbBackupUsecase) buildMongodumpErrorMessage(
	waitErr error,
	stderrOutput []byte,
	mongodumpBin string,
) error {
	stderrStr := string(stderrOutput)

	if len(stderrStr) > 0 {
		return fmt.Errorf(
			"%s failed: %w\nstderr: %s",
			filepath.Base(mongodumpBin),
			waitErr,
			stderrStr,
		)
	}

	return fmt.Errorf("%s failed: %w", filepath.Base(mongodumpBin), waitErr)
}
