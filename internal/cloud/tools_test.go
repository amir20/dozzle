package cloud

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAvailableTools_WithActionsEnabled(t *testing.T) {
	tools := AvailableTools(true)

	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}

	assert.Contains(t, names, "list_containers")
	assert.Contains(t, names, "start_container")
	assert.Contains(t, names, "stop_container")
	assert.Contains(t, names, "restart_container")
	assert.Len(t, tools, 4)
}

func TestAvailableTools_WithActionsDisabled(t *testing.T) {
	tools := AvailableTools(false)

	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}

	assert.Contains(t, names, "list_containers")
	assert.Len(t, tools, 1)
}

func TestAvailableTools_ParametersAreValid(t *testing.T) {
	tools := AvailableTools(true)

	for _, tool := range tools {
		assert.NotEmpty(t, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.Parameters)
	}
}
