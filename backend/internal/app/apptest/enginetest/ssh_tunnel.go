package enginetest

import (
	"log/slog"
	"testing"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/sshtunnel"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testcontainers/bastion"
)

func NewDisabledSSHTunnelSettings() apiclient.SshTunnelSettings {
	return apiclient.SshTunnelSettings{
		IsEnabled:            false,
		AuthType:             nil,
		Host:                 nil,
		Port:                 nil,
		Username:             nil,
		Password:             nil,
		PrivateKey:           nil,
		PrivateKeyPassphrase: nil,
	}
}

func NewBastionSSHTunnelSettings(topology containers.BastionedDatabase) apiclient.SshTunnelSettings {
	tunnelSettings := bastion.GetTunnelSettings(topology)
	authType := apiclient.SshAuthType(tunnelSettings.AuthType)

	return apiclient.SshTunnelSettings{
		IsEnabled:            true,
		Host:                 &tunnelSettings.Host,
		Port:                 &tunnelSettings.Port,
		Username:             &tunnelSettings.Username,
		AuthType:             &authType,
		Password:             getNilIfEmpty(tunnelSettings.Password),
		PrivateKey:           getNilIfEmpty(tunnelSettings.PrivateKey),
		PrivateKeyPassphrase: getNilIfEmpty(tunnelSettings.PrivateKeyPassphrase),
	}
}

// OpenForwardedEndpoint forwards a local port to the bastioned database until the test ends. The test
// has no other route to the database, so it reaches it the way a backup does.
func OpenForwardedEndpoint(t *testing.T, topology containers.BastionedDatabase) containers.Endpoint {
	t.Helper()

	forwarder, err := sshtunnel.Open(t.Context(), sshtunnel.OpenSpec{
		Settings: bastion.GetTunnelSettings(topology),
		Target:   sshtunnel.Endpoint{Host: topology.Database.Host, Port: topology.Database.Port},
		Logger:   slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatalf("open the SSH tunnel to the bastioned database: %v", err)
	}

	t.Cleanup(forwarder.Close)

	localEndpoint := forwarder.GetLocalEndpoint()

	return containers.Endpoint{Host: localEndpoint.Host, Port: localEndpoint.Port}
}

func getNilIfEmpty(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}
