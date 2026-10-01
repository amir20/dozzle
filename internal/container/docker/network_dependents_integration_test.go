package docker

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateRejoinsNetworkDependentsAgainstDocker updates a container that
// another one joined with network_mode: service:x, against a real engine and a
// throwaway registry on localhost:5099. It needs alpine and busybox locally:
//
//	DOCKER_IT=1 go test -run TestUpdateRejoinsNetworkDependentsAgainstDocker ./internal/container/docker/
func TestUpdateRejoinsNetworkDependentsAgainstDocker(t *testing.T) {
	if os.Getenv("DOCKER_IT") == "" {
		t.Skip("set DOCKER_IT=1 to run against the local docker daemon")
	}
	const tag = "localhost:5099/dozzle-rejoin-probe:latest"
	t.Cleanup(func() {
		exec.Command("docker", "rm", "-f", "rejoin-registry", "rejoin-parent", "rejoin-child").Run()
		exec.Command("docker", "rmi", tag).Run()
	})
	run(t, "run", "-d", "--name", "rejoin-registry", "-p", "5099:5000", "registry:2")
	push := func(src string) {
		run(t, "tag", src, tag)
		for range 20 {
			if exec.Command("docker", "push", tag).Run() == nil {
				return
			}
			exec.Command("sleep", "0.5").Run()
		}
		t.Fatalf("push %s failed", src)
	}

	push("alpine:latest")
	parentID := run(t, "run", "-d", "--name", "rejoin-parent", tag, "sleep", "1000")
	run(t, "run", "-d", "--name", "rejoin-child", "--network", "container:"+parentID, "alpine:latest", "sleep", "1000")
	push("busybox:latest")
	run(t, "tag", "alpine:latest", tag) // the pull, not the local tag, must bring the new image

	cli, err := NewLocalClient("", container.NewHostIDResolver(""))
	require.NoError(t, err)
	svc := &Service{client: cli}
	c, err := cli.FindContainer(context.Background(), parentID)
	require.NoError(t, err)

	progress := make(chan container.UpdateProgress, 1000)
	updated, err := svc.UpdateContainer(context.Background(), c, progress)
	require.NoError(t, err)
	assert.True(t, updated)

	newParent := run(t, "inspect", "-f", "{{.Id}}", "rejoin-parent")
	assert.NotEqual(t, parentID, newParent)
	assert.Equal(t, "container:"+newParent, run(t, "inspect", "-f", "{{.HostConfig.NetworkMode}}", "rejoin-child"))
	assert.Equal(t, "true", run(t, "inspect", "-f", "{{.State.Running}}", "rejoin-child"))
	assert.Equal(t,
		run(t, "exec", "rejoin-parent", "hostname", "-i"),
		run(t, "exec", "rejoin-child", "hostname", "-i"),
		"the child sees the new parent's interfaces")
}

func run(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("docker", args...).CombinedOutput()
	require.NoError(t, err, "docker %v: %s", args, out)
	return strings.TrimSpace(string(out))
}
