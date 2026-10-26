package usecases_logical_mysql

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"

	mysql_engine "databasus-backend/internal/engines/mysql"
)

const (
	networkZstdCompressionLevel = 5
	compressionProbeTimeout     = 15 * time.Second
	compressionProbeTable       = "__databasus_compression_probe__"
)

// MyCnfFile holds the credentials and TLS mode, so the probe must not re-derive them.
type CompressionProbeSpec struct {
	MysqldumpBin string
	MyCnfFile    string
	DatabaseName string
	DatabaseID   uuid.UUID
}

// getNetworkCompressionCandidates is ordered best ratio first and ends with the uncompressed
// fallback. Managed providers that front MySQL with a proxy (e.g. Aliyun RDS) advertise algorithms
// the proxy cannot actually negotiate, so the choice has to come from a handshake rather than from
// the protocol_compression_algorithms server variable.
// The deprecated --compress flag belongs to 5.7 alone. Every later line is
// named explicitly rather than reached by falling through, so a line added
// later does not quietly inherit the legacy flag.
func getNetworkCompressionCandidates(version mysql_engine.Version) [][]string {
	if version == mysql_engine.Version57 {
		return [][]string{
			{"--compress"},
			{},
		}
	}

	return [][]string{
		{
			"--compression-algorithms=zstd",
			fmt.Sprintf("--zstd-compression-level=%d", networkZstdCompressionLevel),
		},
		{"--compression-algorithms=zlib"},
		{},
	}
}

func getNetworkCompressionLabel(candidate []string) string {
	if len(candidate) == 0 {
		return "uncompressed"
	}

	return strings.Join(candidate, " ")
}

// Error 2066 is CR_COMPRESSION_WRONGLY_CONFIGURED, raised client-side when the endpoint cannot
// negotiate the requested algorithm.
func isCompressionRejection(stderrStr string) bool {
	return containsIgnoreCase(stderrStr, "compression algorithm") ||
		containsIgnoreCase(stderrStr, "2066")
}

func (uc *CreateMysqlBackupUsecase) probeNetworkCompressionArgs(
	ctx context.Context,
	probe CompressionProbeSpec,
	version mysql_engine.Version,
) []string {
	candidates := getNetworkCompressionCandidates(version)
	compressedCandidates := candidates[:len(candidates)-1]
	chosenCompressionArgs := candidates[len(candidates)-1]

	for _, candidate := range compressedCandidates {
		if uc.isNetworkCompressionAccepted(ctx, probe, candidate) {
			chosenCompressionArgs = candidate
			break
		}
	}

	uc.logger.InfoContext(
		ctx,
		fmt.Sprintf("negotiated MySQL network compression: %s",
			getNetworkCompressionLabel(chosenCompressionArgs)),
		"database_id", probe.DatabaseID,
	)

	return chosenCompressionArgs
}

// isNetworkCompressionAccepted dials with mysqldump rather than the interactive client so the probe
// cannot disagree with the dump about which binary loads. Naming a table that cannot exist keeps
// the attempt O(1): the handshake — where compression is negotiated — happens first, then mysqldump
// gives up. Only a compression complaint counts as a rejection; any other failure (unreachable
// host, bad credentials) recurs identically for every candidate, so it is left for the dump to
// report with a precise message.
func (uc *CreateMysqlBackupUsecase) isNetworkCompressionAccepted(
	ctx context.Context,
	probe CompressionProbeSpec,
	candidate []string,
) bool {
	probeCtx, cancel := context.WithTimeout(ctx, compressionProbeTimeout)
	defer cancel()

	args := append([]string{"--defaults-file=" + probe.MyCnfFile}, candidate...)
	args = append(args, "--no-data", probe.DatabaseName, compressionProbeTable)

	var probeStderr bytes.Buffer

	// The binary is a database client from the bundled tools directory, and the arguments go to it as a list, not through a shell.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(probeCtx, probe.MysqldumpBin, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = &probeStderr
	cmd.Env = append(os.Environ(),
		"MYSQL_PWD=",
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	)

	if err := cmd.Run(); err != nil && isCompressionRejection(probeStderr.String()) {
		uc.logger.DebugContext(
			ctx,
			fmt.Sprintf("MySQL rejected network compression: %s",
				getNetworkCompressionLabel(candidate)),
			"database_id", probe.DatabaseID,
			"stderr", probeStderr.String(),
		)

		return false
	}

	return true
}
