package cloud

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func configContainer(id string, mem uint64) container.Container {
	return container.Container{
		ID: id, Name: "sonarr", Host: "host-1", Image: "linuxserver/sonarr:4.0.10", State: "running", FullyLoaded: true,
		Command:     "/init --password=hunter2",
		Labels:      map[string]string{"traefik.http.middlewares.auth.basicauth.users": "admin:$apr1$secret"},
		MemoryLimit: mem, Ports: []string{"8989/tcp", "80/tcp"}, RestartPolicy: "unless-stopped",
		StartedAt: time.Unix(1700000000, 0),
	}
}

func TestPushConfigs(t *testing.T) {
	// A fixed key instead of the file the first push would create in the
	// package directory.
	configHashOnce.Do(func() { configHashSecret = bytes.Repeat([]byte{7}, 32) })

	hs := &fakeHostService{
		containers: []container.Container{
			configContainer("a1", 2<<30),
			{ID: "b1", Name: "half", Host: "host-1", State: "running"}, // not inspected yet: nothing to send
			{ID: "c1", Name: "gone", Host: "host-1", State: "exited", FullyLoaded: true},
		},
		clients: map[string]*fakeClientService{"host-1": newFakeClientService("host-1")},
	}
	var mu sync.Mutex
	var sent []*pb.ContainerConfig
	send := func(r *pb.ToolResponse) error {
		mu.Lock()
		defer mu.Unlock()
		sent = append(sent, r.GetContainerConfig())
		return nil
	}
	count := func() int {
		mu.Lock()
		defer mu.Unlock()
		return len(sent)
	}

	ctx := t.Context()
	go pushConfigs(ctx, hs, nil, send)

	require.Eventually(t, func() bool { return count() == 1 }, time.Second, 5*time.Millisecond, "the one running, inspected container on connect")
	require.Eventually(t, func() bool {
		hs.mu.Lock()
		defer hs.mu.Unlock()
		return hs.startedCh != nil
	}, time.Second, 5*time.Millisecond)

	// A restart: new id and start time, same config. Not sent again.
	restarted := configContainer("a2", 2<<30)
	restarted.StartedAt = time.Unix(1700000600, 0)
	hs.emitStart(restarted)
	// A redeploy with a lower memory limit. Sent.
	hs.emitStart(configContainer("a3", 1<<30))
	require.Eventually(t, func() bool { return count() == 2 }, time.Second, 5*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 2, count(), "the restart with the same config is not sent")

	mu.Lock()
	defer mu.Unlock()
	first, changed := sent[0], sent[1]
	assert.Equal(t, uint64(2<<30), first.MemoryLimit)
	assert.Equal(t, uint64(1<<30), changed.MemoryLimit)
	assert.Equal(t, "a3", changed.ContainerId)
	assert.Equal(t, []string{"80/tcp", "8989/tcp"}, first.Ports, "sorted, so order never reads as a change")

	// No secret leaves Dozzle: label values and the command only as hashes.
	raw := first.String()
	assert.NotContains(t, raw, "hunter2")
	assert.NotContains(t, raw, "$apr1$secret")
	hash := first.LabelHashes["traefik.http.middlewares.auth.basicauth.users"]
	assert.Len(t, hash, 64)
	assert.Len(t, first.CommandHash, 64)
	assert.False(t, strings.Contains(raw, "admin:"))
	plain := sha256.Sum256([]byte("admin:$apr1$secret"))
	assert.NotEqual(t, hex.EncodeToString(plain[:]), hash, "keyed, so a guess can't be checked by hashing it")
}
