package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateKeepsStaticIPAndMACAgainstDocker updates a container pinned to an
// IP and MAC on a macvlan network (#5652), against a real engine and a
// throwaway registry on localhost:5098. It needs alpine and busybox locally:
//
//	DOCKER_IT=1 go test -run TestUpdateKeepsStaticIPAndMACAgainstDocker ./internal/container/docker/
func TestUpdateKeepsStaticIPAndMACAgainstDocker(t *testing.T) {
	if os.Getenv("DOCKER_IT") == "" {
		t.Skip("set DOCKER_IT=1 to run against the local docker daemon")
	}
	const (
		tag  = "localhost:5098/dozzle-static-mac-probe:latest"
		net  = "dozzle-static-mac-net"
		ip   = "192.168.213.252"
		mac  = "2e:3a:f4:21:c4:f3"
		name = "static-mac-probe"
	)
	t.Cleanup(func() {
		exec.Command("docker", "rm", "-f", "static-mac-registry", name).Run()
		exec.Command("docker", "network", "rm", net).Run()
		exec.Command("docker", "rmi", tag).Run()
	})
	run(t, "run", "-d", "--name", "static-mac-registry", "-p", "5098:5000", "registry:2")
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

	run(t, "network", "create", "-d", "macvlan", "--subnet", "192.168.213.0/24", net)
	push("alpine:latest")
	oldID := run(t, "run", "-d", "--name", name, "--network", net, "--ip", ip, "--mac-address", mac, tag, "sleep", "1000")
	push("busybox:latest")
	run(t, "tag", "alpine:latest", tag) // the pull, not the local tag, must bring the new image

	cli, err := NewLocalClient("", container.NewHostIDResolver(""))
	require.NoError(t, err)
	svc := &Service{client: cli}
	c, err := cli.FindContainer(context.Background(), oldID)
	require.NoError(t, err)

	progress := make(chan container.UpdateProgress, 1000)
	updated, err := svc.UpdateContainer(context.Background(), c, progress)
	require.NoError(t, err)
	assert.True(t, updated)

	assert.NotEqual(t, oldID, run(t, "inspect", "-f", "{{.Id}}", name))
	format := "{{(index .NetworkSettings.Networks \"" + net + "\").%s}}"
	assert.Equal(t, ip, run(t, "inspect", "-f", fmt.Sprintf(format, "IPAddress"), name))
	assert.Equal(t, mac, run(t, "inspect", "-f", fmt.Sprintf(format, "MacAddress"), name))
}
