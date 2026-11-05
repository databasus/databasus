package physical

import pgshared "databasus-backend/internal/engines/postgresql/shared"

func (s *Source) getClientEnvironment(creds *pgshared.CredentialTempFiles) []string {
	sslMode := s.settings.TLS.Mode
	if sslMode == "" {
		sslMode = pgshared.SslModeDisable
	}

	env := []string{
		"PGPASSFILE=" + creds.PgpassPath,
		"PGCLIENTENCODING=UTF8",
		"PGCONNECT_TIMEOUT=30",
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
		"PGSSLMODE=" + string(
			sslMode,
		),
		"PGSSLCERT=" + creds.ClientCertPath,
		"PGSSLKEY=" + creds.ClientKeyPath,
		"PGSSLROOTCERT=" + creds.RootCertPath,
		"PGSSLCRL=",
	}

	return append(env, pgshared.GetPgHostAddrEnv(s.settings.HostAddr)...)
}
