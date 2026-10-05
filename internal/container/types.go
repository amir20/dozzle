package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/amir20/dozzle/internal/utils"
)

// Container represents an internal representation of docker containers
type Container struct {
	ID            string                           `json:"id"`
	Name          string                           `json:"name"`
	Image         string                           `json:"image"`
	Command       string                           `json:"command"`
	Created       time.Time                        `json:"created"`
	StartedAt     time.Time                        `json:"startedAt"`
	FinishedAt    time.Time                        `json:"finishedAt"`
	State         string                           `json:"state"`
	Health        string                           `json:"health,omitempty"`
	Host          string                           `json:"host,omitempty"`
	Tty           bool                             `json:"-"`
	Labels        map[string]string                `json:"labels,omitempty"`
	Stats         *utils.RingBuffer[ContainerStat] `json:"stats,omitempty"`
	MemoryLimit   uint64                           `json:"memoryLimit"`
	CPULimit      float64                          `json:"cpuLimit"`
	Group         string                           `json:"group,omitempty"`
	Env           []string                         `json:"-"`
	Ports         []string                         `json:"ports,omitempty"`
	Mounts        []Mount                          `json:"mounts,omitempty"`
	MountStats    map[string]MountStat             `json:"mountStats,omitempty"`
	RestartPolicy string                           `json:"-"`
	NetworkMode   string                           `json:"-"`
	FullyLoaded   bool                             `json:"-"`
	// RestartCount is how many times the engine restarted the container under
	// its restart policy (Docker) or the kubelet restarted it (k8s). Only an
	// inspect knows it, so a list entry has 0.
	RestartCount int `json:"-"`
	// OOMKilled reports whether the last run ended in an out-of-memory kill.
	OOMKilled bool `json:"-"`
	// ExitCode is the last run's exit code; meaningful only once it has exited.
	ExitCode int `json:"-"`
	// ImageDigest is the image the container runs, as "repo@sha256:...". Only
	// k8s fills it, from the pod status; Docker looks up RepoDigests per check.
	ImageDigest string `json:"-"`
	// SizeRw is how many bytes the container's writable layer holds. Measuring it
	// walks the layer, so the size monitor fills it in on its own schedule; nil
	// until the first measurement.
	SizeRw *int64 `json:"sizeRw,omitempty"`
	// Volumes are the Docker-managed volumes the container mounts, with their size.
	// Docker measures every volume on the host in one walk, so these refresh far less
	// often than SizeRw. Bind mounts are never here: Docker cannot measure them.
	Volumes []VolumeUsage `json:"volumes,omitempty"`
}

// VolumeUsage is one volume a container mounts.
type VolumeUsage struct {
	Name        string `json:"name"`
	Destination string `json:"destination"`
	Size        int64  `json:"size"`
	// Links is how many containers use the volume, this one included, so a volume
	// with Links > 1 is counted in each of their totals.
	Links int64 `json:"links"`
}

// Mount represents a container mount point
type Mount struct {
	Type        string `json:"type"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	RW          bool   `json:"rw"`
}

// String returns a display form for legacy consumers.
func (m Mount) String() string {
	return fmt.Sprintf("%s:%s (%s)", m.Source, m.Destination, m.Type)
}

// MountStat carries free-space information for a single mount point.
type MountStat struct {
	Destination string    `json:"destination"`
	Total       uint64    `json:"total"`
	Free        uint64    `json:"free"`
	Used        uint64    `json:"used"`
	Available   bool      `json:"available"`
	LastChecked time.Time `json:"lastChecked"`
}

// ContainerStat represent stats instant for a container
type ContainerStat struct {
	ID             string  `json:"id,omitempty"`
	CPUPercent     float64 `json:"cpu"`
	MemoryPercent  float64 `json:"memory"`
	MemoryUsage    float64 `json:"memoryUsage"`
	NetworkRxTotal uint64  `json:"networkRxTotal"`
	NetworkTxTotal uint64  `json:"networkTxTotal"`
	DiskReadTotal  uint64  `json:"diskReadTotal"`
	DiskWriteTotal uint64  `json:"diskWriteTotal"`
}

// MarshalJSON writes a stat without reflection. The first containers-changed event of a
// stream carries up to 300 points for every container, so what a single point costs is
// worth caring about: the id is empty on every buffered point (the store clears it before
// pushing) and is dropped here, and percentages are written at two decimals, which is
// finer than anything the UI renders.
func (s ContainerStat) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, 160)
	b = append(b, '{')
	if s.ID != "" {
		id, err := json.Marshal(s.ID)
		if err != nil {
			return nil, err
		}
		b = append(b, `"id":`...)
		b = append(b, id...)
		b = append(b, ',')
	}
	b = append(b, `"cpu":`...)
	b = appendFloat(b, s.CPUPercent, 2)
	b = append(b, `,"memory":`...)
	b = appendFloat(b, s.MemoryPercent, 2)
	b = append(b, `,"memoryUsage":`...)
	b = appendFloat(b, s.MemoryUsage, 0)
	b = append(b, `,"networkRxTotal":`...)
	b = strconv.AppendUint(b, s.NetworkRxTotal, 10)
	b = append(b, `,"networkTxTotal":`...)
	b = strconv.AppendUint(b, s.NetworkTxTotal, 10)
	b = append(b, `,"diskReadTotal":`...)
	b = strconv.AppendUint(b, s.DiskReadTotal, 10)
	b = append(b, `,"diskWriteTotal":`...)
	b = strconv.AppendUint(b, s.DiskWriteTotal, 10)
	b = append(b, '}')
	return b, nil
}

// appendFloat rounds to decimals and then writes the shortest form that round-trips, so a
// rounded value never comes back as 0.29000000000000004 and a whole one stays "0". NaN and
// Inf become 0, which the reflection encoder would have failed the whole event over.
func appendFloat(b []byte, v float64, decimals int) []byte {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return append(b, '0')
	}
	if decimals > 0 {
		pow := math.Pow(10, float64(decimals))
		v = math.Round(v*pow) / pow
	} else {
		v = math.Round(v)
	}
	return strconv.AppendFloat(b, v, 'f', -1, 64)
}

// ContainerEvent represents events that are triggered
type ContainerEvent struct {
	Name            string            `json:"name"`
	Host            string            `json:"host"`
	ActorID         string            `json:"actorId"`
	ActorAttributes map[string]string `json:"actorAttributes,omitempty"`
	Time            time.Time         `json:"time"`
	Container       *Container        `json:"-"`
}

type ContainerLabels map[string][]string

func ParseContainerFilter(commaValues string) (ContainerLabels, error) {
	filter := make(ContainerLabels)
	if commaValues == "" {
		return filter, nil
	}

	for val := range strings.SplitSeq(commaValues, ",") {
		before, after, ok := strings.Cut(val, "=")
		if !ok {
			return nil, fmt.Errorf("invalid filter: %s", filter)
		}
		key := before
		val := after
		filter[key] = append(filter[key], val)
	}

	return filter, nil
}

func (f ContainerLabels) Exists() bool {
	return len(f) > 0
}

type LogPosition string

const (
	Beginning LogPosition = "start"
	Middle    LogPosition = "middle"
	End       LogPosition = "end"
)

type LogType string

const (
	LogTypeSingle  LogType = "single"  // Single simple text log (no grouping)
	LogTypeGroup   LogType = "group"   // Grouped simple logs (array of fragments)
	LogTypeComplex LogType = "complex" // JSON or logfmt parsed log
)

// LogFragment represents a single line within a grouped simple log
type LogFragment struct {
	Message string `json:"m"`
	// TimestampPrefix is the length of a leading timestamp that repeats the
	// line's Docker timestamp. See LogEvent.TimestampPrefix.
	TimestampPrefix int `json:"tp,omitempty"`
}

type ContainerAction string

const (
	Start   ContainerAction = "start"
	Stop    ContainerAction = "stop"
	Restart ContainerAction = "restart"
	Remove  ContainerAction = "remove"
)

func ParseContainerAction(input string) (ContainerAction, error) {
	action := ContainerAction(input)
	switch action {
	case Start, Stop, Restart, Remove:
		return action, nil
	default:
		return "", fmt.Errorf("unknown action: %s", input)
	}
}

// Statuses an update reports, in the order they can arrive. Every update ends
// on exactly one of done, up-to-date, rolled-back or error.
const (
	UpdatePulling    = "pulling"
	UpdateRecreating = "recreating"
	// UpdateVerifying means the replacement started and is being watched to
	// stay up (and healthy, if it has a healthcheck) before the old one goes.
	UpdateVerifying = "verifying"
	UpdateDone      = "done"
	UpdateUpToDate  = "up-to-date"
	// UpdateRolledBack means the replacement failed and the previous
	// container was put back. Error says why the replacement failed.
	UpdateRolledBack = "rolled-back"
	UpdateError      = "error"
)

type UpdateProgress struct {
	Status  string `json:"status"`  // one of the Update* statuses above
	Layer   string `json:"layer"`   // Docker layer ID (pull events only)
	Current int64  `json:"current"` // Bytes downloaded
	Total   int64  `json:"total"`   // Total bytes for layer
	Error   string `json:"error"`   // Only when Status is "error" or "rolled-back"
	// Result is set on the last progress of a swap that ran, whether it
	// committed ("done") or was undone ("rolled-back"). It is what the update
	// record needs, and never reaches the browser.
	Result *UpdateResult `json:"-"`
}

// UpdateResult is what one swap changed, as the host that ran it saw it. An
// agent sends it back with its last progress, so the server records updates on
// agent hosts the same way as its own.
type UpdateResult struct {
	// OldID is the container that ran before, NewID the one running now: the
	// replacement, or the old one put back when RolledBack.
	OldID string
	NewID string
	// FromImageID and ToImageID are local image ids, FromDigest and ToDigest
	// the same images as repo@sha256:... (empty for an image built locally).
	FromImageID  string
	ToImageID    string
	FromDigest   string
	ToDigest     string
	OldStartedAt time.Time
	// RolledBack means the new container did not stay up and the old one was
	// put back.
	RolledBack bool
}

// Labels an update leaves on the container it creates, and reads back on the
// next one.
const (
	// PreviousImageLabel is the image id the container ran before its last
	// update: the rollback target, and what cleanup removes on the update
	// after.
	PreviousImageLabel = "dev.dozzle.previous-image"
	// PreviousRefLabel is that image as repo@sha256:digest, so it can be
	// pulled again by digest once it is gone locally. Absent for an image
	// built locally, which has no registry digest.
	PreviousRefLabel = "dev.dozzle.previous-ref"
	// RolledBackFromLabel is the image a rollback moved the container away
	// from, as repo@sha256:digest (or the image id for an image built
	// locally). The auto-update schedule leaves the container alone while its
	// registry still offers that image, and the next update clears it.
	RolledBackFromLabel = "dev.dozzle.rolled-back-from"
)

// RollbackOptions are what one rollback checks before it runs.
type RollbackOptions struct {
	// ExpectedFromDigest refuses the rollback unless the container still runs
	// this digest (repo@sha256:... or sha256:...), so a stale request cannot
	// roll back a container that has moved on since.
	ExpectedFromDigest string
}

var (
	// ErrNotRunning refuses an update or rollback of a container that is not
	// running. A stopped container may be stopped on purpose, and only someone
	// starting it decides it should run again. The text is what people read,
	// in the UI and in Dozzle Cloud; assets/composable/containers/
	// containerActions.ts matches it to show it translated.
	ErrNotRunning = errors.New("start the container first")
	// ErrRollbackUnsupported is returned where a rollback cannot run: a swarm
	// service, Kubernetes, Dozzle's own container.
	ErrRollbackUnsupported = errors.New("rollback is not supported")
	// ErrNoRollbackTarget means the container has no previous image Dozzle
	// knows of, or that image is gone from the host.
	ErrNoRollbackTarget = errors.New("no previous image to roll back to")
	// ErrDigestMismatch means the container no longer runs the digest the
	// rollback expected.
	ErrDigestMismatch = errors.New("container no longer runs the expected image")
)

// DigestOf is the sha256:... part of a repo@sha256:... reference, or ref
// itself when it is already a bare digest.
func DigestOf(ref string) string {
	if _, digest, found := strings.Cut(ref, "@"); found {
		return digest
	}
	return ref
}

// SameImageID compares image ids with or without their sha256: prefix.
func SameImageID(a, b string) bool {
	a, b = strings.TrimPrefix(a, "sha256:"), strings.TrimPrefix(b, "sha256:")
	return a != "" && a == b
}

type LogEvent struct {
	Type        LogType `json:"t,omitempty"`
	Message     any     `json:"m,omitempty"`
	RawMessage  string  `json:"rm,omitempty"`
	Timestamp   int64   `json:"ts"`
	Id          uint32  `json:"id,omitempty"`
	Level       string  `json:"l,omitempty"`
	Stream      string  `json:"s,omitempty"`
	ContainerID string  `json:"c,omitempty"`
	// TimestampPrefix is the byte length of a timestamp the app printed at the
	// start of a single line when it agrees with Timestamp, so the UI can hide
	// the duplicate. Message itself is left intact for search, alerts and copy.
	TimestampPrefix int `json:"tp,omitempty"`
}

func (l *LogEvent) HasLevel() bool {
	return l.Level != "unknown"
}

func (l *LogEvent) IsSimple() bool {
	return l.Type == LogTypeSingle || l.Type == LogTypeGroup
}

// MaxGroupTimeDelta is the maximum time difference (in milliseconds) between
// consecutive log lines that can be grouped together. Docker can introduce
// up to ~30ms of jitter between related log lines (e.g., a stack trace).
const MaxGroupTimeDelta = 50

// MaxLogLineBytes bounds one log line, and one grouped entry, anywhere in the
// pipeline. Docker hands a long line over in ~16KB frames and nothing upstream
// limits how many, so a container that writes a single enormous line would
// otherwise be held in memory whole and shipped whole to every consumer. It
// matches the largest line Dozzle Cloud keeps, so clipping here loses nothing
// that would have survived ingest.
const MaxLogLineBytes = 1024 * 1024

// LogTruncationSuffix marks a line clipped to MaxLogLineBytes, so it reads as
// clipped rather than as a log that stops mid-word.
const LogTruncationSuffix = "… [truncated]"

// TruncateLogLine clips s to MaxLogLineBytes without splitting a rune, keeping
// a trailing newline so readers that frame on it still see a line end.
func TruncateLogLine(s string) string {
	if len(s) <= MaxLogLineBytes {
		return s
	}
	newline := strings.HasSuffix(s, "\n")
	n := MaxLogLineBytes
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	s = s[:n] + LogTruncationSuffix
	if newline {
		s += "\n"
	}
	return s
}

func (l *LogEvent) IsCloseToTime(other *LogEvent) bool {
	return math.Abs(float64(l.Timestamp-other.Timestamp)) < MaxGroupTimeDelta
}

func (l *LogEvent) MessageId() int64 {
	return l.Timestamp
}
