// Package bastion builds the SSH tunnel configuration that reaches a bastioned test topology.
//
// It sits apart from containers because containers must not import features: the old database
// models embed sshtunnel.Config from features, and their tests import containers, so the reverse
// edge would be an import cycle.
package bastion

import (
	"databasus-backend/internal/features/sshtunnel"
	libsshtunnel "databasus-backend/internal/lib/sshtunnel"
	"databasus-backend/internal/util/testing/containers"
)

func GetTunnelConfig(bastionedDatabase containers.BastionedDatabase) sshtunnel.Config {
	return sshtunnel.Config{
		IsEnabled: true,
		Host:      bastionedDatabase.Bastion.Host,
		Port:      bastionedDatabase.Bastion.Port,
		Username:  containers.SshBastionUsername,
		AuthType:  sshtunnel.AuthTypePassword,
		Password:  containers.SshBastionPassword,
	}
}

func GetTunnelSettings(bastionedDatabase containers.BastionedDatabase) libsshtunnel.Settings {
	return libsshtunnel.Settings{
		Host:                 bastionedDatabase.Bastion.Host,
		Port:                 bastionedDatabase.Bastion.Port,
		Username:             containers.SshBastionUsername,
		AuthType:             libsshtunnel.AuthTypePassword,
		Password:             containers.SshBastionPassword,
		PrivateKey:           "",
		PrivateKeyPassphrase: "",
	}
}
