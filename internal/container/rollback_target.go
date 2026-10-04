package container

import (
	"fmt"
	"slices"
	"strings"
)

// RollbackTarget is the image a container ran before its last update, where a
// rollback takes it.
type RollbackTarget struct {
	// ImageID is the local image id.
	ImageID string `json:"imageId"`
	// Ref is the image as repo@sha256:digest, to pull it again by digest once
	// it is gone locally. Empty for an image built locally.
	Ref string `json:"ref,omitempty"`
}

// RollbackTargetOf is where a rollback takes the container with this id,
// running imageID with these labels, given its host's update events.
//
// The events know the previous image for any update the host saw, whoever
// made it, so when one created this container it is the answer, and the
// labels are not read at all. Watchtower copies every label onto the container
// it recreates, so after a Watchtower update the previous-image label may name
// an older image than the one just replaced, and the update-source label may
// say rollback for a container that was since updated. The labels only cover
// an update made before the host's events begin, such as one before Dozzle
// started.
//
// A container a rollback created has no target of its own: its previous image
// is the newer one it rolled back from.
func RollbackTargetOf(id, imageID string, labels map[string]string, events []ContainerUpdateEvent) (RollbackTarget, error) {
	if labels["com.docker.swarm.service.name"] != "" {
		return RollbackTarget{}, fmt.Errorf("%w for swarm services", ErrRollbackUnsupported)
	}
	if len(id) > 12 {
		id = id[:12]
	}
	for _, e := range slices.Backward(events) {
		// A rolled back swap's NewID is the old container put back: the
		// event says nothing about how that container came to be.
		if e.RolledBack || e.NewID != id {
			continue
		}
		if e.Source == UpdateSourceRollback {
			return RollbackTarget{}, errAlreadyRolledBack
		}
		if e.FromImageID == "" || SameImageID(e.FromImageID, imageID) {
			return RollbackTarget{}, ErrNoRollbackTarget
		}
		return RollbackTarget{ImageID: e.FromImageID, Ref: e.FromDigest}, nil
	}

	if labels[UpdateSourceLabel] == UpdateSourceRollback {
		return RollbackTarget{}, errAlreadyRolledBack
	}
	if previous := strings.TrimSpace(labels[PreviousImageLabel]); previous != "" && !SameImageID(previous, imageID) {
		return RollbackTarget{ImageID: previous, Ref: strings.TrimSpace(labels[PreviousRefLabel])}, nil
	}
	return RollbackTarget{}, ErrNoRollbackTarget
}

var errAlreadyRolledBack = fmt.Errorf("%w: the container was already rolled back", ErrNoRollbackTarget)

// SameImageID compares image ids with or without their sha256: prefix.
func SameImageID(a, b string) bool {
	a, b = strings.TrimPrefix(a, "sha256:"), strings.TrimPrefix(b, "sha256:")
	return a != "" && a == b
}
