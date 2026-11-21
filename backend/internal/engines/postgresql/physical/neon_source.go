package physical

import "strings"

const neonDomain = "neon.tech"

// validateNotNeonSource judges the configured source host, not the tunnel address: Neon accepts no
// physical replication connections, so a physical backup of a Neon database cannot run whatever
// the user configures, and the refusal must not wait for a connection to fail.
func (s *Source) validateNotNeonSource() error {
	if !isNeonHost(s.settings.Host) {
		return nil
	}

	return newNeonReplicationUnsupportedError()
}

func isNeonHost(host string) bool {
	normalizedHost := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")

	return normalizedHost == neonDomain || strings.HasSuffix(normalizedHost, "."+neonDomain)
}
