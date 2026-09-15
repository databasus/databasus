package mysql

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/features/databases/databases/mysqlfamily"
	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/namelist"
	"databasus-backend/internal/util/tools"
)

type MysqlDatabase struct {
	ID         uuid.UUID  `json:"id"         gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	DatabaseID *uuid.UUID `json:"databaseId" gorm:"type:uuid;column:database_id"`

	Version tools.MysqlVersion `json:"version" gorm:"type:text;not null"`

	Host     string  `json:"host"     gorm:"type:text;not null"`
	Port     int     `json:"port"     gorm:"type:int;not null"`
	Username string  `json:"username" gorm:"type:text;not null"`
	Password string  `json:"password" gorm:"type:text;not null"`
	Database *string `json:"database" gorm:"type:text"`
	IsHttps  bool    `json:"isHttps"  gorm:"type:boolean;default:false"`

	// When the tunnel is enabled, Host and Port above address the database as the bastion sees it.
	SshTunnel sshtunnel.Config `json:"sshTunnel" gorm:"embedded;embeddedPrefix:ssh_"`

	ExcludeTables       []string `json:"excludeTables" gorm:"-"`
	ExcludeTablesString string   `json:"-"             gorm:"column:exclude_tables;type:text;not null;default:''"`
	Privileges          string   `json:"privileges"    gorm:"column:privileges;type:text;not null;default:''"`
}

func (m *MysqlDatabase) TableName() string {
	return "mysql_databases"
}

func (m *MysqlDatabase) BeforeSave(_ *gorm.DB) error {
	m.ExcludeTablesString = namelist.FormatUniqueNames(m.ExcludeTables)

	return nil
}

func (m *MysqlDatabase) AfterFind(_ *gorm.DB) error {
	m.ExcludeTables = namelist.ParseUniqueNames(m.ExcludeTablesString)

	return nil
}

func (m *MysqlDatabase) Validate() error {
	if m.Host == "" {
		return errors.New("host is required")
	}
	if m.Port == 0 {
		return errors.New("port is required")
	}
	if m.Username == "" {
		return errors.New("username is required")
	}
	if m.Password == "" {
		return errors.New("password is required")
	}

	return m.SshTunnel.Validate()
}

func (m *MysqlDatabase) TestConnection(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if m.Database == nil || *m.Database == "" {
		return errors.New("database name is required for MySQL backup")
	}

	password, err := decryptPasswordIfNeeded(m.Password, encryptor)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	dsn := m.buildDSN(password, *m.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		// The DSN carries the password, so only the database name goes into the log.
		logger.ErrorContext(ctx, "failed to open the mysql connection", "database_name", *m.Database, "error", err)

		return fmt.Errorf("failed to connect to MySQL database '%s': %w", *m.Database, err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.ErrorContext(ctx, "failed to close the mysql connection", "error", closeErr)
		}
	}()

	db.SetConnMaxLifetime(15 * time.Second)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		logger.ErrorContext(ctx, "failed to ping the mysql database", "database_name", *m.Database, "error", err)

		return fmt.Errorf("failed to ping MySQL database '%s': %w", *m.Database, err)
	}

	detectedVersion, err := detectMysqlVersion(ctx, db)
	if err != nil {
		return err
	}
	m.Version = detectedVersion

	backupPrivileges, err := readBackupPrivileges(ctx, backupPrivilegesQuery{
		DB:            db,
		Logger:        logger,
		Version:       m.Version,
		SchemaName:    *m.Database,
		ExcludeTables: m.ExcludeTables,
	})
	if err != nil {
		return err
	}
	m.Privileges = backupPrivileges.GetEffectivePrivilegesCsv()

	if !backupPrivileges.IsSufficientForDump() {
		return backupPrivileges.NewInsufficiencyError()
	}

	return nil
}

func (m *MysqlDatabase) GetRawDbSizeMb(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (float64, error) {
	if m.Database == nil || *m.Database == "" {
		return 0, nil
	}

	password, err := decryptPasswordIfNeeded(m.Password, encryptor)
	if err != nil {
		return 0, fmt.Errorf("failed to decrypt password: %w", err)
	}

	dsn := m.buildDSN(password, *m.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to MySQL database '%s': %w", *m.Database, err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.ErrorContext(ctx, "failed to close MySQL connection", "error", closeErr)
		}
	}()

	db.SetConnMaxLifetime(15 * time.Second)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	mysqlConnection, err := db.Conn(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to acquire MySQL connection: %w", err)
	}
	defer func() {
		if closeErr := mysqlConnection.Close(); closeErr != nil {
			logger.ErrorContext(ctx, "failed to release MySQL connection", "error", closeErr)
		}
	}()

	if m.Version != tools.MysqlVersion57 {
		_, err = mysqlConnection.ExecContext(ctx, "SET SESSION information_schema_stats_expiry = 0")
		if err != nil {
			return 0, fmt.Errorf("failed to disable MySQL statistics cache: %w", err)
		}
	}

	const query = `
		SELECT COALESCE(SUM(data_length + index_length), 0) / (1024 * 1024)
		FROM information_schema.tables
		WHERE table_schema = ?
	`

	var sizeMB float64
	if err := mysqlConnection.QueryRowContext(ctx, query, *m.Database).Scan(&sizeMB); err != nil {
		return 0, fmt.Errorf("failed to query MySQL database size: %w", err)
	}

	return sizeMB, nil
}

func (m *MysqlDatabase) HideSensitiveData() {
	if m == nil {
		return
	}
	m.Password = ""
	m.SshTunnel.HideSensitiveData()
}

func (m *MysqlDatabase) Update(incoming *MysqlDatabase) {
	m.Version = incoming.Version
	m.Host = incoming.Host
	m.Port = incoming.Port
	m.Username = incoming.Username
	m.Database = incoming.Database
	m.IsHttps = incoming.IsHttps
	m.ExcludeTables = incoming.ExcludeTables
	m.Privileges = incoming.Privileges
	m.SshTunnel.Update(&incoming.SshTunnel)

	if incoming.Password != "" {
		m.Password = incoming.Password
	}
}

func (m *MysqlDatabase) CopyForNewDatabase() *MysqlDatabase {
	if m == nil {
		return nil
	}

	copiedDatabase := *m
	copiedDatabase.ID = uuid.Nil
	copiedDatabase.DatabaseID = nil
	copiedDatabase.ExcludeTables = slices.Clone(m.ExcludeTables)

	if m.Database != nil {
		copiedDatabase.Database = new(*m.Database)
	}

	return &copiedDatabase
}

func (m *MysqlDatabase) EncryptSensitiveFields(
	encryptor encryption.FieldEncryptor,
) error {
	if m.Password != "" {
		encrypted, err := encryptor.Encrypt(m.Password)
		if err != nil {
			return err
		}
		m.Password = encrypted
	}

	return m.SshTunnel.EncryptSensitiveFields(encryptor)
}

func (m *MysqlDatabase) PopulateDbData(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	if m.Database == nil || *m.Database == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	password, err := decryptPasswordIfNeeded(m.Password, encryptor)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	dsn := m.buildDSN(password, *m.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.ErrorContext(ctx, "failed to close the connection", "error", closeErr)
		}
	}()

	detectedVersion, err := detectMysqlVersion(ctx, db)
	if err != nil {
		return err
	}
	m.Version = detectedVersion

	backupPrivileges, err := readBackupPrivileges(ctx, backupPrivilegesQuery{
		DB:            db,
		Logger:        logger,
		Version:       m.Version,
		SchemaName:    *m.Database,
		ExcludeTables: m.ExcludeTables,
	})
	if err != nil {
		return err
	}
	m.Privileges = backupPrivileges.GetEffectivePrivilegesCsv()

	return nil
}

func (m *MysqlDatabase) PopulateVersion(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	if m.Database == nil || *m.Database == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	password, err := decryptPasswordIfNeeded(m.Password, encryptor)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	dsn := m.buildDSN(password, *m.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.ErrorContext(ctx, "failed to close the connection", "error", closeErr)
		}
	}()

	detectedVersion, err := detectMysqlVersion(ctx, db)
	if err != nil {
		return err
	}
	m.Version = detectedVersion

	return nil
}

func (m *MysqlDatabase) ShouldSuggestReadOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (bool, []string, error) {
	password, err := decryptPasswordIfNeeded(m.Password, encryptor)
	if err != nil {
		return false, nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	dsn := m.buildDSN(password, *m.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return false, nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.ErrorContext(ctx, "failed to close connection", "error", closeErr)
		}
	}()

	rows, err := db.QueryContext(ctx, "SHOW GRANTS FOR CURRENT_USER()")
	if err != nil {
		return false, nil, fmt.Errorf("failed to check grants: %w", err)
	}
	defer func() { _ = rows.Close() }()

	writePrivileges := map[string]bool{
		"INSERT":            true,
		"UPDATE":            true,
		"DELETE":            true,
		"CREATE":            true,
		"DROP":              true,
		"ALTER":             true,
		"INDEX":             true,
		"GRANT OPTION":      true,
		"ALL PRIVILEGES":    true,
		"SUPER":             true,
		"EXECUTE":           true,
		"FILE":              true,
		"RELOAD":            true,
		"SHUTDOWN":          true,
		"CREATE ROUTINE":    true,
		"ALTER ROUTINE":     true,
		"CREATE USER":       true,
		"CREATE TABLESPACE": true,
		"REFERENCES":        true,
	}

	detectedPrivileges := make(map[string]bool)

	for rows.Next() {
		var grant string
		if err := rows.Scan(&grant); err != nil {
			return false, nil, fmt.Errorf("failed to scan grant: %w", err)
		}

		parsedGrant := mysqlfamily.ParseGrantLine(grant)
		if parsedGrant == nil {
			continue
		}

		for _, privilege := range parsedGrant.Privileges {
			if writePrivileges[privilege] {
				detectedPrivileges[privilege] = true
			}
		}
	}

	if err := rows.Err(); err != nil {
		return false, nil, fmt.Errorf("error iterating grants: %w", err)
	}

	privileges := make([]string, 0, len(detectedPrivileges))
	for priv := range detectedPrivileges {
		privileges = append(privileges, priv)
	}

	shouldSuggestReadOnlyUser := len(privileges) > 0

	return shouldSuggestReadOnlyUser, privileges, nil
}

func (m *MysqlDatabase) CreateReadOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (string, string, error) {
	password, err := decryptPasswordIfNeeded(m.Password, encryptor)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	dsn := m.buildDSN(password, *m.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.ErrorContext(ctx, "failed to close connection", "error", closeErr)
		}
	}()

	maxRetries := 3
	for attempt := range maxRetries {
		newUsername := fmt.Sprintf("databasus-%s", uuid.New().String()[:8])
		newPassword := encryption.GenerateComplexPassword()

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return "", "", fmt.Errorf("failed to begin transaction: %w", err)
		}

		success := false
		defer func() {
			if !success {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					logger.ErrorContext(ctx, "failed to rollback transaction", "error", rollbackErr)
				}
			}
		}()

		_, err = tx.ExecContext(ctx, fmt.Sprintf(
			"CREATE USER '%s'@'%%' IDENTIFIED BY '%s'",
			newUsername,
			newPassword,
		))
		if err != nil {
			if attempt < maxRetries-1 {
				continue
			}
			return "", "", fmt.Errorf("failed to create user: %w", err)
		}

		_, err = tx.ExecContext(ctx, fmt.Sprintf(
			"GRANT SELECT, SHOW VIEW, LOCK TABLES, TRIGGER, EVENT ON `%s`.* TO '%s'@'%%'",
			*m.Database,
			newUsername,
		))
		if err != nil {
			return "", "", fmt.Errorf("failed to grant database privileges: %w", err)
		}

		_, err = tx.ExecContext(ctx, "FLUSH PRIVILEGES")
		if err != nil {
			return "", "", fmt.Errorf("failed to flush privileges: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return "", "", fmt.Errorf("failed to commit transaction: %w", err)
		}

		success = true
		logger.InfoContext(
			ctx,
			"read-only MySQL user created successfully",
			"username",
			newUsername,
		)
		return newUsername, newPassword, nil
	}

	return "", "", errors.New("failed to generate unique username after 3 attempts")
}

func (m *MysqlDatabase) buildDSN(password, database string) string {
	tlsConfig := "false"
	allowCleartext := ""

	if m.IsHttps {
		err := mysql.RegisterTLSConfig("mysql-skip-verify", &tls.Config{
			InsecureSkipVerify: true,
		})
		if err != nil {
			// Config might already be registered, which is fine
			_ = err
		}

		tlsConfig = "mysql-skip-verify"
		allowCleartext = "&allowCleartextPasswords=1"
	}

	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=15s&tls=%s&charset=utf8mb4%s",
		m.Username,
		password,
		m.Host,
		m.Port,
		database,
		tlsConfig,
		allowCleartext,
	)
}

// detectMysqlVersion parses VERSION() output to detect MySQL version
// Minor versions are mapped to the closest supported version (e.g., 8.1 → 8.0, 8.4+ → 8.4)
func detectMysqlVersion(ctx context.Context, db *sql.DB) (tools.MysqlVersion, error) {
	var versionStr string
	err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&versionStr)
	if err != nil {
		return "", fmt.Errorf("failed to query MySQL version: %w", err)
	}

	re := regexp.MustCompile(`^(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 3 {
		return "", fmt.Errorf("could not parse MySQL version: %s", versionStr)
	}

	major := matches[1]
	minor := matches[2]

	return mapMysqlVersion(major, minor)
}

// Calendar-versioned releases (26.7 and later) continue the 9.x line and use the 9.x client.
func mapMysqlVersion(major, minor string) (tools.MysqlVersion, error) {
	majorNum, err := strconv.Atoi(major)
	if err != nil {
		return "", fmt.Errorf("could not parse MySQL major version: %s", major)
	}

	switch {
	case majorNum == 5:
		return tools.MysqlVersion57, nil
	case majorNum == 8:
		return mapMysql8xVersion(minor), nil
	case majorNum >= 9:
		return tools.MysqlVersion9, nil
	default:
		return "", fmt.Errorf(
			"unsupported MySQL major version: %s (supported: 5.x, 8.x, 9.x and later)",
			major,
		)
	}
}

func mapMysql8xVersion(minor string) tools.MysqlVersion {
	switch minor {
	case "0", "1", "2", "3":
		return tools.MysqlVersion80
	default:
		return tools.MysqlVersion84
	}
}

func decryptPasswordIfNeeded(
	password string,
	encryptor encryption.FieldEncryptor,
) (string, error) {
	if encryptor == nil {
		return password, nil
	}
	return encryptor.Decrypt(password)
}
