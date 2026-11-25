package container

import (
	"context"
	"log/slog"
)

// ImageTag is one local tag of an image repository.
type ImageTag struct {
	Reference string
	Tag       string
}

type verificationImageEngine interface {
	ListImageTags(ctx context.Context, repository string) ([]ImageTag, error)
	RemoveImageTag(ctx context.Context, reference string) error
}

// removeSupersededVerificationImages removes every local tag of the
// verification image repository that is not a pinned PostgreSQL version: the
// old "<major>" tags and earlier versions. Pinning by version gives every
// version raise a new image of more than a gigabyte, and nothing else removes
// them.
//
// The official postgres repository is left alone: it is shared with whatever
// else runs on the host, and the agent cannot tell its own pulls from the
// owner's. Removal is by tag and never forced, so an image a container uses
// stays and an image with another tag keeps it. Best-effort, like the
// container purge: a failure is logged and never blocks startup.
func removeSupersededVerificationImages(
	ctx context.Context,
	engine verificationImageEngine,
	verificationRepository string,
	pinnedVersions PinnedPostgresVersions,
	log *slog.Logger,
) {
	if isOfficialPostgresRepository(verificationRepository) {
		return
	}

	localTags, err := engine.ListImageTags(ctx, verificationRepository)
	if err != nil {
		log.WarnContext(
			ctx,
			"could not list verification images; superseded images stay on the host",
			slog.Any("repository", verificationRepository),
			slog.Any("error", err),
		)

		return
	}

	removed := 0
	for _, localTag := range localTags {
		if pinnedVersions.IsPinnedPostgresVersion(localTag.Tag) {
			continue
		}

		if err := engine.RemoveImageTag(ctx, localTag.Reference); err != nil {
			log.WarnContext(
				ctx,
				"could not remove superseded verification image",
				slog.Any("image", localTag.Reference),
				slog.Any("error", err),
			)

			continue
		}

		removed++
	}

	if removed > 0 {
		log.InfoContext(ctx, "removed superseded verification images", slog.Any("removed", removed))
	}
}
