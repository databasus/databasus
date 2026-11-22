package physical

import pgshared "databasus-backend/internal/engines/postgresql/shared"

// validateNotNeonSource judges the configured source host, not the tunnel address: Neon accepts no
// physical replication connections, so a physical backup of a Neon database cannot run whatever
// the user configures, and the refusal must not wait for a connection to fail.
func (s *Source) validateNotNeonSource() error {
	if !pgshared.IsNeonHost(s.settings.Host) {
		return nil
	}

	return newNeonReplicationUnsupportedError()
}
