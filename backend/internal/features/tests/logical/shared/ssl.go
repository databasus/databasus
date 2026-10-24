//go:build testhelpers

package logicaltesting

import (
	"crypto/tls"
	"sync"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
)

// The MySQL driver panics on duplicate RegisterTLSConfig calls, so guard with sync.Once.
var sslMysqlTLSConfigOnce sync.Once

// SSLMysqlTLSConfigName is the name registered with the MySQL driver for the
// SSL test connections (InsecureSkipVerify). Shared by the MySQL and MariaDB SSL
// tests, which both use the MySQL driver.
const SSLMysqlTLSConfigName = "ssl-test-skip-verify"

// RegisterSSLMysqlTLSConfig registers an insecure-skip-verify TLS config with the
// MySQL driver exactly once per process.
func RegisterSSLMysqlTLSConfig(t *testing.T) {
	t.Helper()
	sslMysqlTLSConfigOnce.Do(func() {
		err := mysqldriver.RegisterTLSConfig(SSLMysqlTLSConfigName, &tls.Config{
			InsecureSkipVerify: true,
		})
		if err != nil {
			t.Fatalf("failed to register MySQL TLS config: %v", err)
		}
	})
}
