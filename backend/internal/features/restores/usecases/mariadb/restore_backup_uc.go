package usecases_mariadb

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
	mariadbtypes "databasus-backend/internal/features/databases/databases/mariadb"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	"databasus-backend/internal/lib/crypto"
	util_encryption "databasus-backend/internal/util/encryption"
	io_utils "databasus-backend/internal/util/io"
	"databasus-backend/internal/util/tools"
)

type RestoreMariadbBackupUsecase struct {
	logger    *slog.Logger
	masterKey *crypto.MasterKey
}

func (uc *RestoreMariadbBackupUsecase) Execute(
	parentCtx context.Context,
	originalDB *databases.Database,
	restoringToDB *databases.Database,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	restore restores_core.Restore,
	backup *backups_core_logical.LogicalBackup,
	storage *storages.Storage,
) error {
	if originalDB.Type != databases.DatabaseTypeMariadb {
		return errors.New("database type not supported")
	}

	uc.logger.InfoContext(parentCtx,
		"restoring MariaDB backup via mariadb client",
		"restore_id", restore.ID,
		"backup_id", backup.ID,
	)

	mdb := restoringToDB.Mariadb
	if mdb == nil {
		return fmt.Errorf("mariadb configuration is required for restore")
	}

	if mdb.Database == nil || *mdb.Database == "" {
		return fmt.Errorf("target database name is required for mariadb restore")
	}

	args := []string{
		"--host=" + mdb.Host,
		"--port=" + strconv.Itoa(mdb.Port),
		"--user=" + mdb.Username,
		"--verbose",
	}

	// Disable Galera Cluster replication for the restore session to prevent
	// "Maximum writeset size exceeded" errors on large restores.
	// wsrep_on is available in MariaDB 10.1+ (all builds with Galera support).
	// On non-Galera instances the variable still exists but is a no-op.
	// Setting it requires the SUPER privilege, so it is skippable for managed
	// clusters that deny SUPER (at the risk of writeset-size errors on big restores).
	if mdb.Version != tools.MariadbVersion55 && !mdb.IsSkipGaleraDisable {
		args = append(args, "--init-command=SET SESSION wsrep_on=OFF")
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

	mariadbBin, err := tools.GetMariadbExecutable(mdb.Version, tools.MariadbExecutableMariadb)
	if err != nil {
		return err
	}

	return uc.restoreFromStorage(
		parentCtx,
		originalDB,
		mariadbBin,
		args,
		mdb.Password,
		backup,
		storage,
		mdb,
	)
}

func (uc *RestoreMariadbBackupUsecase) restoreFromStorage(
	parentCtx context.Context,
	database *databases.Database,
	mariadbBin string,
	args []string,
	password string,
	backup *backups_core_logical.LogicalBackup,
	storage *storages.Storage,
	mdbConfig *mariadbtypes.MariadbDatabase,
) error {
	ctx, cancel := context.WithTimeout(parentCtx, 23*time.Hour)
	defer cancel()

	logger := uc.logger.With("backup_id", backup.ID)

	fieldEncryptor := util_encryption.GetFieldEncryptor()
	decryptedPassword, err := fieldEncryptor.Decrypt(password)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	myCnfFile, err := uc.createTempMyCnfFile(mdbConfig, decryptedPassword)
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

	return uc.executeMariadbRestore(
		ctx,
		database,
		mariadbBin,
		args,
		myCnfFile,
		rawReader,
		backup,
	)
}

func (uc *RestoreMariadbBackupUsecase) executeMariadbRestore(
	ctx context.Context,
	database *databases.Database,
	mariadbBin string,
	args []string,
	myCnfFile string,
	backupReader io.ReadCloser,
	backup *backups_core_logical.LogicalBackup,
) error {
	fullArgs := append([]string{"--defaults-file=" + myCnfFile}, args...)

	// The binary is a database client from the bundled tools directory, and the arguments go to it as a list, not through a shell.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, mariadbBin, fullArgs...)
	uc.logger.InfoContext(ctx, "executing MariaDB restore command", "command", cmd.String())

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
		return fmt.Errorf("start mariadb: %w", err)
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
		return uc.handleMariadbRestoreError(database, waitErr, stderrOutput.Bytes(), mariadbBin)
	}

	return nil
}

func (uc *RestoreMariadbBackupUsecase) setupDecryption(
	reader io.Reader,
	backup *backups_core_logical.LogicalBackup,
) (io.Reader, error) {
	if backup.EncryptionSalt == nil || backup.EncryptionIV == nil {
		return nil, fmt.Errorf("backup is encrypted but missing encryption metadata")
	}

	masterKey := uc.masterKey.GetSecret()

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

func (uc *RestoreMariadbBackupUsecase) createTempMyCnfFile(
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

func (uc *RestoreMariadbBackupUsecase) handleMariadbRestoreError(
	database *databases.Database,
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

	if containsIgnoreCase(stderrStr, "unknown database") {
		backupDbName := "unknown"
		if database.Mariadb != nil && database.Mariadb.Database != nil {
			backupDbName = *database.Mariadb.Database
		}

		return fmt.Errorf(
			"target database does not exist (backup db %s). Create the database before restoring. stderr: %s",
			backupDbName,
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

	if containsIgnoreCase(stderrStr, "writeset size exceeded") {
		return fmt.Errorf(
			"MariaDB Galera Cluster writeset size limit exceeded. Try increasing wsrep_max_ws_size on your cluster nodes. stderr: %s",
			stderrStr,
		)
	}

	return errors.New(errorMsg)
}

func containsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}
