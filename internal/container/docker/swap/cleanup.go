package swap

import (
	"context"
	"strings"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/imagecheck"
	dcontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
	"github.com/rs/zerolog/log"
)

// ImageAPI is the slice of the moby client CleanupImage uses. Removal is by
// id, never forced and never pruning children.
type ImageAPI interface {
	ImageInspect(ctx context.Context, imageID string, opts ...client.ImageInspectOption) (client.ImageInspectResult, error)
	ImageRemove(ctx context.Context, imageID string, options client.ImageRemoveOptions) (client.ImageRemoveResult, error)
}

// PreviousLabels are the labels an update stamps on the replacement for old,
// which ran oldImage (nil when it could not be inspected) and follows ref: the
// image id it replaced, and that image's repo@sha256 digest. The digest is
// empty for an image built locally, which removes a stale one the old
// container carried from its own update. A rollback's mark is cleared: the
// container moved on from the image it was rolled back from.
func PreviousLabels(old dcontainer.InspectResponse, oldImage *image.InspectResponse, ref string) map[string]string {
	return map[string]string{
		container.PreviousImageLabel:  old.Image,
		container.PreviousRefLabel:    PreviousRef(oldImage, ref),
		container.RolledBackFromLabel: "",
	}
}

// PreviousRef is img as repo@sha256:digest for the repository ref names,
// falling back to any digest the image has. Empty for an image built locally.
func PreviousRef(img *image.InspectResponse, ref string) string {
	if img == nil || len(img.RepoDigests) == 0 {
		return ""
	}
	if want, err := imagecheck.ParseReference(ref); err == nil {
		for _, digest := range img.RepoDigests {
			if got, err := imagecheck.ParseReference(digest); err == nil && got.Registry == want.Registry && got.Repository == want.Repository {
				return digest
			}
		}
	}
	return img.RepoDigests[0]
}

// CleanupImage runs after an update committed. It removes the image old, the
// outgoing container, had itself replaced, read from its previous-image label.
// The image old ran until now is kept as the rollback target, so at most one
// spare image per container stays behind. The removal follows
// RemoveLeftoverImage.
func CleanupImage(ctx context.Context, cli ImageAPI, old dcontainer.InspectResponse) {
	var previous string
	if old.Config != nil {
		previous = old.Config.Labels[container.PreviousImageLabel]
	}
	if previous == "" {
		log.Debug().Str("container", strings.TrimPrefix(old.Name, "/")).Msg("update cleanup: nothing to remove, the container has no previous image yet")
		return
	}
	if previous == old.Image {
		// Never the image just replaced: it is the rollback target. An image
		// the replacement runs is refused by the engine below.
		return
	}
	RemoveLeftoverImage(ctx, cli, old, previous, "the image before the previous one")
}

// RemoveLeftoverImage removes imageID once the swap that replaced old, an
// update or a rollback, committed; what names the image in the log. Only a
// leftover is removed: an image that still has a tag is skipped, since the
// engine would untag and delete it when its tags share one repository. An
// update leaves the old image untagged anyway. The removal is not forced, so
// Docker refuses while any container still uses the image, and every failure
// is only logged: the swap already succeeded.
func RemoveLeftoverImage(ctx context.Context, cli ImageAPI, old dcontainer.InspectResponse, imageID, what string) {
	logger := log.With().Str("container", strings.TrimPrefix(old.Name, "/")).Logger()
	if imageID == "" {
		return
	}
	result, err := cli.ImageInspect(ctx, imageID)
	if err != nil {
		logger.Debug().Err(err).Str("image", imageID).Msg("update cleanup: image not inspectable, nothing removed")
		return
	}
	if len(result.RepoTags) > 0 {
		logger.Debug().Str("image", imageID).Strs("tags", result.RepoTags).Msg("update cleanup: image is still tagged, kept")
		return
	}
	if _, err := cli.ImageRemove(ctx, imageID, client.ImageRemoveOptions{}); err != nil {
		logger.Debug().Err(err).Str("image", imageID).Msg("update cleanup: image not removed")
		return
	}
	logger.Info().Str("image", imageID).Msgf("update cleanup: removed %s", what)
}
