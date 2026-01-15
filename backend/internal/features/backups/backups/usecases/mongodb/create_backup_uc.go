package usecases_mongodb

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	common "databasus-backend/internal/features/backups/backups/common"
	backup_encryption "databasus-backend/internal/features/backups/backups/encryption"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	mongodbtypes "databasus-backend/internal/features/databases/databases/mongodb"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/features/storages"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/tools"
)

const (
	backupTimeout            = 23 * time.Hour
	shutdownCheckInterval    = 1 * time.Second
	copyBufferSize           = 8 * 1024 * 1024
	progressReportIntervalMB = 1.0
)

type CreateMongodbBackupUsecase struct {
	logger           *slog.Logger
	secretKeyService *encryption_secrets.SecretKeyService
	fieldEncryptor   encryption.FieldEncryptor
}

type writeResult struct {
	bytesWritten int
	writeErr     error
}

func (uc *CreateMongodbBackupUsecase) Execute(
	ctx context.Context,
	backupID uuid.UUID,
	backupConfig *backups_config.BackupConfig,
	db *databases.Database,
	storage *storages.Storage,
	backupProgressListener func(completedMBs float64),
) (*common.BackupMetadata, error) {
	uc.logger.Info(
		"Creating MongoDB backup via mongodump",
		"databaseId", db.ID,
		"storageId", storage.ID,
	)

	mdb := db.Mongodb
	if mdb == nil {
		return nil, fmt.Errorf("mongodb database configuration is required")
	}

	if mdb.Database == "" {
		return nil, fmt.Errorf("database name is required for mongodump backups")
	}

	decryptedPassword, err := uc.fieldEncryptor.Decrypt(db.ID, mdb.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt database password: %w", err)
	}

	// Handle TLS certificates if provided
	var certPaths *mongodbtypes.TlsCertPaths
	var cleanupCerts func()
	if mdb.IsHttps && (mdb.TlsCaFile != "" || mdb.TlsCertFile != "" || mdb.TlsCertKeyFile != "") {
		certPaths, cleanupCerts, err = mdb.WriteTempCertificates(uc.fieldEncryptor, db.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare TLS certificates: %w", err)
		}
		defer cleanupCerts()
	}

	args := uc.buildMongodumpArgs(mdb, decryptedPassword, certPaths)

	return uc.streamToStorage(
		ctx,
		backupID,
		backupConfig,
		tools.GetMongodbExecutable(
			tools.MongodbExecutableMongodump,
			config.GetEnv().EnvMode,
			config.GetEnv().MongodbInstallDir,
		),
		args,
		storage,
		backupProgressListener,
	)
}

func (uc *CreateMongodbBackupUsecase) buildMongodumpArgs(
	mdb *mongodbtypes.MongodbDatabase,
	password string,
	certPaths *mongodbtypes.TlsCertPaths,
) []string {
	uri := mdb.BuildMongodumpURI(password)

	args := []string{
		"--uri=" + uri,
		"--db=" + mdb.Database,
		"--archive",
		"--gzip",
	}

	// Add TLS certificate arguments if provided
	// --sslCAFile: CA certificate for server verification
	// --sslPEMKeyFile: Combined client certificate+key for client authentication
	if certPaths != nil {
		if certPaths.CaFile != "" {
			args = append(args, "--sslCAFile="+certPaths.CaFile)
		}
		if certPaths.CertKeyFile != "" {
			args = append(args, "--sslPEMKeyFile="+certPaths.CertKeyFile)
		}
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
	backupID uuid.UUID,
	backupConfig *backups_config.BackupConfig,
	mongodumpBin string,
	args []string,
	storage *storages.Storage,
	backupProgressListener func(completedMBs float64),
) (*common.BackupMetadata, error) {
	uc.logger.Info("Streaming MongoDB backup to storage", "mongodumpBin", mongodumpBin)

	ctx, cancel := uc.createBackupContext(parentCtx)
	defer cancel()

	cmd := exec.CommandContext(ctx, mongodumpBin, args...)

	safeArgs := make([]string, len(args))
	for i, arg := range args {
		if len(arg) > 6 && arg[:6] == "--uri=" {
			safeArgs[i] = "--uri=mongodb://***:***@***"
		} else {
			safeArgs[i] = arg
		}
	}
	uc.logger.Info("Executing MongoDB backup command", "command", mongodumpBin, "args", safeArgs)

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	)

	pgStdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	pgStderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	stderrCh := make(chan []byte, 1)
	go func() {
		stderrOutput, _ := io.ReadAll(pgStderr)
		stderrCh <- stderrOutput
	}()

	storageReader, storageWriter := io.Pipe()

	finalWriter, encryptionWriter, backupMetadata, err := uc.setupBackupEncryption(
		backupID,
		backupConfig,
		storageWriter,
	)
	if err != nil {
		return nil, err
	}

	countingWriter := common.NewCountingWriter(finalWriter)

	saveErrCh := make(chan error, 1)
	go func() {
		saveErr := storage.SaveFile(ctx, uc.fieldEncryptor, uc.logger, backupID, storageReader)
		saveErrCh <- saveErr
	}()

	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", filepath.Base(mongodumpBin), err)
	}

	copyResultCh := make(chan error, 1)
	bytesWrittenCh := make(chan int64, 1)
	go func() {
		bytesWritten, copyErr := uc.copyWithShutdownCheck(
			ctx,
			countingWriter,
			pgStdout,
			backupProgressListener,
		)
		bytesWrittenCh <- bytesWritten
		copyResultCh <- copyErr
	}()

	copyErr := <-copyResultCh
	bytesWritten := <-bytesWrittenCh
	waitErr := cmd.Wait()

	select {
	case <-ctx.Done():
		uc.cleanupOnCancellation(encryptionWriter, storageWriter, saveErrCh)
		return nil, uc.checkCancellationReason()
	default:
	}

	if err := uc.closeWriters(encryptionWriter, storageWriter); err != nil {
		<-saveErrCh
		return nil, err
	}

	saveErr := <-saveErrCh
	stderrOutput := <-stderrCh

	if waitErr == nil && copyErr == nil && saveErr == nil && backupProgressListener != nil {
		sizeMB := float64(bytesWritten) / (1024 * 1024)
		backupProgressListener(sizeMB)
	}

	switch {
	case waitErr != nil:
		return nil, uc.buildMongodumpErrorMessage(waitErr, stderrOutput, mongodumpBin)
	case copyErr != nil:
		return nil, fmt.Errorf("copy to storage: %w", copyErr)
	case saveErr != nil:
		return nil, fmt.Errorf("save to storage: %w", saveErr)
	}

	return &backupMetadata, nil
}

func (uc *CreateMongodbBackupUsecase) createBackupContext(
	parentCtx context.Context,
) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(parentCtx, backupTimeout)

	go func() {
		ticker := time.NewTicker(shutdownCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if config.IsShouldShutdown() {
					cancel()
					return
				}
			}
		}
	}()

	return ctx, cancel
}

func (uc *CreateMongodbBackupUsecase) setupBackupEncryption(
	backupID uuid.UUID,
	backupConfig *backups_config.BackupConfig,
	storageWriter io.WriteCloser,
) (io.Writer, *backup_encryption.EncryptionWriter, common.BackupMetadata, error) {
	backupMetadata := common.BackupMetadata{
		Encryption: backups_config.BackupEncryptionNone,
	}

	if backupConfig.Encryption != backups_config.BackupEncryptionEncrypted {
		return storageWriter, nil, backupMetadata, nil
	}

	salt, err := backup_encryption.GenerateSalt()
	if err != nil {
		return nil, nil, backupMetadata, fmt.Errorf("failed to generate salt: %w", err)
	}

	nonce, err := backup_encryption.GenerateNonce()
	if err != nil {
		return nil, nil, backupMetadata, fmt.Errorf("failed to generate nonce: %w", err)
	}

	masterKey, err := uc.secretKeyService.GetSecretKey()
	if err != nil {
		return nil, nil, backupMetadata, fmt.Errorf("failed to get master key: %w", err)
	}

	encryptionWriter, err := backup_encryption.NewEncryptionWriter(
		storageWriter,
		masterKey,
		backupID,
		salt,
		nonce,
	)
	if err != nil {
		return nil, nil, backupMetadata, fmt.Errorf("failed to create encryption writer: %w", err)
	}

	saltBase64 := base64.StdEncoding.EncodeToString(salt)
	nonceBase64 := base64.StdEncoding.EncodeToString(nonce)

	backupMetadata.Encryption = backups_config.BackupEncryptionEncrypted
	backupMetadata.EncryptionSalt = &saltBase64
	backupMetadata.EncryptionIV = &nonceBase64

	return encryptionWriter, encryptionWriter, backupMetadata, nil
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

		if config.IsShouldShutdown() {
			return totalWritten, errors.New("shutdown requested")
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
	saveErrCh chan error,
) {
	if encryptionWriter != nil {
		_ = encryptionWriter.Close()
	}
	_ = storageWriter.CloseWithError(errors.New("backup cancelled"))
	<-saveErrCh
}

func (uc *CreateMongodbBackupUsecase) closeWriters(
	encryptionWriter *backup_encryption.EncryptionWriter,
	storageWriter *io.PipeWriter,
) error {
	if encryptionWriter != nil {
		if err := encryptionWriter.Close(); err != nil {
			uc.logger.Error("Failed to close encryption writer", "error", err)
			return fmt.Errorf("failed to close encryption writer: %w", err)
		}
	}
	if err := storageWriter.Close(); err != nil {
		uc.logger.Error("Failed to close storage writer", "error", err)
		return fmt.Errorf("failed to close storage writer: %w", err)
	}
	return nil
}

func (uc *CreateMongodbBackupUsecase) checkCancellationReason() error {
	if config.IsShouldShutdown() {
		return errors.New("backup cancelled due to shutdown")
	}
	return errors.New("backup cancelled due to timeout")
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
