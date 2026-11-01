package cloud

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"

	"github.com/amir20/dozzle/internal/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

// Config pusher: sends each container's configuration to Dozzle Cloud so it
// can keep a history of what changed (limits, ports, mounts, restart policy,
// image), the part of "what changed before it broke?" that logs and events
// don't record.
//
// It sends every running container once when the connection opens, then each
// container as it starts. Docker only applies a config change by creating a
// new container, which starts, so a start is where a change shows up. An
// in-place `docker update` (limits, restart policy) is not seen until the next
// start. A container whose config is the same as the last one sent for its
// name is not sent again.
//
// Nothing that can hold a secret leaves Dozzle: environment variables are
// never read here, and label values and the command go only as hashes.

// configChanBuf absorbs a compose up starting many containers at once while
// the pusher is busy sending one.
const configChanBuf = 64

// ConfigStreamHostService is what the config pusher needs from the host
// service. MultiHostService and K8sClusterService both satisfy it.
type ConfigStreamHostService interface {
	ToolHostService
	SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container, filter container.ContainerFilter)
}

func pushConfigs(ctx context.Context, hs ConfigStreamHostService, labels container.ContainerLabels, send func(*pb.ToolResponse) error) {
	// Subscribed before the first list, so a container that starts while it
	// runs is not missed. One that arrives both ways is sent once.
	started := make(chan container.Container, configChanBuf)
	hs.SubscribeContainersStarted(ctx, started, func(_ *container.Container) bool { return true })

	lastSent := map[string][]byte{} // host/name -> the last config sent, marshalled
	pushOne := func(c container.Container) {
		// A container the store hasn't inspected yet has no config to send;
		// its start brings it inspected.
		if !c.FullyLoaded || c.State != "running" {
			return
		}
		cfg := containerConfigProto(c)
		key := c.Host + "/" + c.Name
		sig := configSignature(cfg)
		if prev, ok := lastSent[key]; ok && slices.Equal(prev, sig) {
			return
		}
		if err := send(&pb.ToolResponse{Type: &pb.ToolResponse_ContainerConfig{ContainerConfig: cfg}}); err != nil {
			// The connection is going; the next one sends every container again.
			log.Debug().Err(err).Str("container", c.Name).Msg("config pusher: send failed")
			return
		}
		lastSent[key] = sig
	}

	running, _ := hs.ListAllContainers(labels)
	for _, c := range running {
		if ctx.Err() != nil {
			return
		}
		pushOne(c)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case c, ok := <-started:
			if !ok {
				return
			}
			// Only what the cloud user may see: FindContainer applies the
			// principal's labels, as every tool does.
			if labels.Exists() {
				if _, err := hs.FindContainer(c.Host, c.ID, labels); err != nil {
					continue
				}
			}
			pushOne(c)
		}
	}
}

// containerConfigProto is the config Cloud may keep for c: no env, and label
// values and the command hashed.
func containerConfigProto(c container.Container) *pb.ContainerConfig {
	labelHashes := make(map[string]string, len(c.Labels))
	for k, v := range c.Labels {
		labelHashes[k] = hashValue(v)
	}
	ports := slices.Clone(c.Ports)
	slices.Sort(ports)
	mounts := mountStrings(c.Mounts)
	slices.Sort(mounts)
	return &pb.ContainerConfig{
		Host:          c.Host,
		Name:          c.Name,
		ContainerId:   c.ID,
		Image:         c.Image,
		CommandHash:   hashValue(c.Command),
		LabelHashes:   labelHashes,
		MemoryLimit:   c.MemoryLimit,
		CpuLimit:      c.CPULimit,
		Ports:         ports,
		Mounts:        mounts,
		RestartPolicy: c.RestartPolicy,
		NetworkMode:   c.NetworkMode,
		StartedAt:     unixNano(c.StartedAt),
	}
}

// configSignature is what decides whether a config changed: everything but the
// container id and start time, which every restart changes.
func configSignature(cfg *pb.ContainerConfig) []byte {
	c := proto.CloneOf(cfg)
	c.ContainerId = ""
	c.StartedAt = 0
	b, _ := proto.MarshalOptions{Deterministic: true}.Marshal(c)
	return b
}

// hashValue is a value's sha256 in hex, or "" for an empty one.
func hashValue(v string) string {
	if v == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}
