package analytics

import (
	"context"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

// UsageKeys are the only counters the usage beacon carries. Anything else a
// caller or the browser names is dropped, so a key can never smuggle a name.
var UsageKeys = []string{
	"view.container", "view.merged", "view.group", "view.host", "view.stack", "view.service", "view.namespace",
	"logs.search", "logs.older", "logs.download", "logs.sql", "palette.open", "pinned.open",
	"action.start", "action.stop", "action.restart", "action.update", "action.remove",
	"shell.exec", "shell.attach", "image.check", "image.update",
	"notify.log", "notify.event", "notify.metric", "rules.create", "rules.edit",
	"host.add.ok", "host.add.refused", "host.add.cert", "host.add.duplicate", "host.add.timeout", "host.add.other",
	"wizard.shown", "wizard.finished", "wizard.skip.login", "wizard.skip.actions", "wizard.skip.hosts", "wizard.skip.cloud", "wizard.skip.update",
	"cloud.welcome", "cloud.connect", "cloud.chat", "stream.reconnect", "agent.disconnect",
	"memory.chip.shown", "memory.chip.hover", "memory.chip.open",
}

// BrowserUsageKeys are the counters only the browser can see, and so the only
// ones POST /api/usage accepts. The rest are counted where they happen on the
// server, and a report must not be able to inflate them.
var BrowserUsageKeys = map[string]bool{
	"logs.search": true, "logs.sql": true, "palette.open": true, "pinned.open": true,
	"wizard.shown": true, "wizard.finished": true, "wizard.skip.login": true, "wizard.skip.actions": true,
	"wizard.skip.hosts": true, "wizard.skip.cloud": true, "wizard.skip.update": true,
	"cloud.welcome": true, "cloud.connect": true, "cloud.chat": true, "stream.reconnect": true,
	"memory.chip.shown": true, "memory.chip.hover": true, "memory.chip.open": true,
}

// maxLocales bounds the locale map, which the browser fills: a client sending a
// new code every request must not grow it without end.
const maxLocales = 32

var localePattern = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z]{2,4})?$`)

// Usage counts what people do between two usage beacons. Counting is a map read
// and an atomic add, so it is safe to call from any hot path.
type Usage struct {
	enabled atomic.Bool
	counts  map[string]*atomic.Int64 // fixed at construction, never written after

	mu      sync.Mutex
	locales map[string]int
	minutes atomic.Int64
}

func NewUsage() *Usage {
	u := &Usage{counts: make(map[string]*atomic.Int64, len(UsageKeys)), locales: map[string]int{}}
	for _, k := range UsageKeys {
		u.counts[k] = new(atomic.Int64)
	}
	u.enabled.Store(true)
	return u
}

// Default is the process-wide counter set.
var Default = NewUsage()

// Disable turns counting into a no-op, for --no-analytics.
func (u *Usage) Disable() { u.enabled.Store(false) }

func (u *Usage) Enabled() bool { return u.enabled.Load() }

// Count adds one to key. Unknown keys are ignored.
func Count(key string) { Default.Add(key, 1) }

func (u *Usage) Add(key string, n int) {
	if n <= 0 || !u.enabled.Load() {
		return
	}
	if c, ok := u.counts[key]; ok {
		c.Add(int64(n))
	}
}

// AddLocale records one UI session in a language.
func (u *Usage) AddLocale(code string) {
	if !u.enabled.Load() || !localePattern.MatchString(code) {
		return
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.addLocaleLocked(code, 1)
}

// addLocaleLocked adds n sessions in code, unless the map is full and code is
// new. u.mu must be held.
func (u *Usage) addLocaleLocked(code string, n int) {
	if _, ok := u.locales[code]; ok || len(u.locales) < maxLocales {
		u.locales[code] += n
	}
}

func (u *Usage) AddMinutes(n int) {
	if n > 0 && u.enabled.Load() {
		u.minutes.Add(int64(n))
	}
}

// UsageSnapshot is what one usage beacon carries.
type UsageSnapshot struct {
	Counts  map[string]int
	Locales map[string]int
	Minutes int
}

func (s UsageSnapshot) Empty() bool {
	return len(s.Counts) == 0 && len(s.Locales) == 0 && s.Minutes == 0
}

// Take returns everything counted so far and starts over from zero. Counts that
// arrive while it runs land in either this snapshot or the next, never both.
func (u *Usage) Take() UsageSnapshot {
	s := UsageSnapshot{Counts: map[string]int{}}
	for k, c := range u.counts {
		if n := c.Swap(0); n > 0 {
			s.Counts[k] = int(n)
		}
	}
	u.mu.Lock()
	s.Locales = u.locales
	u.locales = map[string]int{}
	u.mu.Unlock()
	s.Minutes = int(u.minutes.Swap(0))
	return s
}

// Restore puts a snapshot back after a send failed, so a bad day on the network
// does not lose it.
func (u *Usage) Restore(s UsageSnapshot) {
	for k, n := range s.Counts {
		u.Add(k, n)
	}
	u.mu.Lock()
	for code, n := range s.Locales {
		u.addLocaleLocked(code, n)
	}
	u.mu.Unlock()
	u.AddMinutes(s.Minutes)
}

// BucketMinutes reports active minutes as a range rather than an exact number.
func BucketMinutes(n int) string {
	switch {
	case n <= 0:
		return "0"
	case n <= 10:
		return "1-10"
	case n <= 60:
		return "11-60"
	case n <= 240:
		return "61-240"
	default:
		return "241+"
	}
}

// BucketUsers reports a user count as a range. Zero is "" so it is left out.
func BucketUsers(n int) string {
	switch {
	case n <= 0:
		return ""
	case n == 1:
		return "1"
	case n <= 5:
		return "2-5"
	case n <= 20:
		return "6-20"
	default:
		return "21+"
	}
}

// RunUsageBeacon sends a usage beacon every interval until ctx ends. base builds
// the install facts at send time. A period with nothing counted sends nothing.
func RunUsageBeacon(ctx context.Context, u *Usage, interval time.Duration, base func() types.BeaconEvent, send func(types.BeaconEvent) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			SendUsage(u, base, send)
		}
	}
}

// SendUsage sends one usage beacon and reports whether it went out.
func SendUsage(u *Usage, base func() types.BeaconEvent, send func(types.BeaconEvent) error) bool {
	if !u.Enabled() {
		return false
	}
	s := u.Take()
	if s.Empty() {
		return false
	}
	b := base()
	b.Name = "usage"
	b.Usage = s.Counts
	b.Locales = s.Locales
	b.ActiveMinutes = BucketMinutes(s.Minutes)
	if err := send(b); err != nil {
		log.Debug().Err(err).Msg("error sending usage beacon")
		u.Restore(s)
		return false
	}
	return true
}
