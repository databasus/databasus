package httpapi

import "databasus-backend/internal/modules/databases/internal/providers"

func toSSHTunnelSettings(body SshTunnelSettings) providers.SSHTunnelSettings {
	return providers.SSHTunnelSettings{
		IsEnabled:            body.IsEnabled,
		Host:                 getValueOrZero(body.Host),
		Port:                 getValueOrZero(body.Port),
		Username:             getValueOrZero(body.Username),
		AuthType:             providers.SSHAuthType(getValueOrZero(body.AuthType)),
		Password:             getValueOrZero(body.Password),
		PrivateKey:           getValueOrZero(body.PrivateKey),
		PrivateKeyPassphrase: getValueOrZero(body.PrivateKeyPassphrase),
	}
}

func fromSSHTunnelSettings(settings providers.SSHTunnelSettings) SshTunnelSettings {
	return SshTunnelSettings{
		IsEnabled:            settings.IsEnabled,
		Host:                 toOptionalValue(settings.Host),
		Port:                 toOptionalValue(settings.Port),
		Username:             toOptionalValue(settings.Username),
		AuthType:             toOptionalValue(SshAuthType(settings.AuthType)),
		Password:             nil,
		PrivateKey:           nil,
		PrivateKeyPassphrase: nil,
	}
}
