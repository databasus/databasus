package mysql

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strconv"

	"databasus-backend/internal/engines/mysqlfamily"
	"databasus-backend/internal/lib/namelist"
)

const (
	// StoredZstdCompressionLevel is the zstd level the caller compresses a dump with before storing
	// it, because mysqldump writes plain SQL.
	StoredZstdCompressionLevel  = 5
	networkZstdCompressionLevel = 5
)

type DumpSpec struct {
	Version       Version
	ExcludeTables []string
	// Privileges are the backup privileges the account holds; TRIGGER and EVENT decide whether
	// triggers and events are dumped.
	Privileges []string
	Logger     *slog.Logger
}

// Dump writes the database as SQL into output until mysqldump exits.
func (s *Source) Dump(ctx context.Context, spec DumpSpec, output io.Writer) error {
	if s.settings.Database == "" {
		return ErrDumpDatabaseNameRequired
	}

	mysqldump, err := s.clientTools.GetExecutable(spec.Version, ExecutableMysqldump)
	if err != nil {
		return err
	}

	return mysqlfamily.Dump(ctx, mysqlfamily.DumpRun{
		Product:               "MySQL",
		DumpToolPath:          mysqldump,
		Connection:            s.getClientConnection(),
		DatabaseName:          s.settings.Database,
		Args:                  s.buildDumpArgs(spec),
		CompressionCandidates: getNetworkCompressionCandidates(spec.Version),
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
		"--set-gtid-purged=OFF",
		"--quick",
		"--skip-add-locks",
		"--verbose",
		// Dumping tablespace definitions makes mysqldump read INFORMATION_SCHEMA.FILES, which costs
		// a global PROCESS privilege that a backup role has no other use for and that managed
		// providers often refuse to grant.
		"--no-tablespaces",
	}

	// Triggers are on by default, so a role without the privilege has to be opted out explicitly
	// or mysqldump fails on SHOW TRIGGERS instead of dumping without them.
	if slices.Contains(spec.Privileges, "TRIGGER") {
		args = append(args, "--triggers")
	} else {
		args = append(args, "--skip-triggers")
	}

	if slices.Contains(spec.Privileges, "EVENT") {
		args = append(args, "--events")
	}

	for _, excludedTable := range namelist.NormalizeUniqueNames(spec.ExcludeTables) {
		args = append(args, "--ignore-table="+s.settings.Database+"."+excludedTable)
	}

	args = append(args, "--max-allowed-packet=1G", s.getSSLModeArg())

	return append(args, s.settings.Database)
}

// getNetworkCompressionCandidates ends with the uncompressed fallback. The deprecated --compress
// flag belongs to 5.7 alone; every later line is named explicitly, so a line added later does not
// quietly inherit the legacy flag.
func getNetworkCompressionCandidates(version Version) mysqlfamily.NetworkCompressionCandidates {
	if version == Version57 {
		return mysqlfamily.NetworkCompressionCandidates{
			Compressed: [][]string{{"--compress"}},
			ProbeArgs:  nil,
		}
	}

	return mysqlfamily.NetworkCompressionCandidates{
		Compressed: [][]string{
			{
				"--compression-algorithms=zstd",
				fmt.Sprintf("--zstd-compression-level=%d", networkZstdCompressionLevel),
			},
			{"--compression-algorithms=zlib"},
		},
		ProbeArgs: nil,
	}
}

func (s *Source) getClientConnection() mysqlfamily.ClientConnection {
	sslOption := "ssl-mode=DISABLED"
	if s.settings.IsHTTPS {
		sslOption = "ssl-mode=REQUIRED"
	}

	return mysqlfamily.ClientConnection{
		Host:      s.settings.Host,
		Port:      s.settings.Port,
		Username:  s.settings.Username,
		Password:  s.settings.Password,
		SSLOption: sslOption,
	}
}

func (s *Source) getSSLModeArg() string {
	if s.settings.IsHTTPS {
		return "--ssl-mode=REQUIRED"
	}

	return "--ssl-mode=DISABLED"
}
