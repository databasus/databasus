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
	"strings"
	"time"

	"databasus-backend/internal/config"
	backups_core "databasus-backend/internal/features/backups/backups/core"
	"databasus-backend/internal/features/backups/backups/encryption"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	mongodbtypes "databasus-backend/internal/features/databases/databases/mongodb"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/features/restores/models"
	"databasus-backend/internal/features/storages"
	util_encryption "databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/tools"
)

const (
	restoreTimeout = 23 * time.Hour
)

type RestoreMongodbBackupUsecase struct {
	logger           *slog.Logger
	secretKeyService *encryption_secrets.SecretKeyService
}

func (uc *RestoreMongodbBackupUsecase) Execute(
	originalDB *databases.Database,
	restoringToDB *databases.Database,
	backupConfig *backups_config.BackupConfig,
	restore models.Restore,
	backup *backups_core.Backup,
	storage *storages.Storage,
) error {
	if originalDB.Type != databases.DatabaseTypeMongodb {
		return errors.New("database type not supported")
	}

	uc.logger.Info(
		"Restoring MongoDB backup via mongorestore",
		"restoreId", restore.ID,
		"backupId", backup.ID,
	)

	mdb := restoringToDB.Mongodb
	if mdb == nil {
		return fmt.Errorf("mongodb configuration is required for restore")
	}

	if mdb.Database == "" {
		return fmt.Errorf("target database name is required for mongorestore")
	}

	fieldEncryptor := util_encryption.GetFieldEncryptor()
	decryptedPassword, err := fieldEncryptor.Decrypt(restoringToDB.ID, mdb.Password)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	// Handle TLS certificates if provided
	var certPaths *mongodbtypes.TlsCertPaths
	var cleanupCerts func()
	if mdb.IsHttps && (mdb.TlsCaFile != "" || mdb.TlsCertFile != "" || mdb.TlsCertKeyFile != "") {
		certPaths, cleanupCerts, err = mdb.WriteTempCertificates(fieldEncryptor, restoringToDB.ID)
		if err != nil {
			return fmt.Errorf("failed to prepare TLS certificates: %w", err)
		}
		defer cleanupCerts()
	}

	sourceDatabase := ""
	if originalDB.Mongodb != nil {
		sourceDatabase = originalDB.Mongodb.Database
	}

	args := uc.buildMongorestoreArgs(mdb, decryptedPassword, sourceDatabase, certPaths)

	return uc.restoreFromStorage(
		tools.GetMongodbExecutable(
			tools.MongodbExecutableMongorestore,
			config.GetEnv().EnvMode,
			config.GetEnv().MongodbInstallDir,
		),
		args,
		backup,
		storage,
	)
}

func (uc *RestoreMongodbBackupUsecase) buildMongorestoreArgs(
	mdb *mongodbtypes.MongodbDatabase,
	password string,
	sourceDatabase string,
	certPaths *mongodbtypes.TlsCertPaths,
) []string {
	uri := mdb.BuildMongodumpURI(password)

	args := []string{
		"--uri=" + uri,
		"--archive",
		"--gzip",
		"--drop",
	}

	// Add TLS certificate arguments if provided
	if certPaths != nil {
		if certPaths.CaFile != "" {
			args = append(args, "--tlsCAFile="+certPaths.CaFile)
		}
		if certPaths.CertFile != "" && certPaths.CertKeyFile != "" {
			args = append(args, "--tlsCertificateKeyFile="+certPaths.CertFile)
		}
	}

	if sourceDatabase != "" && sourceDatabase != mdb.Database {
		args = append(args, "--nsFrom="+sourceDatabase+".*")
		args = append(args, "--nsTo="+mdb.Database+".*")
	} else if mdb.Database != "" {
		args = append(args, "--nsInclude="+mdb.Database+".*")
	}

	// Use numInsertionWorkersPerCollection based on CPU count
	// Cap between 1 and 16 to balance performance and resource usage
	parallelWorkers := max(1, min(mdb.CpuCount, 16))
	if parallelWorkers > 1 {
		args = append(
			args,
			"--numInsertionWorkersPerCollection="+fmt.Sprintf("%d", parallelWorkers),
		)
	}

	return args
}

func (uc *RestoreMongodbBackupUsecase) restoreFromStorage(
	mongorestoreBin string,
	args []string,
	backup *backups_core.Backup,
	storage *storages.Storage,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), restoreTimeout)
	defer cancel()

	go func() {
		ticker := time.NewTicker(1 * time.Second)
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

	// Stream backup directly from storage
	fieldEncryptor := util_encryption.GetFieldEncryptor()
	rawReader, err := storage.GetFile(fieldEncryptor, backup.ID)
	if err != nil {
		return fmt.Errorf("failed to get backup file from storage: %w", err)
	}
	defer func() {
		if err := rawReader.Close(); err != nil {
			uc.logger.Error("Failed to close backup reader", "error", err)
		}
	}()

	return uc.executeMongoRestore(ctx, mongorestoreBin, args, rawReader, backup)
}

func (uc *RestoreMongodbBackupUsecase) executeMongoRestore(
	ctx context.Context,
	mongorestoreBin string,
	args []string,
	backupReader io.ReadCloser,
	backup *backups_core.Backup,
) error {
	cmd := exec.CommandContext(ctx, mongorestoreBin, args...)

	safeArgs := make([]string, len(args))
	for i, arg := range args {
		if len(arg) > 6 && arg[:6] == "--uri=" {
			safeArgs[i] = "--uri=mongodb://***:***@***"
		} else {
			safeArgs[i] = arg
		}
	}
	uc.logger.Info(
		"Executing MongoDB restore command",
		"command",
		mongorestoreBin,
		"args",
		safeArgs,
	)

	var inputReader io.Reader = backupReader

	if backup.Encryption == backups_config.BackupEncryptionEncrypted {
		decryptReader, err := uc.setupDecryption(backupReader, backup)
		if err != nil {
			return fmt.Errorf("failed to setup decryption: %w", err)
		}
		inputReader = decryptReader
	}

	cmd.Stdin = inputReader
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "LC_ALL=C.UTF-8", "LANG=C.UTF-8")

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	stderrCh := make(chan []byte, 1)
	go func() {
		output, _ := io.ReadAll(stderrPipe)
		stderrCh <- output
	}()

	if err = cmd.Start(); err != nil {
		return fmt.Errorf("start mongorestore: %w", err)
	}

	waitErr := cmd.Wait()
	stderrOutput := <-stderrCh

	if config.IsShouldShutdown() {
		return fmt.Errorf("restore cancelled due to shutdown")
	}

	if waitErr != nil {
		return uc.handleMongoRestoreError(waitErr, stderrOutput, mongorestoreBin)
	}

	return nil
}

func (uc *RestoreMongodbBackupUsecase) setupDecryption(
	reader io.Reader,
	backup *backups_core.Backup,
) (io.Reader, error) {
	if backup.EncryptionSalt == nil || backup.EncryptionIV == nil {
		return nil, errors.New("encrypted backup missing salt or IV")
	}

	salt, err := base64.StdEncoding.DecodeString(*backup.EncryptionSalt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption salt: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(*backup.EncryptionIV)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption IV: %w", err)
	}

	masterKey, err := uc.secretKeyService.GetSecretKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get secret key: %w", err)
	}

	decryptReader, err := encryption.NewDecryptionReader(
		reader,
		masterKey,
		backup.ID,
		salt,
		nonce,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create decryption reader: %w", err)
	}

	return decryptReader, nil
}

func (uc *RestoreMongodbBackupUsecase) handleMongoRestoreError(
	waitErr error,
	stderrOutput []byte,
	mongorestoreBin string,
) error {
	stderrStr := string(stderrOutput)

	if containsIgnoreCase(stderrStr, "authentication failed") {
		return fmt.Errorf(
			"MongoDB authentication failed. Check username and password. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "connection refused") ||
		containsIgnoreCase(stderrStr, "server selection error") {
		return fmt.Errorf(
			"MongoDB connection refused. Check if the server is running and accessible. stderr: %s",
			stderrStr,
		)
	}

	if containsIgnoreCase(stderrStr, "timeout") {
		return fmt.Errorf(
			"MongoDB connection timeout. stderr: %s",
			stderrStr,
		)
	}

	if len(stderrStr) > 0 {
		return fmt.Errorf(
			"%s failed: %w\nstderr: %s",
			filepath.Base(mongorestoreBin),
			waitErr,
			stderrStr,
		)
	}

	return fmt.Errorf("%s failed: %w", filepath.Base(mongorestoreBin), waitErr)
}

func containsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}
