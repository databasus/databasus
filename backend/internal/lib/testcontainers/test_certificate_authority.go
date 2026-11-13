package containers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

const testCertificateLifetime = 24 * time.Hour

// TestCertificateAuthority issues server certificates for one test run. Its key lives only in
// memory, so nothing it signs is trusted outside that run.
type TestCertificateAuthority struct {
	CertificatePEM string
	certificate    *x509.Certificate
	key            *ecdsa.PrivateKey
}

type ServerCertificate struct {
	CertificatePEM string
	KeyPEM         string
}

type ServerCertificateSpec struct {
	DNSNames    []string
	IPAddresses []string
}

func NewTestCertificateAuthority() (*TestCertificateAuthority, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate the test authority key: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber:          getSerialNumber(),
		Subject:               pkix.Name{CommonName: "Databasus test authority"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(testCertificateLifetime),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("sign the test authority: %w", err)
	}

	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("parse the test authority: %w", err)
	}

	return &TestCertificateAuthority{
		CertificatePEM: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		certificate:    certificate,
		key:            key,
	}, nil
}

func (authority *TestCertificateAuthority) IssueServerCertificate(
	spec ServerCertificateSpec,
) (ServerCertificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return ServerCertificate{}, fmt.Errorf("generate a server key: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: getSerialNumber(),
		Subject:      pkix.Name{CommonName: spec.DNSNames[0]},
		DNSNames:     spec.DNSNames,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(testCertificateLifetime),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	for _, address := range spec.IPAddresses {
		template.IPAddresses = append(template.IPAddresses, net.ParseIP(address))
	}

	der, err := x509.CreateCertificate(rand.Reader, template, authority.certificate, &key.PublicKey, authority.key)
	if err != nil {
		return ServerCertificate{}, fmt.Errorf("sign a server certificate: %w", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return ServerCertificate{}, fmt.Errorf("encode a server key: %w", err)
	}

	return ServerCertificate{
		CertificatePEM: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		KeyPEM:         string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})),
	}, nil
}

// GetClientTLS verifies a server against this authority under the name the client dials, as libpq's
// verify-full does.
func (authority *TestCertificateAuthority) GetClientTLS(serverName string) *tls.Config {
	roots := x509.NewCertPool()
	roots.AddCert(authority.certificate)

	return &tls.Config{RootCAs: roots, ServerName: serverName, MinVersion: tls.VersionTLS12}
}

func getSerialNumber() *big.Int {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
	if err != nil {
		panic(fmt.Sprintf("read random serial number: %v", err))
	}

	return serial
}
