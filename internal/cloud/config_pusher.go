package cloud

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

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
// name is not sent again, and neither is one announced again for a start
// already sent. The second matters when two running containers share a name on
// one host (an old Swarm task next to its replacement): their configs
// alternate, and against the name alone each one would be sent on every
// announcement. A real start of the same container, such as a failed update
// reverted to the old one, has a new start time and is compared with the name
// as usual.
//
// Nothing that can hold a secret leaves Dozzle: environment variables are
// never read here, and label values and the command go only as HMACs keyed by
// a secret that stays on this machine, so a short password can't be recovered
// by hashing guesses.

// startsPerName bounds how many sent starts are remembered for one name, so a
// container recreated under a fixed name over and over (a cron job, a CI
// runner) can't grow the pusher's memory. It covers up to this many running
// containers sharing a name (an old Swarm task and its replacement, replicas
// with one dev.dozzle.name); a bigger group is resent on each announcement,
// as before.
const startsPerName = 16

// sentStart is one container start whose config was sent.
type sentStart struct {
	id        string
	startedAt time.Time
	sig       []byte
}

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

	lastSent := map[string][]byte{}        // host/name -> the last config sent, marshalled
	sentStarts := map[string][]sentStart{} // host/name -> its latest starts sent, oldest first
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
		if slices.ContainsFunc(sentStarts[key], func(s sentStart) bool {
			return s.id == c.ID && s.startedAt.Equal(c.StartedAt) && slices.Equal(s.sig, sig)
		}) {
			return
		}
		if err := send(&pb.ToolResponse{Type: &pb.ToolResponse_ContainerConfig{ContainerConfig: cfg}}); err != nil {
			// The connection is going; the next one sends every container again.
			log.Debug().Err(err).Str("container", c.Name).Msg("config pusher: send failed")
			return
		}
		lastSent[key] = sig
		starts := append(sentStarts[key], sentStart{id: c.ID, startedAt: c.StartedAt, sig: sig})
		if len(starts) > startsPerName {
			starts = starts[len(starts)-startsPerName:]
		}
		sentStarts[key] = starts
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

// configHashSecretPath sits beside the other data files and never leaves this
// machine. It has to be stored: a key that changed on restart would make every
// label and command look changed.
const configHashSecretPath = "./data/.config-hash-secret"

var (
	configHashOnce   sync.Once
	configHashSecret []byte
)

func loadConfigHashSecret() []byte {
	configHashOnce.Do(func() {
		if existing, err := os.ReadFile(configHashSecretPath); err == nil {
			if decoded, err := hex.DecodeString(string(existing)); err == nil && len(decoded) == 32 {
				configHashSecret = decoded
				return
			}
			log.Warn().Msg("config pusher: hash secret is unreadable, generating a new one; every container will read as changed once")
		}
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			log.Error().Err(err).Msg("config pusher: could not generate a hash secret")
			return
		}
		if err := os.MkdirAll(filepath.Dir(configHashSecretPath), 0o700); err != nil {
			log.Warn().Err(err).Msg("config pusher: could not create the data directory for the hash secret")
		} else if err := os.WriteFile(configHashSecretPath, []byte(hex.EncodeToString(secret)), 0o600); err != nil {
			log.Warn().Err(err).Msg("config pusher: could not persist the hash secret; labels will read as changed after a restart")
		}
		configHashSecret = secret
	})
	return configHashSecret
}

// hashValue is a value's HMAC-SHA256 under this machine's secret, in hex, or
// "" for an empty value or when there is no secret.
func hashValue(v string) string {
	secret := loadConfigHashSecret()
	if v == "" || len(secret) == 0 {
		return ""
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(v))
	return hex.EncodeToString(mac.Sum(nil))
}
