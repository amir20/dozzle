package cloud

import (
	"testing"

	"github.com/amir20/dozzle/internal/imagecheck"
)

// Tool definitions are re-sent on every model call, so their size is a cost on
// every turn. These bounds catch a definition growing back unnoticed. They
// measure what is actually sent — the description plus the JSON-serialized
// parameter schema, in bytes — so escaped quotes and multibyte characters
// count too. Budgets are the current size plus ~10% headroom.
func TestToolSchemasStayCompact(t *testing.T) {
	budget := map[string]int{
		toolFindContainers:           900,
		toolInspectContainer:         900,
		toolStartContainer:           650,
		toolStopContainer:            650,
		toolRestartContainer:         650,
		toolRemoveContainer:          800,
		toolUpdateContainer:          850,
		toolRollbackContainer:        950,
		toolCreateLogNotification:    1800,
		toolCreateMetricNotification: 1950,
		toolCreateEventNotification:  1800,
		toolCheckImageUpdates:        1130,
	}
	for _, tool := range AvailableTools(ToolDeps{EnableActions: true, ImageCheckMode: imagecheck.ModeAutomatic}) {
		max, ok := budget[tool.Name]
		if !ok {
			continue
		}
		if n := len(tool.Description) + len(tool.ParametersJson); n > max {
			t.Errorf("%s definition is %d bytes, want <= %d", tool.Name, n, max)
		}
	}
}

func TestWriteToolsUseWriteSchema(t *testing.T) {
	for _, tool := range AvailableTools(ToolDeps{EnableActions: true}) {
		switch tool.Name {
		case toolStartContainer, toolStopContainer, toolRestartContainer, toolRemoveContainer, toolUpdateContainer:
			if tool.ParametersJson != writeTargetedParams {
				t.Errorf("%s should use writeTargetedParams", tool.Name)
			}
		case toolRollbackContainer:
			if tool.ParametersJson != rollbackContainerParams {
				t.Errorf("%s should use rollbackContainerParams", tool.Name)
			}
		case toolInspectContainer:
			if tool.ParametersJson != targetedParams {
				t.Errorf("%s should use targetedParams", tool.Name)
			}
		}
	}
}
