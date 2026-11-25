package container

import (
	"context"
	"fmt"
	"log/slog"
)

const (
	// stockEngineImageRepo is the official Postgres image. It is the fallback
	// when the configured (extension-bundling) repo can't be pulled, and it
	// publishes the same "<major>.<minor>" tags, so the fallback keeps the
	// pinned server version.
	stockEngineImageRepo = "postgres"

	// timescaleImageRepo carries the timescaledb extension; the tag is
	// "<extversion>-pg<major>" so the engine matches the dump exactly
	// (pg_restore cannot cross extension versions).
	timescaleImageRepo = "timescale/timescaledb"
)

type imageEngine interface {
	EnsureImage(ctx context.Context, ref string) error
}

type jobImageRequest struct {
	spawn           SpawnRequest
	engineImageRepo string
	pinnedVersions  PinnedPostgresVersions
}

// ensureJobImage makes the job's restore image available locally and returns
// its reference. The image is addressed by the pinned server version, never by
// the major alone, so an image pulled for an older version is never reused.
func ensureJobImage(
	ctx context.Context, engine imageEngine, request jobImageRequest, log *slog.Logger,
) (string, error) {
	image, err := imageForJob(request)
	if err != nil {
		return "", err
	}

	pullErr := engine.EnsureImage(ctx, image)
	if pullErr == nil {
		return image, nil
	}

	fallbackImage, canFallBack, err := stockFallbackImage(request)
	if err != nil {
		return "", err
	}

	if !canFallBack {
		return "", fmt.Errorf("ensure image: %w", pullErr)
	}

	// The extension-bundling image is unreachable (registry down, air-gapped,
	// not yet published, or a configured repo without this version's tag).
	// Missing extensions then surface as ignored pg_restore errors the runner
	// tolerates.
	log.WarnContext(
		ctx,
		"bundled verification image unavailable; falling back to stock postgres image",
		slog.Any("bundled_image", image),
		slog.Any("stock_image", fallbackImage),
		slog.Any("error", pullErr),
	)

	if err := engine.EnsureImage(ctx, fallbackImage); err != nil {
		return "", fmt.Errorf("ensure stock image: %w", err)
	}

	return fallbackImage, nil
}

func imageForJob(request jobImageRequest) (string, error) {
	if request.spawn.TimescaledbVersion != "" {
		return fmt.Sprintf(
			"%s:%s-pg%s", timescaleImageRepo, request.spawn.TimescaledbVersion, request.spawn.PgMajor,
		), nil
	}

	return pinnedImage(request.engineImageRepo, request)
}

// stockFallbackImage decides whether a failed pull of the configured image may
// retry against the official Postgres image. Timescale jobs need their exact
// version-matched image (pg_restore cannot cross extension versions), so they
// never fall back; nor does a job already targeting the official repo.
func stockFallbackImage(request jobImageRequest) (string, bool, error) {
	if request.spawn.TimescaledbVersion != "" {
		return "", false, nil
	}

	if isOfficialPostgresRepository(request.engineImageRepo) {
		return "", false, nil
	}

	image, err := pinnedImage(stockEngineImageRepo, request)
	if err != nil {
		return "", false, err
	}

	return image, true, nil
}

func pinnedImage(repository string, request jobImageRequest) (string, error) {
	version, isPinned := request.pinnedVersions.GetPinnedPostgresVersion(request.spawn.PgMajor)
	if !isPinned {
		return "", fmt.Errorf("no pinned PostgreSQL version for major %q", request.spawn.PgMajor)
	}

	return repository + ":" + version, nil
}
