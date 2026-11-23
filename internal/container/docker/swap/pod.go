package swap

import (
	"context"
	"errors"
	"fmt"
)

// ErrInPod refuses to update a container that belongs to a Podman pod.
//
// Podman's Docker-compatible create has no way to put a container in a pod,
// so a replacement would come up outside it and `podman pod start`/`stop` and
// podman-compose would stop managing it (#5662). podman-compose puts every
// service in a pod_<project> pod by default. The libpod API that could join
// one takes a whole different spec, so the update is refused before anything
// is touched.
var ErrInPod = errors.New("container is in a Podman pod")

// PodError names the pod a refused container belongs to.
type PodError struct {
	Pod string
}

func (e *PodError) Error() string {
	return fmt.Sprintf("%s (%s) and Podman's Docker API can't recreate it inside one; update it with podman compose pull and podman compose up -d", ErrInPod, e.Pod)
}

func (e *PodError) Unwrap() error { return ErrInPod }

// PodmanPod returns the name of the pod container id is in (its short id if
// the pod can't be read), or "" for none. Only the libpod inspect reports the
// pod, so a client or engine that can't answer it returns "".
func PodmanPod(ctx context.Context, cli any, id string) string {
	var inspect struct {
		Pod string `json:"Pod"`
	}
	if !libpodGet(ctx, cli, "/libpod/containers/"+id+"/json", &inspect) || inspect.Pod == "" {
		return ""
	}
	var pod struct {
		Name string `json:"Name"`
	}
	if libpodGet(ctx, cli, "/libpod/pods/"+inspect.Pod+"/json", &pod) && pod.Name != "" {
		return pod.Name
	}
	return shortID(inspect.Pod)
}
