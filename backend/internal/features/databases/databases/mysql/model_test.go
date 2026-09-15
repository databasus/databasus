package mysql

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/tools"
)

type mysqlModelVersion struct {
	name     string
	version  tools.MysqlVersion
	image    string
	hasRoles bool
}

var mysqlModelVersions = []mysqlModelVersion{
	{"MySQL 5.7", tools.MysqlVersion57, "mysql:5.7", false},
	{"MySQL 8.0", tools.MysqlVersion80, "mysql:8.0", true},
	{"MySQL 8.4", tools.MysqlVersion84, "mysql:8.4", true},
	{"MySQL 9", tools.MysqlVersion9, "mysql:9", true},
}

// Test_MysqlModel_AcrossSupportedVersions boots each MySQL version once and runs every matrix model
// test against it as a subtest. Only one container is alive per package at a time. See ADR-0013.
func Test_MysqlModel_AcrossSupportedVersions(t *testing.T) {
	for _, dbVersion := range mysqlModelVersions {
		t.Run(dbVersion.name, func(t *testing.T) {
			endpoint := containers.StartMysql(t, dbVersion.image)

			t.Run("Test_TestConnection_InsufficientPermissions_ReturnsError", func(t *testing.T) {
				testTestConnectionInsufficientPermissions(t, endpoint, dbVersion.version)
			})

			t.Run("Test_TestConnection_SufficientPermissions_Success", func(t *testing.T) {
				testTestConnectionSufficientPermissions(t, endpoint, dbVersion.version)
			})

			t.Run("Test_ShouldSuggestReadOnlyUser_AdminUser_ReturnsTrue", func(t *testing.T) {
				testShouldSuggestReadOnlyUserAdminUser(t, endpoint, dbVersion.version)
			})

			t.Run("Test_CreateReadOnlyUser_UserCanReadButNotWrite", func(t *testing.T) {
				testCreateReadOnlyUserCanReadButNotWrite(t, endpoint, dbVersion.version)
			})

			t.Run("Test_TestConnection_DatabaseSpecificPrivilegesWithGlobalProcess_Success", func(t *testing.T) {
				testTestConnectionDatabaseSpecificPrivilegesWithGlobalProcess(t, endpoint, dbVersion.version)
			})

			t.Run("Test_TestConnection_DatabaseWithUnderscoresAndAllPrivileges_Success", func(t *testing.T) {
				testTestConnectionDatabaseWithUnderscoresAndAllPrivileges(t, endpoint, dbVersion.version)
			})

			t.Run("Test_TestConnection_WhenTableGrantsCoverEveryVisibleTable_Success", func(t *testing.T) {
				testTestConnectionTableGrantsCoverEveryVisibleTable(t, endpoint, dbVersion.version)
			})

			t.Run("Test_TestConnection_WhenVisibleTableLacksSelect_ReturnsError", func(t *testing.T) {
				testTestConnectionVisibleTableLacksSelect(t, endpoint, dbVersion.version)
			})

			t.Run("Test_TestConnection_WhenVisibleTableLackingSelectIsExcluded_Success", func(t *testing.T) {
				testTestConnectionVisibleTableLackingSelectIsExcluded(t, endpoint, dbVersion.version)
			})

			t.Run("Test_TestConnection_WhenViewLacksShowView_ReturnsErrorUntilShowViewIsGranted", func(t *testing.T) {
				testTestConnectionViewLacksShowView(t, endpoint, dbVersion.version)
			})

			t.Run("Test_GetRawDbSizeMb_Mysql_ReturnsPositiveSize", func(t *testing.T) {
				testGetRawDbSizeMbReturnsPositiveSize(t, endpoint, dbVersion.version)
			})

			if dbVersion.version == tools.MysqlVersion80 {
				t.Run("Test_GetRawDbSizeMb_WhenMysqlStatisticsCacheIsWarm_ReturnsCurrentSize", func(t *testing.T) {
					testGetRawDbSizeMbWhenMysqlStatisticsCacheIsWarmReturnsCurrentSize(t, endpoint)
				})
			}

			if dbVersion.hasRoles {
				t.Run("Test_TestConnection_WhenBackupPrivilegesComeFromActiveRole_Success", func(t *testing.T) {
					testTestConnectionBackupPrivilegesFromActiveRole(t, endpoint, dbVersion.version)
				})

				t.Run("Test_TestConnection_WhenRoleIsGrantedButNotActivated_ReturnsError", func(t *testing.T) {
					testTestConnectionRoleGrantedButNotActivated(t, endpoint, dbVersion.version)
				})
			}
		})
	}
}

func testTestConnectionInsufficientPermissions(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MysqlVersion,
) {
	container := connectToMysqlEndpoint(t, endpoint, version)
	defer container.DB.Close()

	_, err := container.DB.Exec(`DROP TABLE IF EXISTS permission_test`)
	assert.NoError(t, err)

	_, err = container.DB.Exec(`CREATE TABLE permission_test (
				id INT AUTO_INCREMENT PRIMARY KEY,
				data VARCHAR(255) NOT NULL
			)`)
	assert.NoError(t, err)

	_, err = container.DB.Exec(`INSERT INTO permission_test (data) VALUES ('test1')`)
	assert.NoError(t, err)

	limitedUsername := fmt.Sprintf("limited_%s", uuid.New().String()[:8])
	limitedPassword := "limitedpassword123"

	_, err = container.DB.Exec(fmt.Sprintf(
		"CREATE USER '%s'@'%%' IDENTIFIED BY '%s'",
		limitedUsername,
		limitedPassword,
	))
	assert.NoError(t, err)

	// INSERT makes the schema and its tables visible without making them readable, which is what
	// the dump tool aborts on.
	_, err = container.DB.Exec(fmt.Sprintf(
		"GRANT INSERT ON `%s`.* TO '%s'@'%%'",
		container.Database,
		limitedUsername,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec("FLUSH PRIVILEGES")
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(
			fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", limitedUsername),
		)
	}()

	mysqlModel := &MysqlDatabase{
		Version:  version,
		Host:     container.Host,
		Port:     container.Port,
		Username: limitedUsername,
		Password: limitedPassword,
		Database: &container.Database,
		IsHttps:  false,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err = mysqlModel.TestConnection(logger, nil)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "insufficient permissions")
		assert.Contains(t, err.Error(), "SELECT")
	}
}

func testTestConnectionSufficientPermissions(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MysqlVersion,
) {
	container := connectToMysqlEndpoint(t, endpoint, version)
	defer container.DB.Close()

	_, err := container.DB.Exec(`DROP TABLE IF EXISTS backup_test`)
	assert.NoError(t, err)

	_, err = container.DB.Exec(`CREATE TABLE backup_test (
				id INT AUTO_INCREMENT PRIMARY KEY,
				data VARCHAR(255) NOT NULL
			)`)
	assert.NoError(t, err)

	_, err = container.DB.Exec(`INSERT INTO backup_test (data) VALUES ('test1')`)
	assert.NoError(t, err)

	backupUsername := fmt.Sprintf("backup_%s", uuid.New().String()[:8])
	backupPassword := "backuppassword123"

	_, err = container.DB.Exec(fmt.Sprintf(
		"CREATE USER '%s'@'%%' IDENTIFIED BY '%s'",
		backupUsername,
		backupPassword,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		"GRANT SELECT, SHOW VIEW, LOCK TABLES, TRIGGER, EVENT ON `%s`.* TO '%s'@'%%'",
		container.Database,
		backupUsername,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec("FLUSH PRIVILEGES")
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(
			fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", backupUsername),
		)
	}()

	mysqlModel := &MysqlDatabase{
		Version:  version,
		Host:     container.Host,
		Port:     container.Port,
		Username: backupUsername,
		Password: backupPassword,
		Database: &container.Database,
		IsHttps:  false,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err = mysqlModel.TestConnection(logger, nil)
	assert.NoError(t, err)
}

func testShouldSuggestReadOnlyUserAdminUser(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MysqlVersion,
) {
	container := connectToMysqlEndpoint(t, endpoint, version)
	defer container.DB.Close()

	mysqlModel := createMysqlModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	shouldSuggestReadOnlyUser, privileges, err := mysqlModel.ShouldSuggestReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)
	assert.True(t, shouldSuggestReadOnlyUser, "Root user must be offered a read-only user")
	assert.NotEmpty(t, privileges, "Root user should have privileges")
}

func Test_ShouldSuggestReadOnlyUser_ReadOnlyUser_ReturnsFalse(t *testing.T) {
	container := connectToMysqlContainer(t, "mysql:8.0", tools.MysqlVersion80)
	defer container.DB.Close()

	_, err := container.DB.Exec(`DROP TABLE IF EXISTS readonly_check_test`)
	assert.NoError(t, err)

	_, err = container.DB.Exec(`CREATE TABLE readonly_check_test (
			id INT AUTO_INCREMENT PRIMARY KEY,
			data VARCHAR(255) NOT NULL
		)`)
	assert.NoError(t, err)

	_, err = container.DB.Exec(`INSERT INTO readonly_check_test (data) VALUES ('test1')`)
	assert.NoError(t, err)

	mysqlModel := createMysqlModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := mysqlModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)

	readOnlyModel := &MysqlDatabase{
		Version:  mysqlModel.Version,
		Host:     mysqlModel.Host,
		Port:     mysqlModel.Port,
		Username: username,
		Password: password,
		Database: mysqlModel.Database,
		IsHttps:  false,
	}

	shouldSuggestReadOnlyUser, privileges, err := readOnlyModel.ShouldSuggestReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)
	assert.False(t, shouldSuggestReadOnlyUser, "Read-only user must not be offered another one")
	assert.Empty(t, privileges, "Read-only user should have no write privileges")

	_, err = container.DB.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", username))
	assert.NoError(t, err)
}

func testCreateReadOnlyUserCanReadButNotWrite(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MysqlVersion,
) {
	container := connectToMysqlEndpoint(t, endpoint, version)
	defer container.DB.Close()

	_, err := container.DB.Exec(`DROP TABLE IF EXISTS readonly_test`)
	assert.NoError(t, err)
	_, err = container.DB.Exec(`DROP TABLE IF EXISTS hack_table`)
	assert.NoError(t, err)
	_, err = container.DB.Exec(`DROP TABLE IF EXISTS future_table`)
	assert.NoError(t, err)

	_, err = container.DB.Exec(`
				CREATE TABLE readonly_test (
					id INT AUTO_INCREMENT PRIMARY KEY,
					data VARCHAR(255) NOT NULL
				)
			`)
	assert.NoError(t, err)

	_, err = container.DB.Exec(
		`INSERT INTO readonly_test (data) VALUES ('test1'), ('test2')`,
	)
	assert.NoError(t, err)

	mysqlModel := createMysqlModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := mysqlModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, username)
	assert.NotEmpty(t, password)
	assert.True(t, strings.HasPrefix(username, "databasus-"))

	readOnlyModel := &MysqlDatabase{
		Version:  mysqlModel.Version,
		Host:     mysqlModel.Host,
		Port:     mysqlModel.Port,
		Username: username,
		Password: password,
		Database: mysqlModel.Database,
		IsHttps:  false,
	}

	shouldSuggestReadOnlyUser, privileges, err := readOnlyModel.ShouldSuggestReadOnlyUser(
		ctx,
		logger,
		nil,
	)
	assert.NoError(t, err)
	assert.False(t, shouldSuggestReadOnlyUser, "Created user must not be offered another one")
	assert.Empty(t, privileges, "Read-only user should have no write privileges")

	readOnlyDSN := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true",
		username,
		password,
		container.Host,
		container.Port,
		container.Database,
	)
	readOnlyConn, err := sqlx.Connect("mysql", readOnlyDSN)
	assert.NoError(t, err)
	defer readOnlyConn.Close()

	var count int
	err = readOnlyConn.Get(&count, "SELECT COUNT(*) FROM readonly_test")
	assert.NoError(t, err)
	assert.Equal(t, 2, count)

	_, err = readOnlyConn.Exec("INSERT INTO readonly_test (data) VALUES ('should-fail')")
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "denied")

	_, err = readOnlyConn.Exec("UPDATE readonly_test SET data = 'hacked' WHERE id = 1")
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "denied")

	_, err = readOnlyConn.Exec("DELETE FROM readonly_test WHERE id = 1")
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "denied")

	_, err = readOnlyConn.Exec("CREATE TABLE hack_table (id INT)")
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "denied")

	_, err = container.DB.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", username))
	assert.NoError(t, err)
}

func Test_ReadOnlyUser_FutureTables_NoSelectPermission(t *testing.T) {
	container := connectToMysqlContainer(t, "mysql:8.0", tools.MysqlVersion80)
	defer container.DB.Close()

	mysqlModel := createMysqlModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := mysqlModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)

	_, err = container.DB.Exec(`DROP TABLE IF EXISTS future_table`)
	assert.NoError(t, err)
	_, err = container.DB.Exec(`
		CREATE TABLE future_table (
			id INT AUTO_INCREMENT PRIMARY KEY,
			data VARCHAR(255) NOT NULL
		)
	`)
	assert.NoError(t, err)
	_, err = container.DB.Exec(`INSERT INTO future_table (data) VALUES ('future_data')`)
	assert.NoError(t, err)

	readOnlyDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		username, password, container.Host, container.Port, container.Database)
	readOnlyConn, err := sqlx.Connect("mysql", readOnlyDSN)
	assert.NoError(t, err)
	defer readOnlyConn.Close()

	var data string
	err = readOnlyConn.Get(&data, "SELECT data FROM future_table LIMIT 1")
	assert.NoError(t, err)
	assert.Equal(t, "future_data", data)

	_, err = container.DB.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", username))
	assert.NoError(t, err)
}

func Test_CreateReadOnlyUser_DatabaseNameWithDash_Success(t *testing.T) {
	container := connectToMysqlContainer(t, "mysql:8.0", tools.MysqlVersion80)
	defer container.DB.Close()

	dashDbName := "test-db-with-dash"

	_, err := container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dashDbName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE `%s`", dashDbName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dashDbName))
	}()

	dashDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		container.Username, container.Password, container.Host, container.Port, dashDbName)
	dashDB, err := sqlx.Connect("mysql", dashDSN)
	assert.NoError(t, err)
	defer dashDB.Close()

	_, err = dashDB.Exec(`
		CREATE TABLE dash_test (
			id INT AUTO_INCREMENT PRIMARY KEY,
			data VARCHAR(255) NOT NULL
		)
	`)
	assert.NoError(t, err)

	_, err = dashDB.Exec(`INSERT INTO dash_test (data) VALUES ('test1'), ('test2')`)
	assert.NoError(t, err)

	mysqlModel := &MysqlDatabase{
		Version:  tools.MysqlVersion80,
		Host:     container.Host,
		Port:     container.Port,
		Username: container.Username,
		Password: container.Password,
		Database: &dashDbName,
		IsHttps:  false,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := mysqlModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, username)
	assert.NotEmpty(t, password)
	assert.True(t, strings.HasPrefix(username, "databasus-"))

	readOnlyDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		username, password, container.Host, container.Port, dashDbName)
	readOnlyConn, err := sqlx.Connect("mysql", readOnlyDSN)
	assert.NoError(t, err)
	defer readOnlyConn.Close()

	var count int
	err = readOnlyConn.Get(&count, "SELECT COUNT(*) FROM dash_test")
	assert.NoError(t, err)
	assert.Equal(t, 2, count)

	_, err = readOnlyConn.Exec("INSERT INTO dash_test (data) VALUES ('should-fail')")
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "denied")

	_, err = dashDB.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", username))
	assert.NoError(t, err)
}

func Test_ReadOnlyUser_CannotDropOrAlterTables(t *testing.T) {
	container := connectToMysqlContainer(t, "mysql:8.0", tools.MysqlVersion80)
	defer container.DB.Close()

	_, err := container.DB.Exec(`DROP TABLE IF EXISTS drop_test`)
	assert.NoError(t, err)
	_, err = container.DB.Exec(`
		CREATE TABLE drop_test (
			id INT AUTO_INCREMENT PRIMARY KEY,
			data VARCHAR(255) NOT NULL
		)
	`)
	assert.NoError(t, err)
	_, err = container.DB.Exec(`INSERT INTO drop_test (data) VALUES ('test1')`)
	assert.NoError(t, err)

	mysqlModel := createMysqlModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := mysqlModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)

	readOnlyDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		username, password, container.Host, container.Port, container.Database)
	readOnlyConn, err := sqlx.Connect("mysql", readOnlyDSN)
	assert.NoError(t, err)
	defer readOnlyConn.Close()

	_, err = readOnlyConn.Exec("DROP TABLE drop_test")
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "denied")

	_, err = readOnlyConn.Exec("ALTER TABLE drop_test ADD COLUMN new_col VARCHAR(100)")
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "denied")

	_, err = readOnlyConn.Exec("TRUNCATE TABLE drop_test")
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "denied")

	_, err = container.DB.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", username))
	assert.NoError(t, err)
}

func testTestConnectionDatabaseSpecificPrivilegesWithGlobalProcess(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MysqlVersion,
) {
	container := connectToMysqlEndpoint(t, endpoint, version)
	defer container.DB.Close()

	_, err := container.DB.Exec(`DROP TABLE IF EXISTS privilege_test`)
	assert.NoError(t, err)

	_, err = container.DB.Exec(`CREATE TABLE privilege_test (
				id INT AUTO_INCREMENT PRIMARY KEY,
				data VARCHAR(255) NOT NULL
			)`)
	assert.NoError(t, err)

	_, err = container.DB.Exec(`INSERT INTO privilege_test (data) VALUES ('test1')`)
	assert.NoError(t, err)

	specificUsername := fmt.Sprintf("specific_%s", uuid.New().String()[:8])
	specificPassword := "specificpass123"

	_, err = container.DB.Exec(fmt.Sprintf(
		"CREATE USER '%s'@'%%' IDENTIFIED BY '%s'",
		specificUsername,
		specificPassword,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		"GRANT SELECT, SHOW VIEW ON %s.* TO '%s'@'%%'",
		container.Database,
		specificUsername,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		"GRANT PROCESS ON *.* TO '%s'@'%%'",
		specificUsername,
	))
	assert.NoError(t, err)

	_, err = container.DB.Exec("FLUSH PRIVILEGES")
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(
			fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", specificUsername),
		)
	}()

	mysqlModel := &MysqlDatabase{
		Version:  version,
		Host:     container.Host,
		Port:     container.Port,
		Username: specificUsername,
		Password: specificPassword,
		Database: &container.Database,
		IsHttps:  false,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err = mysqlModel.TestConnection(logger, nil)
	assert.NoError(t, err)
}

func Test_TestConnection_DatabaseWithUnderscores_Success(t *testing.T) {
	container := connectToMysqlContainer(t, "mysql:8.0", tools.MysqlVersion80)
	defer container.DB.Close()

	underscoreDbName := "test_db_name"

	_, err := container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", underscoreDbName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE `%s`", underscoreDbName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", underscoreDbName))
	}()

	underscoreDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		container.Username, container.Password, container.Host, container.Port, underscoreDbName)
	underscoreDB, err := sqlx.Connect("mysql", underscoreDSN)
	assert.NoError(t, err)
	defer underscoreDB.Close()

	_, err = underscoreDB.Exec(`
		CREATE TABLE underscore_test (
			id INT AUTO_INCREMENT PRIMARY KEY,
			data VARCHAR(255) NOT NULL
		)
	`)
	assert.NoError(t, err)

	_, err = underscoreDB.Exec(`INSERT INTO underscore_test (data) VALUES ('test1')`)
	assert.NoError(t, err)

	underscoreUsername := fmt.Sprintf("under_%s", uuid.New().String()[:8])
	underscorePassword := "underscorepass123"

	_, err = underscoreDB.Exec(fmt.Sprintf(
		"CREATE USER '%s'@'%%' IDENTIFIED BY '%s'",
		underscoreUsername,
		underscorePassword,
	))
	assert.NoError(t, err)

	_, err = underscoreDB.Exec(fmt.Sprintf(
		"GRANT SELECT, SHOW VIEW ON `%s`.* TO '%s'@'%%'",
		underscoreDbName,
		underscoreUsername,
	))
	assert.NoError(t, err)

	_, err = underscoreDB.Exec("FLUSH PRIVILEGES")
	assert.NoError(t, err)

	defer func() {
		_, _ = underscoreDB.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", underscoreUsername))
	}()

	mysqlModel := &MysqlDatabase{
		Version:  tools.MysqlVersion80,
		Host:     container.Host,
		Port:     container.Port,
		Username: underscoreUsername,
		Password: underscorePassword,
		Database: &underscoreDbName,
		IsHttps:  false,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err = mysqlModel.TestConnection(logger, nil)
	assert.NoError(t, err)
}

func testTestConnectionDatabaseWithUnderscoresAndAllPrivileges(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MysqlVersion,
) {
	container := connectToMysqlEndpoint(t, endpoint, version)
	defer container.DB.Close()

	underscoreDbName := "test_all_db"

	_, err := container.DB.Exec(
		fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", underscoreDbName),
	)
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE `%s`", underscoreDbName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(
			fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", underscoreDbName),
		)
	}()

	underscoreDSN := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true",
		container.Username,
		container.Password,
		container.Host,
		container.Port,
		underscoreDbName,
	)
	underscoreDB, err := sqlx.Connect("mysql", underscoreDSN)
	assert.NoError(t, err)
	defer underscoreDB.Close()

	_, err = underscoreDB.Exec(`
				CREATE TABLE all_priv_test (
					id INT AUTO_INCREMENT PRIMARY KEY,
					data VARCHAR(255) NOT NULL
				)
			`)
	assert.NoError(t, err)

	_, err = underscoreDB.Exec(`INSERT INTO all_priv_test (data) VALUES ('test1')`)
	assert.NoError(t, err)

	allPrivUsername := fmt.Sprintf("allpriv_%s", uuid.New().String()[:8])
	allPrivPassword := "allprivpass123"

	_, err = underscoreDB.Exec(fmt.Sprintf(
		"CREATE USER '%s'@'%%' IDENTIFIED BY '%s'",
		allPrivUsername,
		allPrivPassword,
	))
	assert.NoError(t, err)

	_, err = underscoreDB.Exec(fmt.Sprintf(
		"GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%'",
		underscoreDbName,
		allPrivUsername,
	))
	assert.NoError(t, err)

	_, err = underscoreDB.Exec("FLUSH PRIVILEGES")
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(
			fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", allPrivUsername),
		)
	}()

	mysqlModel := &MysqlDatabase{
		Version:  version,
		Host:     container.Host,
		Port:     container.Port,
		Username: allPrivUsername,
		Password: allPrivPassword,
		Database: &underscoreDbName,
		IsHttps:  false,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err = mysqlModel.TestConnection(logger, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, mysqlModel.Privileges)
	assert.Contains(t, mysqlModel.Privileges, "SELECT")
	assert.Contains(t, mysqlModel.Privileges, "SHOW VIEW")
}

type MysqlContainer struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Version  tools.MysqlVersion
	DB       *sqlx.DB
}

func testGetRawDbSizeMbReturnsPositiveSize(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MysqlVersion,
) {
	container := connectToMysqlEndpoint(t, endpoint, version)
	defer container.DB.Close()

	tableName := fmt.Sprintf("size_test_%s", uuid.New().String()[:8])
	_, err := container.DB.Exec(fmt.Sprintf(
		`CREATE TABLE %s (id INT AUTO_INCREMENT PRIMARY KEY, payload TEXT NOT NULL)`,
		tableName,
	))
	assert.NoError(t, err)
	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	}()

	for i := 0; i < 1000; i++ {
		_, err = container.DB.Exec(
			fmt.Sprintf("INSERT INTO %s (payload) VALUES (?)", tableName),
			strings.Repeat("x", 1024),
		)
		assert.NoError(t, err)
	}

	_, err = container.DB.Exec("ANALYZE TABLE " + tableName)
	assert.NoError(t, err)

	mysqlModel := createMysqlModel(container)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	sizeMB, err := mysqlModel.GetRawDbSizeMb(t.Context(), logger, nil)
	assert.NoError(t, err)
	assert.Greater(t, sizeMB, 0.0, "raw db size should be > 0 after inserting data")
}

func testGetRawDbSizeMbWhenMysqlStatisticsCacheIsWarmReturnsCurrentSize(
	t *testing.T,
	endpoint containers.Endpoint,
) {
	container := connectToMysqlEndpoint(t, endpoint, tools.MysqlVersion80)
	t.Cleanup(func() {
		assert.NoError(t, container.DB.Close())
	})

	tableName := fmt.Sprintf("cached_size_test_%s", uuid.New().String()[:8])
	_, err := container.DB.Exec(fmt.Sprintf(
		"CREATE TABLE `%s` (id INT AUTO_INCREMENT PRIMARY KEY, payload VARBINARY(8000) NOT NULL)",
		tableName,
	))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := container.DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName))
		assert.NoError(t, cleanupErr)
	})

	_, err = container.DB.Exec("SET GLOBAL information_schema_stats_expiry = 86400")
	require.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf(
		"INSERT INTO `%s` (payload) VALUES (REPEAT('x', 8000))",
		tableName,
	))
	require.NoError(t, err)

	_, err = container.DB.Exec("ANALYZE TABLE `" + tableName + "`")
	require.NoError(t, err)

	mysqlModel := createMysqlModel(container)
	mysqlModel.Username = containers.MysqlUsername
	mysqlModel.Password = containers.MysqlPassword
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	initialSizeMB, err := mysqlModel.GetRawDbSizeMb(t.Context(), logger, nil)
	require.NoError(t, err)

	for range 13 {
		_, err = container.DB.Exec(fmt.Sprintf(
			"INSERT INTO `%[1]s` (payload) SELECT payload FROM `%[1]s`",
			tableName,
		))
		require.NoError(t, err)
	}

	freshStatisticsConnection, err := container.DB.Connx(t.Context())
	require.NoError(t, err)
	defer freshStatisticsConnection.Close()

	_, err = freshStatisticsConnection.ExecContext(
		t.Context(),
		"SET SESSION information_schema_stats_expiry = 0",
	)
	require.NoError(t, err)

	var currentSizeMB float64
	require.Eventually(t, func() bool {
		err = freshStatisticsConnection.QueryRowxContext(t.Context(), `
			SELECT COALESCE(SUM(data_length + index_length), 0) / (1024 * 1024)
			FROM information_schema.tables
			WHERE table_schema = ?
		`, container.Database).Scan(&currentSizeMB)

		return err == nil && currentSizeMB > initialSizeMB+16
	}, 20*time.Second, 100*time.Millisecond)
	require.NoError(t, err)

	reportedSizeMB, err := mysqlModel.GetRawDbSizeMb(t.Context(), logger, nil)
	require.NoError(t, err)

	assert.InDelta(t, currentSizeMB, reportedSizeMB, 1.0)
}

func Test_HideSensitiveData_WhenCalled_ClearsPasswordAndPreservesOtherFields(t *testing.T) {
	databaseName := "appdb"
	mysqlModel := &MysqlDatabase{
		Version:       tools.MysqlVersion80,
		Host:          "db.example.com",
		Port:          3306,
		Username:      "appuser",
		Password:      "supersecret",
		Database:      &databaseName,
		IsHttps:       true,
		ExcludeTables: []string{"audit_logs"},
		Privileges:    "SELECT, INSERT",
	}

	mysqlModel.HideSensitiveData()

	assert.Empty(t, mysqlModel.Password)
	assert.Equal(t, "db.example.com", mysqlModel.Host)
	assert.Equal(t, 3306, mysqlModel.Port)
	assert.Equal(t, "appuser", mysqlModel.Username)
	assert.Equal(t, &databaseName, mysqlModel.Database)
	assert.True(t, mysqlModel.IsHttps)
	assert.Equal(t, []string{"audit_logs"}, mysqlModel.ExcludeTables)
	assert.Equal(t, "SELECT, INSERT", mysqlModel.Privileges)
}

func Test_HideSensitiveData_WhenReceiverIsNil_DoesNotPanic(t *testing.T) {
	var mysqlModel *MysqlDatabase

	assert.NotPanics(t, func() {
		mysqlModel.HideSensitiveData()
	})
}

func Test_MapMysqlVersion_MapsMajorsToBundledClients(t *testing.T) {
	cases := []struct {
		major                 string
		minor                 string
		expectedClientVersion tools.MysqlVersion
	}{
		{"5", "7", tools.MysqlVersion57},
		{"8", "0", tools.MysqlVersion80},
		{"8", "3", tools.MysqlVersion80},
		{"8", "4", tools.MysqlVersion84},
		{"9", "7", tools.MysqlVersion9},
		{"26", "7", tools.MysqlVersion9},
		{"27", "1", tools.MysqlVersion9},
	}

	for _, c := range cases {
		clientVersion, err := mapMysqlVersion(c.major, c.minor)

		require.NoError(t, err, "%s.%s", c.major, c.minor)
		assert.Equal(t, c.expectedClientVersion, clientVersion, "%s.%s", c.major, c.minor)
	}
}

func Test_MapMysqlVersion_RejectsUnsupportedMajors(t *testing.T) {
	cases := []struct {
		major string
		minor string
	}{
		{"4", "1"},
		{"6", "0"},
		{"7", "0"},
		{"10", "11"},
		{"11", "4"},
		{"25", "0"},
	}

	for _, c := range cases {
		_, err := mapMysqlVersion(c.major, c.minor)

		assert.ErrorContains(t, err, "unsupported MySQL major version", "%s.%s", c.major, c.minor)
	}
}

func connectToMysqlContainer(
	t *testing.T,
	image string,
	version tools.MysqlVersion,
) *MysqlContainer {
	endpoint := containers.StartMysql(t, image)

	return connectToMysqlEndpoint(t, endpoint, version)
}

func connectToMysqlEndpoint(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MysqlVersion,
) *MysqlContainer {
	dbName := containers.MysqlDatabase
	username := "root"
	password := containers.MysqlRootPassword

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		username, password, endpoint.Host, endpoint.Port, dbName)

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to MySQL %s: %v", version, err)
	}

	return &MysqlContainer{
		Host:     endpoint.Host,
		Port:     endpoint.Port,
		Username: username,
		Password: password,
		Database: dbName,
		Version:  version,
		DB:       db,
	}
}

func createMysqlModel(container *MysqlContainer) *MysqlDatabase {
	return &MysqlDatabase{
		Version:  container.Version,
		Host:     container.Host,
		Port:     container.Port,
		Username: container.Username,
		Password: container.Password,
		Database: &container.Database,
		IsHttps:  false,
	}
}
