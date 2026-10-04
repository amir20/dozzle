// Package updatepolicy decides which containers the auto-update schedule
// touches. Each container ends up with one of three answers:
//
//   - auto: updated on the schedule
//   - manual: checked and shown as "update available", updated by hand
//   - off: never checked, never updated
//
// The dev.dozzle.update label (auto or off) and the instance-wide mode decide
// it together. The mode is one setting in dozzle.yml: off updates no
// container, labelled updates the containers labelled auto, and all updates
// every container that is not labelled off. Dozzle itself follows the
// schedule in every mode; only a label on its own container stops that.
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

// Mode is which containers the schedule updates, besides Dozzle itself.
type Mode string

const (
	// ModeOff updates no container. Dozzle still updates itself.
	ModeOff Mode = "off"
	// ModeLabelled updates the containers labelled dev.dozzle.update=auto. It
	// is what an install that never chose gets, which is exactly how Dozzle
	// behaved before the mode existed.
	ModeLabelled Mode = "labelled"
	// ModeAll updates every container not labelled dev.dozzle.update=off.
	ModeAll Mode = "all"
)

// DefaultMode is the mode of an install that never chose one.
const DefaultMode = ModeLabelled

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
	case ModeOff:
		return ModeOff
	case ModeAll:
		return ModeAll
	default:
		return ModeLabelled
	}
}

// ValidMode reports whether value names a mode exactly.
func ValidMode(value string) bool {
	switch Mode(value) {
	case ModeOff, ModeLabelled, ModeAll:
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

// Resolve decides a container's policy from its labels and the mode.
func Resolve(labels map[string]string, mode Mode) Policy {
	p, labelled := FromLabels(labels)
	switch {
	case labelled && p == Auto && mode == ModeOff:
		// Off means no container moves on its own, labelled or not. The
		// label applies again once the mode allows it.
		return Manual
	case labelled:
		return p
	case mode == ModeAll:
		return Auto
	default:
		return Manual
	}
}
