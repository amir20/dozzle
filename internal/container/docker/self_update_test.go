package docker

import (
	"context"
	"testing"

	"github.com/amir20/dozzle/internal/container"

	docker_types "github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSelf(t *testing.T) {
	const full = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	prev := selfContainerID
	t.Cleanup(func() { selfContainerID = prev })

	selfContainerID = func() string { return full }
	assert.True(t, isSelf(full[:12]), "Dozzle's short id")
	assert.True(t, isSelf(full))
	assert.False(t, isSelf("abc"), "too short to be an id")
	assert.False(t, isSelf("0123456789ab"))

	selfContainerID = func() string { return "" }
	assert.False(t, isSelf(full[:12]), "not in a container")
}

func TestMayBeSelf(t *testing.T) {
	const full = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	prevID, prevHost := selfContainerID, hostname
	t.Cleanup(func() { selfContainerID, hostname = prevID, prevHost })
	hostname = func() (string, error) { return "somehost", nil }

	inspect := func(id, image string, labels map[string]string) docker_types.InspectResponse {
		return docker_types.InspectResponse{ID: id, Config: &docker_types.Config{Image: image, Labels: labels}}
	}

	selfContainerID = func() string { return "" }
	assert.True(t, mayBeSelf(inspect("0123456789ab", "amir20/dozzle:latest", nil)), "Dozzle image with no known id")
	assert.True(t, mayBeSelf(inspect("0123456789ab", "sha256:"+full, map[string]string{"dev.dozzle.self-update.image": "amir20/dozzle:latest"})), "rolled back Dozzle")
	assert.False(t, mayBeSelf(inspect("0123456789ab", "nginx", nil)))
	hostname = func() (string, error) { return full[:12], nil }
	assert.True(t, mayBeSelf(inspect(full, "my-registry/logs:latest", nil)), "renamed image, hostname is its short id")

	selfContainerID = func() string { return full }
	assert.False(t, mayBeSelf(inspect("0123456789ab", "amir20/dozzle:latest", nil)), "a known id decides through isSelf")
}

// rejoinClient records the calls rejoinDependents makes. The embedded nil
// interface panics on anything else.
type rejoinClient struct {
	UpdateClient
	containers map[string]docker_types.InspectResponse
	calls      []string
}

func (r *rejoinClient) ContainerInspect(_ context.Context, id string) (docker_types.InspectResponse, error) {
	return r.containers[id], nil
}

func (r *rejoinClient) ContainerActions(_ context.Context, action container.ContainerAction, id string) error {
	r.calls = append(r.calls, string(action)+" "+id)
	return nil
}

func (r *rejoinClient) ContainerRemove(_ context.Context, id string) error {
	r.calls = append(r.calls, "remove "+id)
	return nil
}

func (r *rejoinClient) ContainerCreate(_ context.Context, _ docker_types.InspectResponse, name string) (string, error) {
	r.calls = append(r.calls, "create "+name)
	return "new-" + name, nil
}

// The helper stops Dozzle within seconds of starting, so it must not start
// while another dependent is between its remove and its create.
func TestRejoinDependentsSelfGoesLast(t *testing.T) {
	const (
		oldID = "1111111111110000000000000000000000000000000000000000000000000000"
		self  = "aaaaaaaaaaaa0000000000000000000000000000000000000000000000000000"
		app   = "bbbbbbbbbbbb0000000000000000000000000000000000000000000000000000"
	)
	prevID, prevRejoin := selfContainerID, startRejoin
	t.Cleanup(func() { selfContainerID, startRejoin = prevID, prevRejoin })
	selfContainerID = func() string { return self }

	joined := &docker_types.HostConfig{NetworkMode: "container:" + oldID}
	running := &docker_types.State{Running: true}
	cli := &rejoinClient{containers: map[string]docker_types.InspectResponse{
		self: {ID: self, Name: "/dozzle", State: running, HostConfig: joined, Config: &docker_types.Config{}},
		app:  {ID: app, Name: "/app", State: running, HostConfig: joined, Config: &docker_types.Config{}},
	}}
	startRejoin = func(_ context.Context, id, mode string) error {
		cli.calls = append(cli.calls, "rejoin "+id[:12]+" "+mode)
		return nil
	}

	svc := &Service{client: cli}
	require.NoError(t, svc.rejoinDependents(context.Background(), []string{self, app}, oldID, "new-sidecar"))
	assert.Equal(t, []string{
		"stop " + app,
		"remove " + app,
		"create app",
		"start new-app",
		"rejoin aaaaaaaaaaaa container:new-sidecar",
	}, cli.calls)
}

// A dependent that was stopped is recreated against the replacement but not
// started: it was stopped before the update, and stays that way.
func TestRejoinDependentsLeavesStoppedDependentStopped(t *testing.T) {
	const (
		oldID = "1111111111110000000000000000000000000000000000000000000000000000"
		app   = "bbbbbbbbbbbb0000000000000000000000000000000000000000000000000000"
	)
	joined := &docker_types.HostConfig{NetworkMode: "container:" + oldID}
	cli := &rejoinClient{containers: map[string]docker_types.InspectResponse{
		app: {ID: app, Name: "/app", State: &docker_types.State{Status: "exited"}, HostConfig: joined, Config: &docker_types.Config{}},
	}}

	svc := &Service{client: cli}
	require.NoError(t, svc.rejoinDependents(context.Background(), []string{app}, oldID, "new-vpn"))
	assert.Equal(t, []string{
		"remove " + app,
		"create app",
	}, cli.calls)
}
