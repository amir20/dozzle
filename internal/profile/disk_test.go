package profile

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsRoundTrip(t *testing.T) {
	in := `{"showAppIcons":false,"showImageUpdateAlert":true,"terminalFontSize":16}`

	var s Settings
	require.NoError(t, json.Unmarshal([]byte(in), &s))

	out, err := json.Marshal(s)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out, &got))
	assert.Equal(t, false, got["showAppIcons"])
	assert.Equal(t, true, got["showImageUpdateAlert"])
	assert.Equal(t, float64(16), got["terminalFontSize"])
}

func TestSettingsOmitsUnsetFields(t *testing.T) {
	out, err := json.Marshal(Settings{})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out, &got))
	assert.NotContains(t, got, "showAppIcons")
	assert.NotContains(t, got, "showImageUpdateAlert")
	assert.NotContains(t, got, "terminalFontSize")
}
