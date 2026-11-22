package neon

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testreport"
)

const defaultPostgresqlPort = 5432

var errConnectionURLIncomplete = errors.New("the connection URL needs a host, a user, a password and a database")

// managedServer is a PostgreSQL server reached through a connection URL. The URL is a secret: it
// is never printed, and every error the server reports goes through the redactor first.
type managedServer struct {
	connectionURL       string
	host                string
	port                int
	username            string
	password            string
	maintenanceDatabase string
	sslMode             apiclient.PostgresqlSslMode
	redactor            secretRedactor
}

func parseManagedServer(connectionURL string) (managedServer, error) {
	redactor := newSecretRedactor(connectionURL)

	parsedURL, err := url.Parse(connectionURL)
	if err != nil {
		return managedServer{}, errors.New(redactor.redact(err.Error()))
	}

	if parsedURL.Scheme != "postgres" && parsedURL.Scheme != "postgresql" {
		return managedServer{}, fmt.Errorf("the connection URL must start with postgresql://, not %q", parsedURL.Scheme)
	}

	port := defaultPostgresqlPort
	if parsedURL.Port() != "" {
		port, err = strconv.Atoi(parsedURL.Port())
		if err != nil {
			return managedServer{}, errors.New(redactor.redact(err.Error()))
		}
	}

	password, _ := parsedURL.User.Password()
	server := managedServer{
		connectionURL:       connectionURL,
		host:                parsedURL.Hostname(),
		port:                port,
		username:            parsedURL.User.Username(),
		password:            password,
		maintenanceDatabase: strings.TrimPrefix(parsedURL.Path, "/"),
		sslMode:             toPostgresqlSslMode(parsedURL.Query().Get("sslmode")),
		redactor:            redactor,
	}

	if server.host == "" || server.username == "" || server.password == "" || server.maintenanceDatabase == "" {
		return managedServer{}, errConnectionURLIncomplete
	}

	return server, nil
}

func toPostgresqlSslMode(sslMode string) apiclient.PostgresqlSslMode {
	switch sslMode {
	case "disable", "allow":
		return apiclient.PostgresqlSslModeDisable
	case "verify-ca":
		return apiclient.PostgresqlSslModeVerifyCa
	case "verify-full":
		return apiclient.PostgresqlSslModeVerifyFull
	default:
		return apiclient.PostgresqlSslModeRequire
	}
}

// describe names the server for a reader of a failure: the host, the user and the database name
// tell which project failed, and none of them is a secret.
func (s managedServer) describe(databaseName string) string {
	return fmt.Sprintf("database %s as %s on %s:%d", databaseName, s.username, s.host, s.port)
}

func (s managedServer) connect(ctx context.Context, databaseName string) (*pgx.Conn, error) {
	config, err := pgx.ParseConfig(s.connectionURL)
	if err != nil {
		return nil, errors.New(s.redactor.redact(err.Error()))
	}

	config.Database = databaseName

	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %s", s.describe(databaseName), s.redactor.redact(err.Error()))
	}

	return conn, nil
}

// open connects to the database and closes the connection when the test ends.
func (s managedServer) open(reporter testreport.Reporter, databaseName string) *pgx.Conn {
	reporter.Helper()

	conn, err := s.connect(reporter.Context(), databaseName)
	if err != nil {
		reporter.Fatalf("%s", err)
	}

	reporter.Cleanup(func() { _ = conn.Close(context.WithoutCancel(reporter.Context())) })

	return conn
}

func (s managedServer) execute(reporter testreport.Reporter, conn *pgx.Conn, statements string) {
	reporter.Helper()

	if _, err := conn.Exec(reporter.Context(), statements); err != nil {
		reporter.Fatalf("run statements on %s: %s",
			s.describe(conn.Config().Database), s.redactor.redact(err.Error()))
	}
}

func (s managedServer) newLogicalSettings(databaseName string) apiclient.PostgresqlLogicalSettings {
	password := s.password

	return apiclient.PostgresqlLogicalSettings{
		Type:               apiclient.PostgresqlLogicalSettingsTypePOSTGRESLOGICAL,
		Host:               s.host,
		Port:               s.port,
		Username:           s.username,
		Password:           &password,
		Database:           &databaseName,
		SslMode:            s.sslMode,
		SslClientCert:      nil,
		SslClientKey:       nil,
		SslRootCert:        nil,
		SshTunnel:          enginetest.NewDisabledSSHTunnelSettings(),
		IncludeSchemas:     []string{},
		ExcludeTables:      []string{},
		ExcludeTableData:   []string{},
		IsSkipUserMappings: false,
		CpuCount:           1,
		Version:            nil,
	}
}

func (s managedServer) newRestoreTarget(databaseName string) apiclient.PostgresqlRestoreTarget {
	password := s.password

	return apiclient.PostgresqlRestoreTarget{
		Type:                apiclient.PostgresqlRestoreTargetTypePOSTGRESLOGICAL,
		Host:                s.host,
		Port:                s.port,
		Username:            s.username,
		Password:            &password,
		Database:            databaseName,
		SslMode:             s.sslMode,
		SslClientCert:       nil,
		SslClientKey:        nil,
		SslRootCert:         nil,
		SshTunnel:           enginetest.NewDisabledSSHTunnelSettings(),
		CpuCount:            1,
		IsExcludeExtensions: false,
		IsRestoreOwnership:  false,
		IsRestorePrivileges: false,
	}
}
