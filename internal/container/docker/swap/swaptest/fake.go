// Package swaptest is a fake engine for code that swaps containers through
// package swap, so its tests can drive a whole update without a daemon.
package swaptest

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container/docker/swap"
	dcontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
)

type notFoundErr struct{}

func (notFoundErr) Error() string { return "not found" }
func (notFoundErr) NotFound()     {}

// Fake implements swap.API over an in-memory set of containers and records
// every call in Calls, as "verb id [arg]".
type Fake struct {
	mu    sync.Mutex
	Calls []string

	Containers map[string]dcontainer.InspectResponse
	// Created holds every create request, in order.
	Created []client.ContainerCreateOptions
	// NewIDs names created containers in order. Past its end they are new1,
	// new2, ... by create count.
	NewIDs []string
	// NewState is what an inspect of a created container reports. Running
	// and stable when nil.
	NewState *dcontainer.State
	// StartErrFor is a container id whose start fails.
	StartErrFor string
	// CreateErr fails the next create.
	CreateErr error

	// Images are what an image inspect finds, by id.
	Images map[string]image.InspectResponse
	// RemovedImages are the image removals asked for, in order.
	RemovedImages []string
	// ImageRemoveErr fails every image removal.
	ImageRemoveErr error
}

var _ swap.API = (*Fake)(nil)

// New returns a Fake holding the given containers.
func New(containers ...dcontainer.InspectResponse) *Fake {
	f := &Fake{Containers: map[string]dcontainer.InspectResponse{}}
	for _, c := range containers {
		f.Containers[c.ID] = c
	}
	return f
}

// FastTimings shrinks swap's waits for the length of the test.
func FastTimings(t testing.TB) {
	prev := []time.Duration{swap.StableFor, swap.HealthTimeout, swap.GoneTimeout, swap.PollInterval}
	swap.StableFor, swap.HealthTimeout, swap.GoneTimeout, swap.PollInterval = 5*time.Millisecond, 20*time.Millisecond, 20*time.Millisecond, time.Millisecond
	t.Cleanup(func() {
		swap.StableFor, swap.HealthTimeout, swap.GoneTimeout, swap.PollInterval = prev[0], prev[1], prev[2], prev[3]
	})
}

func (f *Fake) record(format string, args ...any) {
	f.Calls = append(f.Calls, fmt.Sprintf(format, args...))
}

func (f *Fake) ContainerInspect(_ context.Context, id string, _ client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.Containers[id]
	if !ok {
		return client.ContainerInspectResult{}, notFoundErr{}
	}
	return client.ContainerInspectResult{Container: c}, nil
}

func (f *Fake) ContainerCreate(_ context.Context, opts client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("create %s", opts.Name)
	f.Created = append(f.Created, opts)
	if f.CreateErr != nil {
		err := f.CreateErr
		f.CreateErr = nil
		return client.ContainerCreateResult{}, err
	}
	n := len(f.Created)
	id := fmt.Sprintf("new%d", n)
	if n <= len(f.NewIDs) {
		id = f.NewIDs[n-1]
	}
	state := f.NewState
	if state == nil {
		state = &dcontainer.State{Running: true, Status: "running", StartedAt: "t0"}
	}
	f.Containers[id] = dcontainer.InspectResponse{ID: id, Name: "/" + opts.Name, Image: opts.Config.Image, Config: opts.Config, HostConfig: opts.HostConfig, State: state}
	return client.ContainerCreateResult{ID: id}, nil
}

func (f *Fake) ContainerStart(_ context.Context, id string, _ client.ContainerStartOptions) (client.ContainerStartResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("start %s", id)
	if id == f.StartErrFor {
		return client.ContainerStartResult{}, errors.New("boom")
	}
	return client.ContainerStartResult{}, nil
}

func (f *Fake) ContainerStop(_ context.Context, id string, _ client.ContainerStopOptions) (client.ContainerStopResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("stop %s", id)
	if c, ok := f.Containers[id]; ok && c.HostConfig != nil && c.HostConfig.AutoRemove {
		delete(f.Containers, id)
	}
	return client.ContainerStopResult{}, nil
}

func (f *Fake) ContainerRename(_ context.Context, id string, opts client.ContainerRenameOptions) (client.ContainerRenameResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("rename %s %s", id, opts.NewName)
	return client.ContainerRenameResult{}, nil
}

func (f *Fake) ContainerRemove(_ context.Context, id string, _ client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("remove %s", id)
	delete(f.Containers, id)
	return client.ContainerRemoveResult{}, nil
}

func (f *Fake) ImageInspect(_ context.Context, id string, _ ...client.ImageInspectOption) (client.ImageInspectResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	img, ok := f.Images[id]
	if !ok {
		return client.ImageInspectResult{}, notFoundErr{}
	}
	return client.ImageInspectResult{InspectResponse: img}, nil
}

func (f *Fake) ImageRemove(_ context.Context, id string, _ client.ImageRemoveOptions) (client.ImageRemoveResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.RemovedImages = append(f.RemovedImages, id)
	return client.ImageRemoveResult{}, f.ImageRemoveErr
}
