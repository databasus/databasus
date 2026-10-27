package mysqlfamily

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"time"
)

const (
	compressionProbeTimeout = 15 * time.Second
	compressionProbeTable   = "__databasus_compression_probe__"
)

// NetworkCompressionCandidates are tried in order, best ratio first. Managed providers that front
// the server with a proxy, such as Aliyun RDS, advertise algorithms the proxy cannot negotiate, so
// the choice comes from a handshake rather than from what the server advertises.
type NetworkCompressionCandidates struct {
	// Compressed hold the arguments of each algorithm; the uncompressed fallback has none.
	Compressed [][]string
	// ProbeArgs follow a candidate in the probe only, such as an option the .my.cnf cannot carry.
	ProbeArgs []string
}

type compressionProbe struct {
	product      string
	dumpToolPath string
	configFile   *clientConfigFile
	databaseName string
	candidates   NetworkCompressionCandidates
	logger       *slog.Logger
}

// negotiateNetworkCompression returns the arguments of the first candidate the server accepts, or
// none for the uncompressed fallback.
func negotiateNetworkCompression(ctx context.Context, probe compressionProbe) []string {
	var chosenCompressionArgs []string

	for _, candidate := range probe.candidates.Compressed {
		if isNetworkCompressionAccepted(ctx, probe, candidate) {
			chosenCompressionArgs = candidate

			break
		}
	}

	probe.logger.InfoContext(ctx, "negotiated network compression",
		slog.String("product", probe.product),
		slog.String("compression", getNetworkCompressionLabel(chosenCompressionArgs)),
	)

	return chosenCompressionArgs
}

// isNetworkCompressionAccepted dials with the dump tool rather than the interactive client, so the
// probe cannot disagree with the dump about which binary loads. Naming a table that cannot exist
// keeps the attempt short: the handshake, where compression is negotiated, happens first, then the
// tool gives up. Only a compression complaint counts as a rejection; any other failure, such as an
// unreachable host or bad credentials, recurs for every candidate, so the dump reports it.
func isNetworkCompressionAccepted(ctx context.Context, probe compressionProbe, candidate []string) bool {
	probeCtx, cancel := context.WithTimeout(ctx, compressionProbeTimeout)
	defer cancel()

	args := append([]string{probe.configFile.getDefaultsFileArg()}, candidate...)
	args = append(args, probe.candidates.ProbeArgs...)
	args = append(args, "--no-data", probe.databaseName, compressionProbeTable)

	var probeStderr bytes.Buffer

	err := runClientTool(probeCtx, clientToolRun{
		path:   probe.dumpToolPath,
		args:   args,
		stdin:  nil,
		stdout: io.Discard,
		stderr: &probeStderr,
	})
	if err != nil && isCompressionRejection(probeStderr.String()) {
		probe.logger.DebugContext(ctx, "server rejected network compression",
			slog.String("product", probe.product),
			slog.String("compression", getNetworkCompressionLabel(candidate)),
			slog.String("stderr", probeStderr.String()),
		)

		return false
	}

	return true
}

func getNetworkCompressionLabel(candidate []string) string {
	if len(candidate) == 0 {
		return "uncompressed"
	}

	return strings.Join(candidate, " ")
}
