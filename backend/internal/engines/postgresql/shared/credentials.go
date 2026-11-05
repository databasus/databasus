package shared

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type CredentialSpec struct {
	Host string
	// Named after the libpq parameter it maps to: when set, the connection is made to this address
	// while Host is what the certificate and .pgpass are matched against. That split is what lets an
	// SSH tunnel forward the traffic without breaking sslmode=verify-full.
	HostAddr string
	Port     int
	Username string
	TLS      TLS
}

// CredentialTempFiles are the .pgpass file and the certificates a client tool reads, in a directory
// only the backend user can read.
type CredentialTempFiles struct {
	Dir            string
	PgpassPath     string
	ClientCertPath string
	ClientKeyPath  string
	RootCertPath   string
}

func WriteCredentialFilesToTempDir(spec CredentialSpec, password string) (*CredentialTempFiles, error) {
	dir, err := os.MkdirTemp(os.TempDir(), "pgcreds_")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	if err := os.Chmod(dir, 0o700); err != nil {
		_ = os.RemoveAll(dir)

		return nil, fmt.Errorf("failed to set temporary directory permissions: %w", err)
	}

	files := &CredentialTempFiles{Dir: dir}

	if err := files.writePgpass(spec, password); err != nil {
		_ = os.RemoveAll(dir)

		return nil, err
	}

	if spec.TLS.getModeOrDefault() != SslModeDisable {
		if err := files.writeCertFiles(spec.TLS); err != nil {
			_ = os.RemoveAll(dir)

			return nil, err
		}
	}

	return files, nil
}

func (f *CredentialTempFiles) Remove() {
	if f == nil || f.Dir == "" {
		return
	}

	_ = os.RemoveAll(f.Dir)
}

// pgx has no hostaddr parameter, so through a tunnel the address goes in as the host and the real
// name is restored on the TLS config, which is what sslmode=verify-full matches against.
func BuildConnConfig(
	spec CredentialSpec,
	password, dbName string,
	files *CredentialTempFiles,
) (*pgx.ConnConfig, error) {
	connConfig, err := pgx.ParseConfig(buildConnString(spec, password, dbName, files))
	if err != nil {
		return nil, fmt.Errorf("failed to parse the PostgreSQL connection string: %w", err)
	}

	restoreServerNameForTunneledHost(&connConfig.Config, spec)

	return connConfig, nil
}

// BuildReplicationConnConfig builds the config of a replication connection. pgconn has no hostaddr
// either, and unlike the pgx path there is no ParseConfig step in the caller, so the ServerName
// restoration has to happen here or verify-full would match the tunnel's loopback.
func BuildReplicationConnConfig(
	spec CredentialSpec,
	password, dbName string,
	files *CredentialTempFiles,
) (*pgconn.Config, error) {
	connConfig, err := pgconn.ParseConfig(buildReplicationConnString(spec, password, dbName, files))
	if err != nil {
		return nil, fmt.Errorf("failed to parse the PostgreSQL replication connection string: %w", err)
	}

	restoreServerNameForTunneledHost(connConfig, spec)

	return connConfig, nil
}

// libpq resolves and connects to hostaddr while still matching the certificate and .pgpass against
// the host argument, so the CLI tools need no rewritten -h.
func GetPgHostAddrEnv(hostAddr string) []string {
	if hostAddr == "" {
		return nil
	}

	return []string{"PGHOSTADDR=" + hostAddr}
}

func BuildDatabaseNameConninfo(databaseName string) string {
	return "dbname=" + quoteConninfoValue(databaseName)
}

func buildConnString(
	spec CredentialSpec,
	password, dbName string,
	files *CredentialTempFiles,
) string {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		quoteConninfoValue(getConnectHost(spec)),
		spec.Port,
		quoteConninfoValue(spec.Username),
		quoteConninfoValue(password),
		quoteConninfoValue(dbName),
		quoteConninfoValue(string(spec.TLS.getModeOrDefault())),
	)

	connStr += " default_query_exec_mode=simple_protocol standard_conforming_strings=on client_encoding=UTF8"

	return appendSslFilePaths(connStr, files)
}

// replication=true is the mode pg_basebackup / pg_receivewal use, and the one pg_hba matches via
// "host replication" rules rather than ordinary "host all" rules. The pgx-only query-exec params are
// omitted so the string stays consumable by the low-level pgconn used for the connectivity probe.
func buildReplicationConnString(
	spec CredentialSpec,
	password, dbName string,
	files *CredentialTempFiles,
) string {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s replication=true",
		quoteConninfoValue(getConnectHost(spec)),
		spec.Port,
		quoteConninfoValue(spec.Username),
		quoteConninfoValue(password),
		quoteConninfoValue(dbName),
		quoteConninfoValue(string(spec.TLS.getModeOrDefault())),
	)

	return appendSslFilePaths(connStr, files)
}

// pgx builds one fallback config per sslmode it may retry with, so skipping those would silently
// verify against the tunnel address on the retry.
func restoreServerNameForTunneledHost(connConfig *pgconn.Config, spec CredentialSpec) {
	if spec.HostAddr == "" {
		return
	}

	if connConfig.TLSConfig != nil {
		connConfig.TLSConfig.ServerName = spec.Host
	}

	for _, fallback := range connConfig.Fallbacks {
		if fallback.TLSConfig != nil {
			fallback.TLSConfig.ServerName = spec.Host
		}
	}
}

func getConnectHost(spec CredentialSpec) string {
	if spec.HostAddr != "" {
		return spec.HostAddr
	}

	return spec.Host
}

func appendSslFilePaths(connStr string, files *CredentialTempFiles) string {
	if files == nil {
		return connStr
	}

	if files.ClientCertPath != "" {
		connStr += fmt.Sprintf(" sslcert=%s", quoteConninfoValue(files.ClientCertPath))
	}

	if files.ClientKeyPath != "" {
		connStr += fmt.Sprintf(" sslkey=%s", quoteConninfoValue(files.ClientKeyPath))
	}

	if files.RootCertPath != "" {
		connStr += fmt.Sprintf(" sslrootcert=%s", quoteConninfoValue(files.RootCertPath))
	}

	return connStr
}

func (f *CredentialTempFiles) writePgpass(spec CredentialSpec, password string) error {
	content := fmt.Sprintf("%s:%d:*:%s:%s",
		escapePgpassField(spec.Host),
		spec.Port,
		escapePgpassField(spec.Username),
		escapePgpassField(password),
	)

	path := filepath.Join(f.Dir, ".pgpass")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("failed to write .pgpass file: %w", err)
	}

	f.PgpassPath = path

	return nil
}

func (f *CredentialTempFiles) writeCertFiles(tls TLS) error {
	var err error

	if f.ClientCertPath, err = f.writeCert("client.crt", tls.ClientCert); err != nil {
		return fmt.Errorf("failed to write client certificate: %w", err)
	}

	if f.ClientKeyPath, err = f.writeCert("client.key", tls.ClientKey); err != nil {
		return fmt.Errorf("failed to write client key: %w", err)
	}

	if f.RootCertPath, err = f.writeCert("root.crt", tls.RootCert); err != nil {
		return fmt.Errorf("failed to write server CA certificate: %w", err)
	}

	return nil
}

func (f *CredentialTempFiles) writeCert(fileName, pem string) (string, error) {
	if pem == "" {
		return "", nil
	}

	path := filepath.Join(f.Dir, fileName)
	if err := os.WriteFile(path, []byte(pem), 0o600); err != nil {
		return "", err
	}

	return path, nil
}

// escapePgpassField escapes a value for the .pgpass format, which requires backslash as \\ and
// colon as \:. Newlines and carriage returns are stripped, because they would start a new entry.
func escapePgpassField(field string) string {
	field = strings.ReplaceAll(field, "\r", "")
	field = strings.ReplaceAll(field, "\n", "")
	field = strings.ReplaceAll(field, "\\", "\\\\")
	field = strings.ReplaceAll(field, ":", "\\:")

	return field
}

func quoteConninfoValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `\'`)

	return "'" + value + "'"
}
