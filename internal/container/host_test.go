package container

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveHostID(t *testing.T) {
	tests := []struct {
		name       string
		candidates []string
		want       string
	}{
		{"takes the first non-empty", []string{"override", "engine"}, "override"},
		{"skips a source with nothing to say", []string{"", "", "engine"}, "engine"},
		{"empty when every source declines", []string{"", ""}, ""},
		{"empty with no sources at all", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ResolveHostID(tt.candidates...))
		})
	}
}
