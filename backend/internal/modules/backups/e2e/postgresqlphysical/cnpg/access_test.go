package cnpg_test

import (
	"testing"

	"databasus-backend/internal/app/apptest/physicaltest"
	containers "databasus-backend/internal/lib/testcontainers"
)

func TestCNPGAccessPostgres17(t *testing.T) {
	t.Parallel()
	runCNPGScenarios(t, postgres17, getAccessScenarios())
}

func TestCNPGAccessPostgres18(t *testing.T) {
	t.Parallel()
	runCNPGScenarios(t, postgres18, getAccessScenarios())
}

func getAccessScenarios() []cnpgScenario {
	return []cnpgScenario{
		{
			name: "keeps the declared role restricted across a switch with ",
			run:  physicaltest.RunCNPGRestrictedAccessAcrossSwitch,
		},
		{
			name: "refuses a source certificate from another authority with ",
			run: func(t *testing.T, spec physicaltest.CNPGManagerSpec) {
				t.Helper()
				physicaltest.RunCNPGRefusedCertificate(t, spec, physicaltest.CNPGCertificateRefusal{
					Mode:              containers.CNPGCertificateForeign,
					ConnectionFailure: "tls: failed to verify certificate: x509: certificate signed by unknown authority",
					BackupFailure:     "tls: failed to verify certificate: x509: certificate signed by unknown authority",
				})
			},
		},
		{
			name: "refuses a source certificate that does not name the address with ",
			run: func(t *testing.T, spec physicaltest.CNPGManagerSpec) {
				t.Helper()
				physicaltest.RunCNPGRefusedCertificate(t, spec, physicaltest.CNPGCertificateRefusal{
					Mode:              containers.CNPGCertificateWithoutAddress,
					ConnectionFailure: "tls: failed to verify certificate: x509: certificate is valid for 127.0.0.1, not ",
					BackupFailure:     "tls: failed to verify certificate: x509: certificate is valid for 127.0.0.1, not ",
				})
			},
		},
		{
			name: "rejects replication without TLS with ",
			run:  physicaltest.RunCNPGPlaintextReplicationRefusal,
		},
	}
}
