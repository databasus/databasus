package restores_core

import (
	mariadb_engine "databasus-backend/internal/engines/mariadb"
	mongodb_engine "databasus-backend/internal/engines/mongodb"
	mysql_engine "databasus-backend/internal/engines/mysql"
	pglogical "databasus-backend/internal/engines/postgresql/logical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/sshtunnel"
)

// The target types carry the JSON field names the restore form has always sent. The secrets arrive
// in clear text and are never stored.

type TargetSshTunnel struct {
	IsEnabled            bool               `json:"isEnabled"`
	Host                 string             `json:"host"`
	Port                 int                `json:"port"`
	Username             string             `json:"username"`
	AuthType             sshtunnel.AuthType `json:"authType"`
	Password             string             `json:"password"`
	PrivateKey           string             `json:"privateKey"`
	PrivateKeyPassphrase string             `json:"privateKeyPassphrase"`
}

func (t TargetSshTunnel) getSettings() *sshtunnel.Settings {
	if !t.IsEnabled {
		return nil
	}

	return &sshtunnel.Settings{
		Host:                 t.Host,
		Port:                 t.Port,
		Username:             t.Username,
		AuthType:             t.AuthType,
		Password:             t.Password,
		PrivateKey:           t.PrivateKey,
		PrivateKeyPassphrase: t.PrivateKeyPassphrase,
	}
}

type PostgresqlTargetDatabase struct {
	Version  pgshared.Version `json:"version"`
	Host     string           `json:"host"`
	Port     int              `json:"port"`
	Username string           `json:"username"`
	Password string           `json:"password"`
	Database *string          `json:"database"`

	SslMode       pgshared.SslMode `json:"sslMode"`
	SslClientCert string           `json:"sslClientCert"`
	SslClientKey  string           `json:"sslClientKey"`
	SslRootCert   string           `json:"sslRootCert"`

	SshTunnel TargetSshTunnel `json:"sshTunnel"`

	CpuCount            int  `json:"cpuCount"`
	IsExcludeExtensions bool `json:"isExcludeExtensions"`
	IsRestoreOwnership  bool `json:"isRestoreOwnership"`
	IsRestorePrivileges bool `json:"isRestorePrivileges"`

	// HostAddr is the local end of the SSH tunnel the restore connects through; TLS and .pgpass
	// still match Host.
	HostAddr string `json:"-"`
}

func (d PostgresqlTargetDatabase) GetEngineSettings() pglogical.Settings {
	databaseName := ""
	if d.Database != nil {
		databaseName = *d.Database
	}

	return pglogical.Settings{
		Host:     d.Host,
		HostAddr: d.HostAddr,
		Port:     d.Port,
		Username: d.Username,
		Password: d.Password,
		Database: databaseName,
		TLS: pgshared.TLS{
			Mode:       d.SslMode,
			ClientCert: d.SslClientCert,
			ClientKey:  d.SslClientKey,
			RootCert:   d.SslRootCert,
		},
		SSHTunnel: d.SshTunnel.getSettings(),
	}
}

func (d PostgresqlTargetDatabase) withConnection(settings pglogical.Settings) *PostgresqlTargetDatabase {
	d.HostAddr = settings.HostAddr
	d.Port = settings.Port
	d.SshTunnel = TargetSshTunnel{}

	return &d
}

type MysqlTargetDatabase struct {
	Version  mysql_engine.Version `json:"version"`
	Host     string               `json:"host"`
	Port     int                  `json:"port"`
	Username string               `json:"username"`
	Password string               `json:"password"`
	Database *string              `json:"database"`
	IsHttps  bool                 `json:"isHttps"`

	SshTunnel TargetSshTunnel `json:"sshTunnel"`
}

func (d MysqlTargetDatabase) GetEngineSettings() mysql_engine.Settings {
	return mysql_engine.Settings{
		Host:      d.Host,
		Port:      d.Port,
		Username:  d.Username,
		Password:  d.Password,
		Database:  getDatabaseName(d.Database),
		IsHTTPS:   d.IsHttps,
		SSHTunnel: d.SshTunnel.getSettings(),
	}
}

func (d MysqlTargetDatabase) withConnection(settings mysql_engine.Settings) *MysqlTargetDatabase {
	d.Host = settings.Host
	d.Port = settings.Port
	d.SshTunnel = TargetSshTunnel{}

	return &d
}

type MariadbTargetDatabase struct {
	Version             mariadb_engine.Version `json:"version"`
	Host                string                 `json:"host"`
	Port                int                    `json:"port"`
	Username            string                 `json:"username"`
	Password            string                 `json:"password"`
	Database            *string                `json:"database"`
	IsHttps             bool                   `json:"isHttps"`
	IsSkipGaleraDisable bool                   `json:"isSkipGaleraDisable"`

	SshTunnel TargetSshTunnel `json:"sshTunnel"`
}

func (d MariadbTargetDatabase) GetEngineSettings() mariadb_engine.Settings {
	return mariadb_engine.Settings{
		Host:      d.Host,
		Port:      d.Port,
		Username:  d.Username,
		Password:  d.Password,
		Database:  getDatabaseName(d.Database),
		IsHTTPS:   d.IsHttps,
		SSHTunnel: d.SshTunnel.getSettings(),
	}
}

func (d MariadbTargetDatabase) withConnection(settings mariadb_engine.Settings) *MariadbTargetDatabase {
	d.Host = settings.Host
	d.Port = settings.Port
	d.SshTunnel = TargetSshTunnel{}

	return &d
}

type MongodbTargetDatabase struct {
	Version            mongodb_engine.Version `json:"version"`
	Host               string                 `json:"host"`
	Port               *int                   `json:"port"`
	Username           string                 `json:"username"`
	Password           string                 `json:"password"`
	Database           string                 `json:"database"`
	AuthDatabase       string                 `json:"authDatabase"`
	IsHttps            bool                   `json:"isHttps"`
	IsSrv              bool                   `json:"isSrv"`
	IsDirectConnection bool                   `json:"isDirectConnection"`
	CpuCount           int                    `json:"cpuCount"`

	SshTunnel TargetSshTunnel `json:"sshTunnel"`
}

func (d MongodbTargetDatabase) GetEngineSettings() mongodb_engine.Settings {
	port := 0
	if d.Port != nil {
		port = *d.Port
	}

	return mongodb_engine.Settings{
		Host:               d.Host,
		Port:               port,
		Username:           d.Username,
		Password:           d.Password,
		Database:           d.Database,
		AuthDatabase:       d.AuthDatabase,
		IsTLS:              d.IsHttps,
		IsSrv:              d.IsSrv,
		IsDirectConnection: d.IsDirectConnection,
		SSHTunnel:          d.SshTunnel.getSettings(),
	}
}

func (d MongodbTargetDatabase) withConnection(settings mongodb_engine.Settings) *MongodbTargetDatabase {
	d.Host = settings.Host
	d.IsDirectConnection = settings.IsDirectConnection
	d.SshTunnel = TargetSshTunnel{}

	if settings.Port != 0 {
		port := settings.Port
		d.Port = &port
	}

	return &d
}

func getDatabaseName(database *string) string {
	if database == nil {
		return ""
	}

	return *database
}
