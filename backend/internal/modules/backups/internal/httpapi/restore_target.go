package httpapi

import (
	mariadb_engine "databasus-backend/internal/engines/mariadb"
	mongodb_engine "databasus-backend/internal/engines/mongodb"
	mysql_engine "databasus-backend/internal/engines/mysql"
	pglogical "databasus-backend/internal/engines/postgresql/logical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/sshtunnel"
	"databasus-backend/internal/modules/backups/internal/pipeline"
)

func toRestoreTarget(body RestoreTarget) (pipeline.RestoreTarget, error) {
	targetBody, err := body.ValueByDiscriminator()
	if err != nil {
		return nil, errUnknownRestoreTarget
	}

	switch typedTarget := targetBody.(type) {
	case PostgresqlRestoreTarget:
		return pipeline.PostgresqlRestoreTarget{
			Settings: pglogical.Settings{
				Host:     typedTarget.Host,
				HostAddr: "",
				Port:     typedTarget.Port,
				Username: typedTarget.Username,
				Password: getValueOrZero(typedTarget.Password),
				Database: typedTarget.Database,
				TLS: pgshared.TLS{
					Mode:       pgshared.SslMode(typedTarget.SslMode),
					ClientCert: getValueOrZero(typedTarget.SslClientCert),
					ClientKey:  getValueOrZero(typedTarget.SslClientKey),
					RootCert:   getValueOrZero(typedTarget.SslRootCert),
				},
				SSHTunnel: toSSHTunnelSettings(typedTarget.SshTunnel),
			},
			CPUCount:            typedTarget.CpuCount,
			IsExcludeExtensions: typedTarget.IsExcludeExtensions,
			IsRestoreOwnership:  typedTarget.IsRestoreOwnership,
			IsRestorePrivileges: typedTarget.IsRestorePrivileges,
		}, nil
	case MysqlRestoreTarget:
		return pipeline.MysqlRestoreTarget{
			Settings: mysql_engine.Settings{
				Host:      typedTarget.Host,
				Port:      typedTarget.Port,
				Username:  typedTarget.Username,
				Password:  getValueOrZero(typedTarget.Password),
				Database:  typedTarget.Database,
				IsHTTPS:   typedTarget.IsHttps,
				SSHTunnel: toSSHTunnelSettings(typedTarget.SshTunnel),
			},
		}, nil
	case MariadbRestoreTarget:
		return pipeline.MariadbRestoreTarget{
			Settings: mariadb_engine.Settings{
				Host:      typedTarget.Host,
				Port:      typedTarget.Port,
				Username:  typedTarget.Username,
				Password:  getValueOrZero(typedTarget.Password),
				Database:  typedTarget.Database,
				IsHTTPS:   typedTarget.IsHttps,
				SSHTunnel: toSSHTunnelSettings(typedTarget.SshTunnel),
			},
			IsSkipGaleraDisable: typedTarget.IsSkipGaleraDisable,
		}, nil
	case MongodbRestoreTarget:
		return pipeline.MongodbRestoreTarget{
			Settings: mongodb_engine.Settings{
				Host:               typedTarget.Host,
				Port:               getValueOrZero(typedTarget.Port),
				Username:           typedTarget.Username,
				Password:           getValueOrZero(typedTarget.Password),
				Database:           typedTarget.Database,
				AuthDatabase:       typedTarget.AuthDatabase,
				IsTLS:              typedTarget.IsHttps,
				IsSrv:              typedTarget.IsSrv,
				IsDirectConnection: typedTarget.IsDirectConnection,
				SSHTunnel:          toSSHTunnelSettings(typedTarget.SshTunnel),
			},
			CPUCount: typedTarget.CpuCount,
		}, nil
	default:
		return nil, errUnknownRestoreTarget
	}
}

func toSSHTunnelSettings(body SshTunnelSettings) *sshtunnel.Settings {
	if !body.IsEnabled {
		return nil
	}

	return &sshtunnel.Settings{
		Host:                 getValueOrZero(body.Host),
		Port:                 getValueOrZero(body.Port),
		Username:             getValueOrZero(body.Username),
		AuthType:             sshtunnel.AuthType(getValueOrZero(body.AuthType)),
		Password:             getValueOrZero(body.Password),
		PrivateKey:           getValueOrZero(body.PrivateKey),
		PrivateKeyPassphrase: getValueOrZero(body.PrivateKeyPassphrase),
	}
}

func getValueOrZero[T any](value *T) T {
	if value == nil {
		var zero T

		return zero
	}

	return *value
}
