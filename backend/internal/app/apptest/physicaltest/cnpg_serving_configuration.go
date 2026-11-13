package physicaltest

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

type servingConfiguration struct {
	IsSSLOn         bool
	CertificateHash string
	HBAHash         string
}

func (source cnpgSource) getServingConfiguration(t *testing.T) servingConfiguration {
	t.Helper()

	output := source.Fixture.QueryNode(t, containers.NodeQuery{
		Node: source.Fixture.GetIdentity(t).Node,
		SQL: "SELECT current_setting('ssl') = 'on', md5(pg_read_file(current_setting('ssl_cert_file'))), " +
			"md5(pg_read_file(current_setting('hba_file')))",
	})
	fields := strings.Split(output, "|")
	require.Len(t, fields, 3, "serving configuration %q", output)
	configuration := servingConfiguration{IsSSLOn: fields[0] == "t", CertificateHash: fields[1], HBAHash: fields[2]}
	require.True(t, configuration.IsSSLOn)

	return configuration
}
