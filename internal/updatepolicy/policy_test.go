package updatepolicy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromLabels(t *testing.T) {
	cases := []struct {
		name   string
		labels map[string]string
		want   Policy
		ok     bool
	}{
		{"none", nil, "", false},
		{"auto", map[string]string{Label: "auto"}, Auto, true},
		{"off", map[string]string{Label: "off"}, Off, true},
		{"case and space", map[string]string{Label: " AUTO "}, Auto, true},
		{"unknown value is no label", map[string]string{Label: "sometimes"}, "", false},
		{"legacy auto-update=true", map[string]string{LegacyAutoLabel: "true"}, Auto, true},
		{"legacy auto-update=false is manual", map[string]string{LegacyAutoLabel: "false"}, Manual, true},
		{"legacy auto-update garbage", map[string]string{LegacyAutoLabel: "maybe"}, "", false},
		{"legacy update-check=false", map[string]string{LegacyCheckLabel: "false"}, Off, true},
		{"legacy update-check=true says nothing", map[string]string{LegacyCheckLabel: "true"}, "", false},
		{"check off beats legacy auto", map[string]string{LegacyAutoLabel: "true", LegacyCheckLabel: "false"}, Off, true},
		{"new label beats legacy", map[string]string{Label: "off", LegacyAutoLabel: "true"}, Off, true},
		{"new label beats legacy off", map[string]string{Label: "auto", LegacyCheckLabel: "false"}, Auto, true},
		{"invalid new label falls back to legacy", map[string]string{Label: "x", LegacyAutoLabel: "true"}, Auto, true},
	}
	for _, c := range cases {
		got, ok := FromLabels(c.labels)
		assert.Equal(t, c.ok, ok, c.name)
		assert.Equal(t, c.want, got, c.name)
	}
}

func TestParseMode(t *testing.T) {
	assert.Equal(t, ModeLabelled, ParseMode(""))
	assert.Equal(t, ModeLabelled, ParseMode("everything"))
	assert.Equal(t, ModeAll, ParseMode("all"))
	assert.Equal(t, ModeOff, ParseMode("Off"))
	assert.True(t, ValidMode("labelled"))
	assert.False(t, ValidMode(""))
	assert.False(t, ValidMode("Off"))
}

func TestResolve(t *testing.T) {
	label := func(v string) map[string]string { return map[string]string{Label: v} }

	// Off moves nothing, not even a container labelled auto.
	assert.Equal(t, Manual, Resolve(label("auto"), ModeOff))
	assert.Equal(t, Manual, Resolve(nil, ModeOff))
	assert.Equal(t, Off, Resolve(label("off"), ModeOff))

	// Labelled: only the label opts in.
	assert.Equal(t, Auto, Resolve(label("auto"), ModeLabelled))
	assert.Equal(t, Manual, Resolve(nil, ModeLabelled))
	assert.Equal(t, Off, Resolve(label("off"), ModeLabelled))

	// All: everything but a label that says otherwise.
	assert.Equal(t, Auto, Resolve(nil, ModeAll))
	assert.Equal(t, Auto, Resolve(label("auto"), ModeAll))
	assert.Equal(t, Off, Resolve(label("off"), ModeAll))
	assert.Equal(t, Manual, Resolve(map[string]string{LegacyAutoLabel: "false"}, ModeAll))
}

// An install upgraded with nothing new configured behaves exactly as before:
// the old label opts in, everything else is checked and left alone.
func TestResolveUpgradeKeepsBehaviour(t *testing.T) {
	mode := ParseMode("")
	assert.Equal(t, Auto, Resolve(map[string]string{LegacyAutoLabel: "true"}, mode))
	assert.Equal(t, Manual, Resolve(map[string]string{"com.docker.compose.project": "x"}, mode))
	assert.Equal(t, Off, Resolve(map[string]string{LegacyCheckLabel: "false"}, mode))
}
