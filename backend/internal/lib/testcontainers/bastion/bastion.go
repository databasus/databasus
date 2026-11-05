// Package bastion builds the SSH tunnel settings that reach a bastioned test topology.
//
// It sits apart from containers because the tests of lib/sshtunnel start their bastion through
// containers, so containers importing lib/sshtunnel would be an import cycle.
package bastion

import (
	"databasus-backend/internal/lib/sshtunnel"
	containers "databasus-backend/internal/lib/testcontainers"
)

func GetTunnelSettings(bastionedDatabase containers.BastionedDatabase) sshtunnel.Settings {
	return sshtunnel.Settings{
		Host:                 bastionedDatabase.Bastion.Host,
		Port:                 bastionedDatabase.Bastion.Port,
		Username:             containers.SSHBastionUsername,
		AuthType:             sshtunnel.AuthTypePassword,
		Password:             containers.SSHBastionPassword,
		PrivateKey:           "",
		PrivateKeyPassphrase: "",
	}
}
