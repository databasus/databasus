package mariadb

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"strconv"

	"databasus-backend/internal/engines/mysqlfamily"
	"databasus-backend/internal/lib/namelist"
)

// StoredZstdCompressionLevel is the zstd level the caller compresses a dump with before storing it,
// because mariadb-dump writes plain SQL.
const StoredZstdCompressionLevel = 5

type DumpSpec struct {
	Version       Version
	ExcludeTables []string
	// Privileges are the backup privileges the account holds; TRIGGER and EVENT decide whether
	// triggers and events are dumped.
	Privileges      []string
	IsExcludeEvents bool
	Logger          *slog.Logger
}

// Dump writes the database as SQL into output until mariadb-dump exits.
func (s *Source) Dump(ctx context.Context, spec DumpSpec, output io.Writer) error {
	if s.settings.Database == "" {
		return ErrDumpDatabaseNameRequired
	}

	mariadbDump, err := s.clientTools.GetExecutable(spec.Version, ExecutableMariadbDump)
	if err != nil {
		return err
	}

	return mysqlfamily.Dump(ctx, mysqlfamily.DumpRun{
		Product:               "MariaDB",
		DumpToolPath:          mariadbDump,
		Connection:            s.getClientConnection(),
		DatabaseName:          s.settings.Database,
		Args:                  s.buildDumpArgs(spec),
		CompressionCandidates: s.getNetworkCompressionCandidates(),
		Logger:                spec.Logger,
	}, output)
}

func (s *Source) buildDumpArgs(spec DumpSpec) []string {
	args := []string{
		"--host=" + s.settings.Host,
		"--port=" + strconv.Itoa(s.settings.Port),
		"--user=" + s.settings.Username,
		"--single-transaction",
		"--routines",
		"--quick",
		"--skip-add-locks",
		"--verbose",
		// Dumping tablespace definitions makes mariadb-dump read INFORMATION_SCHEMA.FILES, which
		// costs a global PROCESS privilege that a backup role has no other use for and that managed
		// providers often refuse to grant.
		"--no-tablespaces",
	}

	// Triggers are on by default, so a role without the privilege has to be opted out explicitly
	// or mariadb-dump fails on SHOW TRIGGERS instead of dumping without them.
	if slices.Contains(spec.Privileges, "TRIGGER") {
		args = append(args, "--triggers")
	} else {
		args = append(args, "--skip-triggers")
	}

	if slices.Contains(spec.Privileges, "EVENT") && !spec.IsExcludeEvents {
		args = append(args, "--events")
	}

	for _, excludedTable := range namelist.NormalizeUniqueNames(spec.ExcludeTables) {
		args = append(args, "--ignore-table="+s.settings.Database+"."+excludedTable)
	}

	args = append(args, "--max-allowed-packet=1G")
	args = append(args, s.getSSLArgs()...)

	return append(args, s.settings.Database)
}

// getNetworkCompressionCandidates has the one algorithm MariaDB's protocol offers, zlib. The
// certificate opt-out has no .my.cnf equivalent, so the probe passes it as an argument.
func (s *Source) getNetworkCompressionCandidates() mysqlfamily.NetworkCompressionCandidates {
	var probeArgs []string
	if s.settings.IsHTTPS {
		probeArgs = []string{"--skip-ssl-verify-server-cert"}
	}

	return mysqlfamily.NetworkCompressionCandidates{
		Compressed: [][]string{{"--compress"}},
		ProbeArgs:  probeArgs,
	}
}

func (s *Source) getClientConnection() mysqlfamily.ClientConnection {
	sslOption := "ssl=false"
	if s.settings.IsHTTPS {
		sslOption = "ssl=true"
	}

	return mysqlfamily.ClientConnection{
		Host:      s.settings.Host,
		Port:      s.settings.Port,
		Username:  s.settings.Username,
		Password:  s.settings.Password,
		SSLOption: sslOption,
	}
}

func (s *Source) getSSLArgs() []string {
	if s.settings.IsHTTPS {
		return []string{"--ssl", "--skip-ssl-verify-server-cert"}
	}

	return []string{"--skip-ssl"}
}
