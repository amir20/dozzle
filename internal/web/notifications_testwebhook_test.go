package web

import (
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_testNotification_follows_template makes sure a webhook test renders templates written
// for event and metric alerts. A log sample leaves .Event and .Stat nil. See #5527.
func Test_testNotification_follows_template(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     string
	}{
		{"event", `{{ .Event.Attributes.currentVersion }} -> {{ .Event.Attributes.latestVersion }}`, "1.27.0 -> 1.27.1"},
		{"metric", `{{ .Stat.CPUPercent }}`, "85"},
		{"log", `{{ .Log.Message }}`, "This is a test log message from Dozzle"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := template.New("t").Option("missingkey=error").Parse(tt.template)
			require.NoError(t, err)
			var out strings.Builder
			require.NoError(t, tmpl.Execute(&out, testNotification(tt.template)))
			assert.Equal(t, tt.want, out.String())
		})
	}
}
