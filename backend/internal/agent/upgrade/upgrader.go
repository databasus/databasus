package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"

	"databasus-backend/internal/agent/api"
	"databasus-backend/internal/lib/process"
)

// verifyBinaryTimeout bounds the freshly-downloaded binary's `version` probe so
// a corrupt or wrong-arch binary that hangs cannot wedge startup or the
// background-upgrade goroutine. A var so tests can shrink it.
var verifyBinaryTimeout = 30 * time.Second

// binaryDownloadTimeout bounds the agent self-download. The streaming HTTP
// client has no client-level timeout (it is shared with the long-lived backup
// stream), so the deadline is scoped per call here.
const binaryDownloadTimeout = 10 * time.Minute

// CheckAndUpdate checks if a new version is available and upgrades the binary on disk.
// Returns (true, nil) if the binary was upgraded, (false, nil) if already up to date,
// or (false, err) on failure. Callers are responsible for re-exec or restart signaling.
func CheckAndUpdate(
	ctx context.Context,
	apiClient *api.Client,
	currentVersion string,
	isDev bool,
	log *slog.Logger,
) (bool, error) {
	if isDev {
		log.InfoContext(ctx, "skipping update check (development mode)")

		return false, nil
	}

	serverVersion, err := apiClient.FetchServerVersion(ctx)
	if err != nil {
		log.WarnContext(ctx, "could not reach server for update check", slog.Any("error", err))

		return false, fmt.Errorf(
			"unable to check version, please verify Databasus server is available: %w",
			err,
		)
	}

	if serverVersion == currentVersion {
		log.InfoContext(ctx, "agent version is up to date", slog.Any("version", currentVersion))

		return false, nil
	}

	log.InfoContext(ctx, "updating agent...", slog.Any("current", currentVersion), slog.Any("target", serverVersion))

	selfPath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("failed to determine executable path: %w", err)
	}

	tempPath := selfPath + ".update"

	defer func() {
		_ = os.Remove(tempPath)
	}()

	downloadCtx, cancelDownload := context.WithTimeout(ctx, binaryDownloadTimeout)
	defer cancelDownload()

	downloadedVersion, err := apiClient.DownloadVerificationAgentBinary(
		downloadCtx, runtime.GOARCH, tempPath, serverVersion,
	)
	if err != nil {
		return false, fmt.Errorf("failed to download update: %w", err)
	}

	if err := os.Chmod(tempPath, 0o755); err != nil {
		return false, fmt.Errorf("failed to set permissions on update: %w", err)
	}

	if err := verifyBinary(ctx, tempPath, downloadedVersion); err != nil {
		return false, fmt.Errorf("update verification failed: %w", err)
	}

	if downloadedVersion == currentVersion {
		log.InfoContext(ctx, "agent version is up to date", slog.Any("version", currentVersion))
		return false, nil
	}

	if err := os.Rename(tempPath, selfPath); err != nil {
		return false, fmt.Errorf("failed to replace binary (try --skip-update if this persists): %w", err)
	}

	log.InfoContext(ctx, "agent binary updated", slog.Any("version", downloadedVersion))

	return true, nil
}

func verifyBinary(ctx context.Context, binaryPath, expectedVersion string) error {
	ctx, cancel := context.WithTimeout(ctx, verifyBinaryTimeout)
	defer cancel()

	var output bytes.Buffer
	err := process.Run(ctx, process.Spec{Path: binaryPath, Args: []string{"version"}, Stdout: &output})
	if err != nil {
		return fmt.Errorf("binary failed to execute: %w", err)
	}

	reportedVersion := strings.TrimSpace(output.String())
	if reportedVersion != expectedVersion {
		return fmt.Errorf("version mismatch: expected %q, got %q", expectedVersion, reportedVersion)
	}

	return nil
}
