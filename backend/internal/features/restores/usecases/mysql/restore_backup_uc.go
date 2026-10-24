package usecases_mysql

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
	"github.com/klauspost/compress/zstd"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/features/backups/backups/encryption"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	mysqltypes "databasus-backend/internal/features/databases/databases/mysql"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	util_encryption "databasus-backend/internal/util/encryption"
	io_utils "databasus-backend/internal/util/io"
	"databasus-backend/internal/util/tools"
)

type RestoreMysqlBackupUsecase struct {
	logger           *slog.Logger
	secretKeyService *encryption_secrets.SecretKeyService
}

func (uc *RestoreMysqlBackupUsecase) Execute(
	parentCtx context.Context,
	originalDB *databases.Database,
	restoringToDB *databases.Database,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	restore restores_core.Restore,
	backup *backups_core_logical.LogicalBackup,
	storage *storages.Storage,
) error {
	if originalDB.Type != databases.DatabaseTypeMysql {
		return errors.New("database type not supported")
	}

	uc.logger.InfoContext(parentCtx,
		"restoring MySQL backup via mysql client",
		"restore_id", restore.ID,
		"backup_id", backup.ID,
	)

	my := restoringToDB.Mysql
	if my == nil {
		return fmt.Errorf("mysql configuration is required for restore")
	}

	if my.Database == nil || *my.Database == "" {
		return fmt.Errorf("target database name is required for mysql restore")
	}

	args := []string{
		"--host=" + my.Host,
		"--port=" + strconv.Itoa(my.Port),
		"--user=" + my.Username,
		"--verbose",
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

	mysqlBin, err := tools.GetMysqlExecutable(my.Version, tools.MysqlExecutableMysql)
	if err != nil {
		return err
	}

	return uc.restoreFromStorage(
		parentCtx,
		originalDB,
		mysqlBin,
		args,
		my.Password,
		backup,
		storage,
		my,
	)
}

func (uc *RestoreMysqlBackupUsecase) restoreFromStorage(
	parentCtx context.Context,
	database *databases.Database,
	mysqlBin string,
	args []string,
	password string,
	backup *backups_core_logical.LogicalBackup,
	storage *storages.Storage,
	myConfig *mysqltypes.MysqlDatabase,
) error {
	ctx, cancel := context.WithTimeout(parentCtx, 23*time.Hour)
	defer cancel()

	logger := uc.logger.With("backup_id", backup.ID)

	fieldEncryptor := util_encryption.GetFieldEncryptor()
	decryptedPassword, err := fieldEncryptor.Decrypt(password)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	myCnfFile, err := uc.createTempMyCnfFile(myConfig, decryptedPassword)
	if err != nil {
		return fmt.Errorf("failed to create .my.cnf: %w", err)
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(myCnfFile)) }()

	rawReader, err := storage.GetFile(ctx, fieldEncryptor, logger, backup.FileName)
	if err != nil {
		return fmt.Errorf("failed to get backup file from storage: %w", err)
	}
	defer func() {
		if err := rawReader.Close(); err != nil {
			logger.ErrorContext(parentCtx, "failed to close backup reader", "error", err)
		}
	}()

	return uc.executeMysqlRestore(ctx, database, mysqlBin, args, myCnfFile, rawReader, backup)
}

func (uc *RestoreMysqlBackupUsecase) executeMysqlRestore(
	ctx context.Context,
	database *databases.Database,
	mysqlBin string,
	args []string,
	myCnfFile string,
	backupReader io.ReadCloser,
	backup *backups_core_logical.LogicalBackup,
) error {
	fullArgs := append([]string{"--defaults-file=" + myCnfFile}, args...)

	// The binary is a database client from the bundled tools directory, and the arguments go to it as a list, not through a shell.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, mysqlBin, fullArgs...)
	uc.logger.InfoContext(ctx, "executing MySQL restore command", "command", cmd.String())

	storageReadFailureTracker := io_utils.NewFailureTrackingReader(backupReader)

	var inputReader io.Reader = storageReadFailureTracker

	if backup.Encryption == backups_core_enums.BackupEncryptionEncrypted {
		decryptReader, err := uc.setupDecryption(storageReadFailureTracker, backup)
		if err != nil {
			return fmt.Errorf("failed to setup decryption: %w", err)
		}
		inputReader = decryptReader
	}

	zstdReader, err := zstd.NewReader(inputReader)
	if err != nil {
		return fmt.Errorf("failed to create zstd reader: %w", err)
	}
	defer zstdReader.Close()

	decodedStreamFailureTracker := io_utils.NewFailureTrackingReader(zstdReader)
	cmd.Stdin = decodedStreamFailureTracker

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
		"MYSQL_PWD=",
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	)

	var stderrOutput bytes.Buffer
	cmd.Stderr = &stderrOutput
	cmd.WaitDelay = restores_core.ToolOutputWaitDelay

	if err = cmd.Start(); err != nil {
		return fmt.Errorf("start mysql: %w", err)
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
		return uc.handleMysqlRestoreError(database, waitErr, stderrOutput.Bytes(), mysqlBin)
	}

	return nil
}

func (uc *RestoreMysqlBackupUsecase) setupDecryption(
	reader io.Reader,
	backup *backups_core_logical.LogicalBackup,
) (io.Reader, error) {
	if backup.EncryptionSalt == nil || backup.EncryptionIV == nil {
		return nil, fmt.Errorf("backup is encrypted but missing encryption metadata")
	}

	masterKey, err := uc.secretKeyService.GetSecretKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get master key for decryption: %w", err)
	}

	salt, err := base64.StdEncoding.DecodeString(*backup.EncryptionSalt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption salt: %w", err)
	}

	iv, err := base64.StdEncoding.DecodeString(*backup.EncryptionIV)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption IV: %w", err)
	}

	decryptReader, err := encryption.NewDecryptionReader(
		reader,
		masterKey,
		backup.ID,
		salt,
		iv,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create decryption reader: %w", err)
	}

	uc.logger.Info("using decryption for encrypted backup", "backup_id", backup.ID)
	return decryptReader, nil
}

func (uc *RestoreMysqlBackupUsecase) createTempMyCnfFile(
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

func (uc *RestoreMysqlBackupUsecase) handleMysqlRestoreError(
	database *databases.Database,
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

	if containsIgnoreCase(stderrStr, "unknown database") {
		backupDbName := "unknown"
		if database.Mysql != nil && database.Mysql.Database != nil {
			backupDbName = *database.Mysql.Database
		}

		return fmt.Errorf(
			"target database does not exist (backup db %s). Create the database before restoring. stderr: %s",
			backupDbName,
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

	return errors.New(errorMsg)
}

func containsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}
