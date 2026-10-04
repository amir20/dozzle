// Package updatepolicy decides which containers the auto-update schedule
// touches. There is one question per container, with three answers:
//
//   - auto: updated on the schedule
//   - manual: checked and shown as "update available", updated by hand
//   - off: never checked, never updated
//
// The answer comes from, in order: the dev.dozzle.update label, the choice
// saved from the UI in dozzle.yml, and the instance-wide mode for containers
// nobody chose for. A label always wins, because it is what the operator wrote
// in their compose file.
//
// The package has no dependencies, so both imagecheck and container can ask it.
package updatepolicy

import "strings"

// Policy is one container's answer.
type Policy string

const (
	Auto   Policy = "auto"
	Manual Policy = "manual"
	Off    Policy = "off"
)

// Mode is how a container nobody chose for is treated.
type Mode string

const (
	// ModeDozzle updates only Dozzle itself. A container set to auto from the
	// UI waits, but a label still wins.
	ModeDozzle Mode = "dozzle"
	// ModePicked updates Dozzle and the containers set to auto, by label or
	// from the UI. It is what an install that never chose gets, which is
	// exactly how Dozzle behaved before the mode existed.
	ModePicked Mode = "picked"
	// ModeAll updates every container that is not set to manual or off.
	ModeAll Mode = "all"
)

// DefaultMode is the mode of an install that never chose one.
const DefaultMode = ModePicked

// Label is the one label that chooses a container's policy.
const Label = "dev.dozzle.update"

// Older labels, still read so nobody has to edit a compose file on upgrade.
// Neither is documented any more.
const (
	// LegacyAutoLabel set to true is auto.
	LegacyAutoLabel = "dev.dozzle.auto-update"
	// LegacyCheckLabel set to false is off.
	LegacyCheckLabel = "dev.dozzle.update-check"
)

// Parse reads a policy, accepting the usual spellings of yes and no for
// auto and off. Anything else is not a policy.
func Parse(value string) (Policy, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "auto", "automatic", "true", "on", "yes", "1":
		return Auto, true
	case "manual":
		return Manual, true
	case "off", "false", "no", "0", "never":
		return Off, true
	default:
		return "", false
	}
}

// ParseMode reads a mode. Empty or anything unknown is DefaultMode, so a typo
// in dozzle.yml never widens what gets updated.
func ParseMode(value string) Mode {
	switch Mode(strings.ToLower(strings.TrimSpace(value))) {
	case ModeDozzle:
		return ModeDozzle
	case ModeAll:
		return ModeAll
	default:
		return ModePicked
	}
}

// ValidMode reports whether value names a mode exactly.
func ValidMode(value string) bool {
	switch Mode(value) {
	case ModeDozzle, ModePicked, ModeAll:
		return true
	default:
		return false
	}
}

// FromLabels is the policy container labels choose, if any. dev.dozzle.update
// wins over the older labels. Of those, update-check=false is off whatever
// auto-update says, since it also stopped the check, and an explicit
// auto-update=false is manual: someone wrote it to keep the container still.
func FromLabels(labels map[string]string) (Policy, bool) {
	if value, ok := labels[Label]; ok {
		if p, ok := Parse(value); ok {
			return p, true
		}
	}
	if value, ok := labels[LegacyCheckLabel]; ok {
		if p, ok := Parse(value); ok && p == Off {
			return Off, true
		}
	}
	if value, ok := labels[LegacyAutoLabel]; ok {
		switch p, _ := Parse(value); p {
		case Auto:
			return Auto, true
		case Off:
			return Manual, true
		}
	}
	return "", false
}

// Source is where a decision came from.
type Source string

const (
	SourceLabel  Source = "label"
	SourceChoice Source = "choice"
	// SourceDefault is the mode, for a container nobody chose for, and for a
	// UI choice of auto that ModeDozzle holds back.
	SourceDefault Source = "default"
)

// Decision is a container's effective policy and why.
type Decision struct {
	Policy Policy `json:"policy"`
	Source Source `json:"source"`
}

// Resolve decides a container's policy from its labels, the UI choice saved
// for it ("" when none) and the mode.
func Resolve(labels map[string]string, choice Policy, mode Mode) Decision {
	if p, ok := FromLabels(labels); ok {
		return Decision{Policy: p, Source: SourceLabel}
	}
	switch choice {
	case Auto:
		// Dozzle only means the schedule leaves everything else alone. The
		// choice is kept, and applies again once the mode allows picks.
		if mode == ModeDozzle {
			return Decision{Policy: Manual, Source: SourceDefault}
		}
		return Decision{Policy: Auto, Source: SourceChoice}
	case Manual, Off:
		return Decision{Policy: choice, Source: SourceChoice}
	}
	if mode == ModeAll {
		return Decision{Policy: Auto, Source: SourceDefault}
	}
	return Decision{Policy: Manual, Source: SourceDefault}
}
