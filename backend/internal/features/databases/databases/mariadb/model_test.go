package mariadb

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/tools"
)

type mariadbModelVersion struct {
	name                 string
	version              tools.MariadbVersion
	image                string
	runsRoutineGrantTest bool // only 11.4/11.8/12.0 expose SHOW CREATE ROUTINE
	runsRoleTests        bool // MariaDB grew roles in 10.0.5, so 5.5 has none
}

// 5.5 and 10.1 are here because they are the only servers the legacy client
// tier serves: without them nothing exercises that tier.
var mariadbModelVersions = []mariadbModelVersion{
	{"MariaDB 5.5", tools.MariadbVersion55, "mariadb:5.5", false, false},
	{"MariaDB 10.1", tools.MariadbVersion101, "mariadb:10.1", false, true},
	{"MariaDB 10.6", tools.MariadbVersion106, "mariadb:10.6", false, true},
	{"MariaDB 10.11", tools.MariadbVersion1011, "mariadb:10.11", false, true},
	{"MariaDB 11.4", tools.MariadbVersion114, "mariadb:11.4", true, true},
	{"MariaDB 11.8", tools.MariadbVersion118, "mariadb:11.8", true, true},
	{"MariaDB 12.0", tools.MariadbVersion120, "mariadb:12.0", true, true},
	{"MariaDB 13.0", tools.MariadbVersion130, "mariadb:13.0", true, true},
}

// Test_MariadbModel_AcrossSupportedVersions boots each MariaDB version once and runs every matrix
// model test against it as a subtest. Only one container is alive per package at a time. See docs/testing.md.
func Test_MariadbModel_AcrossSupportedVersions(t *testing.T) {
	for _, dbVersion := range mariadbModelVersions {
		t.Run(dbVersion.name, func(t *testing.T) {
			endpoint := containers.StartMariadb(t, dbVersion.image)

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

			if dbVersion.runsRoleTests {
				t.Run("Test_TestConnection_WhenBackupPrivilegesComeFromActiveRole_Success", func(t *testing.T) {
					testTestConnectionBackupPrivilegesFromActiveRole(t, endpoint, dbVersion.version)
				})

				t.Run("Test_TestConnection_WhenBackupPrivilegesComeFromNestedRole_Success", func(t *testing.T) {
					testTestConnectionBackupPrivilegesFromNestedRole(t, endpoint, dbVersion.version)
				})

				t.Run("Test_TestConnection_WhenRoleIsGrantedButNotActivated_ReturnsError", func(t *testing.T) {
					testTestConnectionRoleGrantedButNotActivated(t, endpoint, dbVersion.version)
				})
			}

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

			if dbVersion.runsRoutineGrantTest {
				t.Run("Test_ShouldSuggestReadOnlyUser_WithShowCreateRoutineGrant_ReturnsFalse", func(t *testing.T) {
					testShowCreateRoutineGrant(t, endpoint, dbVersion.version)
				})
			}
		})
	}
}

func testTestConnectionInsufficientPermissions(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MariadbVersion,
) {
	container := connectToMariadbEndpoint(t, endpoint, version)
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

	defer dropUserSafe(container.DB, limitedUsername)

	mariadbModel := &MariadbDatabase{
		Version:  version,
		Host:     container.Host,
		Port:     container.Port,
		Username: limitedUsername,
		Password: limitedPassword,
		Database: &container.Database,
		IsHttps:  false,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err = mariadbModel.TestConnection(logger, nil)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "insufficient permissions")
		assert.Contains(t, err.Error(), "SELECT")
	}
}

func testTestConnectionSufficientPermissions(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MariadbVersion,
) {
	container := connectToMariadbEndpoint(t, endpoint, version)
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

	defer dropUserSafe(container.DB, backupUsername)

	mariadbModel := &MariadbDatabase{
		Version:  version,
		Host:     container.Host,
		Port:     container.Port,
		Username: backupUsername,
		Password: backupPassword,
		Database: &container.Database,
		IsHttps:  false,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err = mariadbModel.TestConnection(logger, nil)
	assert.NoError(t, err)
}

func testShouldSuggestReadOnlyUserAdminUser(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MariadbVersion,
) {
	container := connectToMariadbEndpoint(t, endpoint, version)
	defer container.DB.Close()

	mariadbModel := createMariadbModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	shouldSuggestReadOnlyUser, privileges, err := mariadbModel.ShouldSuggestReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)
	assert.True(t, shouldSuggestReadOnlyUser, "Root user must be offered a read-only user")
	assert.NotEmpty(t, privileges, "Root user should have privileges")
}

func Test_ShouldSuggestReadOnlyUser_ReadOnlyUser_ReturnsFalse(t *testing.T) {
	container := connectToMariadbContainer(t, "mariadb:10.11", tools.MariadbVersion1011)
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

	mariadbModel := createMariadbModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := mariadbModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)

	readOnlyModel := &MariadbDatabase{
		Version:  mariadbModel.Version,
		Host:     mariadbModel.Host,
		Port:     mariadbModel.Port,
		Username: username,
		Password: password,
		Database: mariadbModel.Database,
		IsHttps:  false,
	}

	shouldSuggestReadOnlyUser, privileges, err := readOnlyModel.ShouldSuggestReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)
	assert.False(t, shouldSuggestReadOnlyUser, "Read-only user must not be offered another one")
	assert.Empty(t, privileges, "Read-only user should have no write privileges")

	dropUserSafe(container.DB, username)
}

func testCreateReadOnlyUserCanReadButNotWrite(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MariadbVersion,
) {
	container := connectToMariadbEndpoint(t, endpoint, version)
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

	mariadbModel := createMariadbModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := mariadbModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, username)
	assert.NotEmpty(t, password)
	assert.True(t, strings.HasPrefix(username, "pgs-"))

	if err != nil {
		return
	}

	readOnlyModel := &MariadbDatabase{
		Version:  mariadbModel.Version,
		Host:     mariadbModel.Host,
		Port:     mariadbModel.Port,
		Username: username,
		Password: password,
		Database: mariadbModel.Database,
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

	dropUserSafe(container.DB, username)
}

func Test_ReadOnlyUser_FutureTables_NoSelectPermission(t *testing.T) {
	container := connectToMariadbContainer(t, "mariadb:10.11", tools.MariadbVersion1011)
	defer container.DB.Close()

	mariadbModel := createMariadbModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := mariadbModel.CreateReadOnlyUser(ctx, logger, nil)
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

	dropUserSafe(container.DB, username)
}

func Test_CreateReadOnlyUser_DatabaseNameWithDash_Success(t *testing.T) {
	container := connectToMariadbContainer(t, "mariadb:10.11", tools.MariadbVersion1011)
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

	mariadbModel := &MariadbDatabase{
		Version:  tools.MariadbVersion1011,
		Host:     container.Host,
		Port:     container.Port,
		Username: container.Username,
		Password: container.Password,
		Database: &dashDbName,
		IsHttps:  false,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := mariadbModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, username)
	assert.NotEmpty(t, password)
	assert.True(t, strings.HasPrefix(username, "pgs-"))

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

	dropUserSafe(dashDB, username)
}

func Test_ReadOnlyUser_CannotDropOrAlterTables(t *testing.T) {
	container := connectToMariadbContainer(t, "mariadb:10.11", tools.MariadbVersion1011)
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

	mariadbModel := createMariadbModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	username, password, err := mariadbModel.CreateReadOnlyUser(ctx, logger, nil)
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

	dropUserSafe(container.DB, username)
}

func testTestConnectionDatabaseSpecificPrivilegesWithGlobalProcess(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MariadbVersion,
) {
	container := connectToMariadbEndpoint(t, endpoint, version)
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

	specificUsername := fmt.Sprintf("spec_%s", uuid.New().String()[:8])
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

	defer dropUserSafe(container.DB, specificUsername)

	mariadbModel := &MariadbDatabase{
		Version:  version,
		Host:     container.Host,
		Port:     container.Port,
		Username: specificUsername,
		Password: specificPassword,
		Database: &container.Database,
		IsHttps:  false,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err = mariadbModel.TestConnection(logger, nil)
	assert.NoError(t, err)
}

func Test_TestConnection_DatabaseWithUnderscores_Success(t *testing.T) {
	container := connectToMariadbContainer(t, "mariadb:10.11", tools.MariadbVersion1011)
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

	underscoreUsername := fmt.Sprintf("under%s", uuid.New().String()[:8])
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

	defer dropUserSafe(underscoreDB, underscoreUsername)

	mariadbModel := &MariadbDatabase{
		Version:  tools.MariadbVersion1011,
		Host:     container.Host,
		Port:     container.Port,
		Username: underscoreUsername,
		Password: underscorePassword,
		Database: &underscoreDbName,
		IsHttps:  false,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err = mariadbModel.TestConnection(logger, nil)
	assert.NoError(t, err)
}

func testTestConnectionDatabaseWithUnderscoresAndAllPrivileges(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MariadbVersion,
) {
	container := connectToMariadbEndpoint(t, endpoint, version)
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

	allPrivUsername := fmt.Sprintf("allpriv%s", uuid.New().String()[:8])
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

	defer dropUserSafe(underscoreDB, allPrivUsername)

	mariadbModel := &MariadbDatabase{
		Version:  version,
		Host:     container.Host,
		Port:     container.Port,
		Username: allPrivUsername,
		Password: allPrivPassword,
		Database: &underscoreDbName,
		IsHttps:  false,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err = mariadbModel.TestConnection(logger, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, mariadbModel.Privileges)
	assert.Contains(t, mariadbModel.Privileges, "SELECT")
	assert.Contains(t, mariadbModel.Privileges, "SHOW VIEW")
}

type MariadbContainer struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Version  tools.MariadbVersion
	DB       *sqlx.DB
}

func Test_GetRawDbSizeMb_Mariadb_ReturnsPositiveSize(t *testing.T) {
	container := connectToMariadbContainer(t, "mariadb:10.11", tools.MariadbVersion1011)
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

	mariadbModel := createMariadbModel(container)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	sizeMB, err := mariadbModel.GetRawDbSizeMb(t.Context(), logger, nil)
	assert.NoError(t, err)
	assert.Greater(t, sizeMB, 0.0, "raw db size should be > 0 after inserting data")
}

// Regression test for issue #568: a user with only SELECT, SHOW VIEW, and
// SHOW CREATE ROUTINE must be detected as read-only. Previously the
// substring-regex check falsely matched "CREATE" and "CREATE ROUTINE" inside
// "SHOW CREATE ROUTINE".
//
// SHOW CREATE ROUTINE was introduced as a distinct privilege in MariaDB
// 11.3; the matrix below only covers versions that recognize the GRANT
// statement.
func testShowCreateRoutineGrant(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MariadbVersion,
) {
	container := connectToMariadbEndpoint(t, endpoint, version)
	defer container.DB.Close()

	username := fmt.Sprintf("ro_%s", uuid.New().String()[:8])
	password := "ropass123"

	_, err := container.DB.Exec(fmt.Sprintf(
		"CREATE USER '%s'@'%%' IDENTIFIED BY '%s'", username, password))
	assert.NoError(t, err)
	defer dropUserSafe(container.DB, username)

	for _, stmt := range []string{
		"GRANT SELECT ON *.* TO '%s'@'%%'",
		"GRANT SHOW VIEW ON *.* TO '%s'@'%%'",
		"GRANT SHOW CREATE ROUTINE ON *.* TO '%s'@'%%'",
	} {
		_, err = container.DB.Exec(fmt.Sprintf(stmt, username))
		assert.NoError(t, err)
	}

	_, err = container.DB.Exec("FLUSH PRIVILEGES")
	assert.NoError(t, err)

	readOnlyModel := &MariadbDatabase{
		Version:  version,
		Host:     container.Host,
		Port:     container.Port,
		Username: username,
		Password: password,
		Database: &container.Database,
		IsHttps:  false,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	shouldSuggestReadOnlyUser, privileges, err := readOnlyModel.ShouldSuggestReadOnlyUser(
		t.Context(),
		logger,
		nil,
	)
	assert.NoError(t, err)
	assert.False(
		t,
		shouldSuggestReadOnlyUser,
		"SELECT + SHOW VIEW + SHOW CREATE ROUTINE must not be offered a read-only user, got privileges=%v",
		privileges,
	)
	assert.Empty(t, privileges)
}

func Test_HideSensitiveData_WhenCalled_ClearsPasswordAndPreservesOtherFields(t *testing.T) {
	databaseName := "appdb"
	mariadbModel := &MariadbDatabase{
		Version:         tools.MariadbVersion114,
		Host:            "db.example.com",
		Port:            3306,
		Username:        "appuser",
		Password:        "supersecret",
		Database:        &databaseName,
		IsHttps:         true,
		IsExcludeEvents: true,
		ExcludeTables:   []string{"audit_logs"},
		Privileges:      "SELECT, INSERT",
	}

	mariadbModel.HideSensitiveData()

	assert.Empty(t, mariadbModel.Password)
	assert.Equal(t, "db.example.com", mariadbModel.Host)
	assert.Equal(t, 3306, mariadbModel.Port)
	assert.Equal(t, "appuser", mariadbModel.Username)
	assert.Equal(t, &databaseName, mariadbModel.Database)
	assert.True(t, mariadbModel.IsHttps)
	assert.True(t, mariadbModel.IsExcludeEvents)
	assert.Equal(t, []string{"audit_logs"}, mariadbModel.ExcludeTables)
	assert.Equal(t, "SELECT, INSERT", mariadbModel.Privileges)
}

func Test_HideSensitiveData_WhenReceiverIsNil_DoesNotPanic(t *testing.T) {
	var mariadbModel *MariadbDatabase

	assert.NotPanics(t, func() {
		mariadbModel.HideSensitiveData()
	})
}

func connectToMariadbContainer(
	t *testing.T,
	image string,
	version tools.MariadbVersion,
) *MariadbContainer {
	endpoint := containers.StartMariadb(t, image)

	return connectToMariadbEndpoint(t, endpoint, version)
}

func connectToMariadbEndpoint(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MariadbVersion,
) *MariadbContainer {
	dbName := containers.MariadbDatabase
	username := "root"
	password := containers.MariadbRootPassword

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		username, password, endpoint.Host, endpoint.Port, dbName)

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to MariaDB %s: %v", version, err)
	}

	return &MariadbContainer{
		Host:     endpoint.Host,
		Port:     endpoint.Port,
		Username: username,
		Password: password,
		Database: dbName,
		Version:  version,
		DB:       db,
	}
}

func createMariadbModel(container *MariadbContainer) *MariadbDatabase {
	return &MariadbDatabase{
		Version:  container.Version,
		Host:     container.Host,
		Port:     container.Port,
		Username: container.Username,
		Password: container.Password,
		Database: &container.Database,
		IsHttps:  false,
	}
}

func dropUserSafe(db *sqlx.DB, username string) {
	_, _ = db.Exec(fmt.Sprintf("DROP USER '%s'@'%%'", username))
}
