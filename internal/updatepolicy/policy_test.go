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
		{"manual", map[string]string{Label: "manual"}, Manual, true},
		{"off", map[string]string{Label: "off"}, Off, true},
		{"case and space", map[string]string{Label: " AUTO "}, Auto, true},
		{"unknown value is no label", map[string]string{Label: "sometimes"}, "", false},
		{"legacy auto-update=true", map[string]string{LegacyAutoLabel: "true"}, Auto, true},
		{"legacy auto-update=false is manual", map[string]string{LegacyAutoLabel: "false"}, Manual, true},
		{"legacy auto-update garbage", map[string]string{LegacyAutoLabel: "maybe"}, "", false},
		{"legacy update-check=false", map[string]string{LegacyCheckLabel: "false"}, Off, true},
		{"legacy update-check=true says nothing", map[string]string{LegacyCheckLabel: "true"}, "", false},
		{"check off beats legacy auto", map[string]string{LegacyAutoLabel: "true", LegacyCheckLabel: "false"}, Off, true},
		{"new label beats legacy", map[string]string{Label: "manual", LegacyAutoLabel: "true"}, Manual, true},
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
	assert.Equal(t, ModePicked, ParseMode(""))
	assert.Equal(t, ModePicked, ParseMode("everything"))
	assert.Equal(t, ModeAll, ParseMode("all"))
	assert.Equal(t, ModeDozzle, ParseMode("Dozzle"))
	assert.True(t, ValidMode("picked"))
	assert.False(t, ValidMode(""))
	assert.False(t, ValidMode("Dozzle"))
}

func TestResolvePrecedence(t *testing.T) {
	label := func(v string) map[string]string { return map[string]string{Label: v} }

	// Label beats everything.
	assert.Equal(t, Decision{Off, SourceLabel}, Resolve(label("off"), Auto, ModeAll))
	assert.Equal(t, Decision{Auto, SourceLabel}, Resolve(label("auto"), Off, ModeDozzle))
	assert.Equal(t, Decision{Manual, SourceLabel}, Resolve(label("manual"), Auto, ModeAll))

	// The UI choice beats the mode.
	assert.Equal(t, Decision{Manual, SourceChoice}, Resolve(nil, Manual, ModeAll))
	assert.Equal(t, Decision{Off, SourceChoice}, Resolve(nil, Off, ModeAll))
	assert.Equal(t, Decision{Auto, SourceChoice}, Resolve(nil, Auto, ModePicked))
	// Except auto under Dozzle only, which waits.
	assert.Equal(t, Decision{Manual, SourceDefault}, Resolve(nil, Auto, ModeDozzle))
	assert.Equal(t, Decision{Off, SourceChoice}, Resolve(nil, Off, ModeDozzle))

	// Nobody chose: the mode.
	assert.Equal(t, Decision{Auto, SourceDefault}, Resolve(nil, "", ModeAll))
	assert.Equal(t, Decision{Manual, SourceDefault}, Resolve(nil, "", ModePicked))
	assert.Equal(t, Decision{Manual, SourceDefault}, Resolve(nil, "", ModeDozzle))
}

// An install upgraded with nothing new configured behaves exactly as before:
// the old label opts in, everything else is checked and left alone.
func TestResolveUpgradeKeepsBehaviour(t *testing.T) {
	mode := ParseMode("")
	assert.Equal(t, Auto, Resolve(map[string]string{LegacyAutoLabel: "true"}, "", mode).Policy)
	assert.Equal(t, Manual, Resolve(map[string]string{"com.docker.compose.project": "x"}, "", mode).Policy)
	assert.Equal(t, Off, Resolve(map[string]string{LegacyCheckLabel: "false"}, "", mode).Policy)
}
