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

type Upgrader struct {
	apiClient           *api.Client
	log                 *slog.Logger
	verifyBinaryTimeout time.Duration
}

type Deps struct {
	APIClient *api.Client
	Logger    *slog.Logger
}

func New(deps Deps) *Upgrader {
	return &Upgrader{apiClient: deps.APIClient, log: deps.Logger, verifyBinaryTimeout: 30 * time.Second}
}

// binaryDownloadTimeout bounds the agent self-download. The streaming HTTP
// client has no client-level timeout (it is shared with the long-lived backup
// stream), so the deadline is scoped per call here.
const binaryDownloadTimeout = 10 * time.Minute

// CheckAndUpdate checks if a new version is available and upgrades the binary on disk.
// Returns (true, nil) if the binary was upgraded, (false, nil) if already up to date,
// or (false, err) on failure. Callers are responsible for re-exec or restart signaling.
func (u *Upgrader) CheckAndUpdate(ctx context.Context, currentVersion string, isDev bool) (bool, error) {
	if isDev {
		u.log.InfoContext(ctx, "skipping update check (development mode)")

		return false, nil
	}

	serverVersion, err := u.apiClient.FetchServerVersion(ctx)
	if err != nil {
		u.log.WarnContext(ctx, "could not reach server for update check", slog.Any("error", err))

		return false, fmt.Errorf(
			"unable to check version, please verify Databasus server is available: %w",
			err,
		)
	}

	if serverVersion == currentVersion {
		u.log.InfoContext(ctx, "agent version is up to date", slog.Any("version", currentVersion))

		return false, nil
	}

	u.log.InfoContext(ctx, "updating agent...", slog.Any("current", currentVersion), slog.Any("target", serverVersion))

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

	downloadedVersion, err := u.apiClient.DownloadVerificationAgentBinary(
		downloadCtx, runtime.GOARCH, tempPath, serverVersion,
	)
	if err != nil {
		return false, fmt.Errorf("failed to download update: %w", err)
	}

	if err := os.Chmod(tempPath, 0o755); err != nil {
		return false, fmt.Errorf("failed to set permissions on update: %w", err)
	}

	if err := u.verifyBinary(ctx, tempPath, downloadedVersion); err != nil {
		return false, fmt.Errorf("update verification failed: %w", err)
	}

	if downloadedVersion == currentVersion {
		u.log.InfoContext(ctx, "agent version is up to date", slog.Any("version", currentVersion))
		return false, nil
	}

	if err := os.Rename(tempPath, selfPath); err != nil {
		return false, fmt.Errorf("failed to replace binary (try --skip-update if this persists): %w", err)
	}

	u.log.InfoContext(ctx, "agent binary updated", slog.Any("version", downloadedVersion))

	return true, nil
}

func (u *Upgrader) verifyBinary(ctx context.Context, binaryPath, expectedVersion string) error {
	ctx, cancel := context.WithTimeout(ctx, u.verifyBinaryTimeout)
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
