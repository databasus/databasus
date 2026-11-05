package sshtunnel

type AuthType string

const (
	AuthTypePassword   AuthType = "PASSWORD"
	AuthTypePrivateKey AuthType = "PRIVATE_KEY"
)

// Settings reach a bastion. The secrets are in clear text: the caller decrypts them where it reads
// them, immediately before opening the tunnel.
type Settings struct {
	Host                 string
	Port                 int
	Username             string
	AuthType             AuthType
	Password             string
	PrivateKey           string
	PrivateKeyPassphrase string
}
