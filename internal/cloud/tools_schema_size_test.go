package cloud

import "testing"

// Tool definitions are re-sent on every model call, so their size is a cost on
// every turn. These bounds catch a description growing back unnoticed.
func TestToolSchemaDescriptionsStayCompact(t *testing.T) {
	cases := []struct {
		name string
		desc string
		max  int
	}{
		{"containerIDParam", containerIDParam.Description, 300},
		{"writeContainerIDParam", writeContainerIDParam.Description, 120},
		{"containerExpressionParam", containerExpressionParam.Description, 400},
	}
	for _, c := range cases {
		if n := len([]rune(c.desc)); n > c.max {
			t.Errorf("%s description is %d runes, want <= %d", c.name, n, c.max)
		}
	}
}

func TestWriteToolsUseWriteSchema(t *testing.T) {
	for _, tool := range AvailableTools(true, Principal{}) {
		switch tool.Name {
		case toolStartContainer, toolStopContainer, toolRestartContainer, toolRemoveContainer, toolUpdateContainer:
			if tool.ParametersJson != writeTargetedParams {
				t.Errorf("%s should use writeTargetedParams", tool.Name)
			}
		case toolInspectContainer:
			if tool.ParametersJson != targetedParams {
				t.Errorf("%s should use targetedParams", tool.Name)
			}
		}
	}
}
