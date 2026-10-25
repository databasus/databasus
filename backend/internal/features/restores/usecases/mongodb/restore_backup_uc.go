package usecases_mongodb

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
	"strings"
	"time"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/features/backups/backups/encryption"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	mongodbtypes "databasus-backend/internal/features/databases/databases/mongodb"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	"databasus-backend/internal/lib/crypto"
	util_encryption "databasus-backend/internal/util/encryption"
	io_utils "databasus-backend/internal/util/io"
	"databasus-backend/internal/util/tools"
)

const (
	restoreTimeout = 23 * time.Hour
)

type RestoreMongodbBackupUsecase struct {
	logger    *slog.Logger
	masterKey *crypto.MasterKey
}

func (uc *RestoreMongodbBackupUsecase) Execute(
	parentCtx context.Context,
	originalDB *databases.Database,
	restoringToDB *databases.Database,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	restore restores_core.Restore,
	backup *backups_core_logical.LogicalBackup,
	storage *storages.Storage,
) error {
	if originalDB.Type != databases.DatabaseTypeMongodb {
		return errors.New("database type not supported")
	}

	uc.logger.InfoContext(parentCtx,
		"restoring MongoDB backup via mongorestore",
		"restore_id", restore.ID,
		"backup_id", backup.ID,
	)

	mdb := restoringToDB.Mongodb
	if mdb == nil {
		return fmt.Errorf("mongodb configuration is required for restore")
	}

	if mdb.Database == "" {
		return fmt.Errorf("target database name is required for mongorestore")
	}

	fieldEncryptor := util_encryption.GetFieldEncryptor()
	decryptedPassword, err := fieldEncryptor.Decrypt(mdb.Password)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	sourceDatabase := ""
	if originalDB.Mongodb != nil {
		sourceDatabase = originalDB.Mongodb.Database
	}

	args := uc.buildMongorestoreArgs(mdb, decryptedPassword, sourceDatabase)

	return uc.restoreFromStorage(
		parentCtx,
		tools.GetMongodbExecutable(tools.MongodbExecutableMongorestore),
		args,
		backup,
		storage,
	)
}

func (uc *RestoreMongodbBackupUsecase) buildMongorestoreArgs(
	mdb *mongodbtypes.MongodbDatabase,
	password string,
	sourceDatabase string,
) []string {
	uri := mdb.BuildRestoreURI(password)

	args := []string{
		"--uri=" + uri,
		"--archive",
		"--gzip",
		"--drop",
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
	parentCtx context.Context,
	mongorestoreBin string,
	args []string,
	backup *backups_core_logical.LogicalBackup,
	storage *storages.Storage,
) error {
	ctx, cancel := context.WithTimeout(parentCtx, restoreTimeout)
	defer cancel()

	logger := uc.logger.With("backup_id", backup.ID)

	fieldEncryptor := util_encryption.GetFieldEncryptor()
	rawReader, err := storage.GetFile(ctx, fieldEncryptor, logger, backup.FileName)
	if err != nil {
		return fmt.Errorf("failed to get backup file from storage: %w", err)
	}
	defer func() {
		if err := rawReader.Close(); err != nil {
			logger.ErrorContext(parentCtx, "failed to close backup reader", "error", err)
		}
	}()

	return uc.executeMongoRestore(ctx, mongorestoreBin, args, rawReader, backup)
}

func (uc *RestoreMongodbBackupUsecase) executeMongoRestore(
	ctx context.Context,
	mongorestoreBin string,
	args []string,
	backupReader io.ReadCloser,
	backup *backups_core_logical.LogicalBackup,
) error {
	// The binary is a database client from the bundled tools directory, and the arguments go to it as a list, not through a shell.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, mongorestoreBin, args...)

	safeArgs := make([]string, len(args))
	for i, arg := range args {
		if len(arg) > 6 && arg[:6] == "--uri=" {
			safeArgs[i] = "--uri=mongodb://***:***@***"
		} else {
			safeArgs[i] = arg
		}
	}
	uc.logger.InfoContext(
		ctx,
		"executing MongoDB restore command",
		"command",
		mongorestoreBin,
		"args",
		safeArgs,
	)

	storageReadFailureTracker := io_utils.NewFailureTrackingReader(backupReader)

	var inputReader io.Reader = storageReadFailureTracker

	if backup.Encryption == backups_core_enums.BackupEncryptionEncrypted {
		decryptReader, err := uc.setupDecryption(storageReadFailureTracker, backup)
		if err != nil {
			return fmt.Errorf("failed to setup decryption: %w", err)
		}
		inputReader = decryptReader
	}

	decodedStreamFailureTracker := io_utils.NewFailureTrackingReader(inputReader)
	cmd.Stdin = decodedStreamFailureTracker

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "LC_ALL=C.UTF-8", "LANG=C.UTF-8")

	var stderrOutput bytes.Buffer
	cmd.Stderr = &stderrOutput
	cmd.WaitDelay = restores_core.ToolOutputWaitDelay

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start mongorestore: %w", err)
	}

	waitErr := cmd.Wait()

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
		return uc.handleMongoRestoreError(waitErr, stderrOutput.Bytes(), mongorestoreBin)
	}

	return nil
}

func (uc *RestoreMongodbBackupUsecase) setupDecryption(
	reader io.Reader,
	backup *backups_core_logical.LogicalBackup,
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

	masterKey := uc.masterKey.GetSecret()

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
