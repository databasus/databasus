package containers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

// CNPGCertificateMode selects the server certificate a source presents. The two refused modes keep
// the CloudNativePG cluster consistent internally, and the harness verifies its own connections
// against what the certificate really is; only what Databasus is given to verify differs.
type CNPGCertificateMode string

const (
	CNPGCertificateTrusted        CNPGCertificateMode = "trusted"
	CNPGCertificateForeign        CNPGCertificateMode = "foreign-authority"
	CNPGCertificateWithoutAddress CNPGCertificateMode = "without-address"
)

// RotateServerCertificate replaces the certificate the source presents and waits until the routed
// primary serves it. Databasus refuses to save a source whose certificate it cannot verify, so a
// refused certificate reaches a saved source only through such a rotation.
func (cluster *CNPGCluster) RotateServerCertificate(t *testing.T, mode CNPGCertificateMode) {
	t.Helper()

	cluster.applyServerCertificate(t, mode)
	block, _ := pem.Decode([]byte(cluster.ServerCertificate.CertificatePEM))
	require.NotNil(t, block)

	testwait.Until(t, "the routed primary presents the rotated certificate", func(ctx context.Context) error {
		presented, err := cluster.getPresentedCertificate(ctx, t)
		if err != nil {
			return err
		}

		if !bytes.Equal(presented, block.Bytes) {
			return fmt.Errorf("the routed primary still presents the previous certificate")
		}

		return nil
	})
}

// applyServerCertificate issues a certificate for the mode and stores it with its authority in the
// secrets the Cluster manifest names. The loopback address lets the test proxy present the same
// certificate (see WithProxyTLS).
func (cluster *CNPGCluster) applyServerCertificate(t *testing.T, mode CNPGCertificateMode) {
	t.Helper()

	authority := cluster.Authority
	addresses := []string{cluster.K3s.Address, loopbackIP}
	verifiedName := ""
	switch mode {
	case CNPGCertificateTrusted:
	case CNPGCertificateForeign:
		var err error
		authority, err = NewTestCertificateAuthority()
		require.NoError(t, err)
	case CNPGCertificateWithoutAddress:
		addresses = []string{loopbackIP}
		verifiedName = cluster.getServiceNames()[0]
	default:
		t.Fatalf("unknown certificate mode %s", mode)
	}

	certificate, err := authority.IssueServerCertificate(ServerCertificateSpec{
		DNSNames:    cluster.getServiceNames(),
		IPAddresses: addresses,
	})
	require.NoError(t, err)

	for _, secret := range []map[string]any{
		cluster.getSecret(cnpgSecretSpec{Name: "server-ca", Kind: "Opaque", Values: map[string]string{"ca.crt": authority.CertificatePEM}}),
		cluster.getSecret(cnpgSecretSpec{Name: "server-tls", Kind: "kubernetes.io/tls", Values: map[string]string{
			"tls.crt": certificate.CertificatePEM,
			"tls.key": certificate.KeyPEM,
		}}),
	} {
		cluster.apply(t, secret)
	}

	cluster.issuer = authority
	cluster.verifiedName = verifiedName
	cluster.ServerCertificate = certificate
}

func (cluster *CNPGCluster) getPresentedCertificate(ctx context.Context, t *testing.T) ([]byte, error) {
	t.Helper()

	var presented []byte
	config := cluster.getConnectionConfig(t, routeLogin{Username: PostgresUsername, Database: CNPGDatabase})
	config.TLSConfig.VerifyConnection = func(state tls.ConnectionState) error {
		presented = state.PeerCertificates[0].Raw

		return nil
	}

	connection, err := pgconn.ConnectConfig(ctx, &config.Config)
	if err != nil {
		return nil, err
	}

	_ = connection.Close(context.WithoutCancel(ctx))

	return presented, nil
}
