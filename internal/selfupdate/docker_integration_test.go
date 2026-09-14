package selfupdate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunAgainstDocker drives Run against a real engine with throwaway
// containers, for plain and --rm containers, on success and on rollback. It
// needs alpine and busybox pulled locally:
//
//	SELFUPDATE_DOCKER_IT=1 go test -run TestRunAgainstDocker ./internal/selfupdate/
func TestRunAgainstDocker(t *testing.T) {
	if os.Getenv("SELFUPDATE_DOCKER_IT") == "" {
		t.Skip("set SELFUPDATE_DOCKER_IT=1 to run against the local docker daemon")
	}
	prev := stableFor
	stableFor = 2 * time.Second
	t.Cleanup(func() { stableFor = prev })

	for _, rm := range []bool{false, true} {
		for _, fail := range []bool{false, true} {
			name := fmt.Sprintf("selfupdate-probe-rm-%v-fail-%v", rm, fail)
			t.Run(name, func(t *testing.T) { runProbe(t, name, rm, fail) })
		}
	}
}

func runProbe(t *testing.T, name string, rm, fail bool) {
	tag := name + ":latest"
	dockerCmd(t, "network", "create", name)
	dockerCmd(t, "tag", "alpine:latest", tag)
	vol := ""
	t.Cleanup(func() {
		for id := range strings.FieldsSeq(dockerOut("ps", "-aq", "--filter", "name="+name)) {
			exec.Command("docker", "rm", "-f", id).Run()
		}
		exec.Command("docker", "network", "rm", name).Run()
		exec.Command("docker", "rmi", tag).Run()
		if vol != "" {
			exec.Command("docker", "volume", "rm", vol).Run()
		}
	})

	script := "[ -f /data/f ] || echo hello > /data/f; sleep 1000"
	if fail {
		// busybox has no Alpine os-release, so the replacement exits at once.
		script = "grep -q Alpine /etc/os-release || exit 3; " + script
	}
	args := []string{"run", "-d", "--name", name, "--network", name, "--network-alias", "probe",
		"-v", "/data", "--label", "keep=me"}
	if rm {
		args = append(args, "--rm")
	}
	args = append(args, tag, "sh", "-c", script)
	oldID := dockerCmd(t, args...)
	vol = dockerCmd(t, "inspect", "-f", "{{range .Mounts}}{{.Name}}{{end}}", oldID)
	time.Sleep(time.Second)

	dockerCmd(t, "tag", "busybox:latest", tag)
	alpineID := dockerCmd(t, "image", "inspect", "-f", "{{.Id}}", "alpine:latest")
	busyboxID := dockerCmd(t, "image", "inspect", "-f", "{{.Id}}", "busybox:latest")

	err := Run(context.Background(), oldID)
	if fail {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
	}

	currentID := dockerCmd(t, "inspect", "-f", "{{.Id}}", name)
	assert.Equal(t, "true", dockerCmd(t, "inspect", "-f", "{{.State.Running}}", currentID))
	assert.Equal(t, vol, dockerCmd(t, "inspect", "-f", "{{range .Mounts}}{{.Name}}{{end}}", currentID), "anonymous volume reused")
	assert.Equal(t, "hello", dockerCmd(t, "exec", currentID, "cat", "/data/f"))
	assert.Contains(t, dockerCmd(t, "inspect", "-f", "{{json .NetworkSettings.Networks}}", currentID), `"probe"`)
	assert.Equal(t, "me", dockerCmd(t, "inspect", "-f", `{{index .Config.Labels "keep"}}`, currentID))
	assert.Equal(t, rm, dockerCmd(t, "inspect", "-f", "{{.HostConfig.AutoRemove}}", currentID) == "true")
	assert.Len(t, strings.Fields(dockerOut("ps", "-aq", "--filter", "name="+name)), 1, "exactly one container left")

	image := dockerCmd(t, "inspect", "-f", "{{.Image}}", currentID)
	if fail {
		assert.Equal(t, alpineID, image, "rolled back to the old image")
		if !rm {
			assert.Equal(t, oldID, currentID, "the original container is restored")
		}
	} else {
		assert.Equal(t, busyboxID, image)
		assert.NotEqual(t, oldID, currentID)
	}
}

func dockerCmd(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("docker", args...).CombinedOutput()
	require.NoError(t, err, "docker %v: %s", args, out)
	return strings.TrimSpace(string(out))
}

func dockerOut(args ...string) string {
	out, _ := exec.Command("docker", args...).Output()
	return strings.TrimSpace(string(out))
}
