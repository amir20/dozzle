package cloud

import (
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// AvailableTools returns the list of tool definitions based on configuration.
// list_containers is always available. Action tools require enableActions.
func AvailableTools(enableActions bool) []openai.FunctionDefinition {
	tools := []openai.FunctionDefinition{
		{
			Name:        "list_containers",
			Description: "List all Docker containers with their current state, name, image, and host",
			Parameters: jsonschema.Definition{
				Type:       jsonschema.Object,
				Properties: map[string]jsonschema.Definition{},
			},
		},
	}

	if enableActions {
		tools = append(tools,
			openai.FunctionDefinition{
				Name:        "start_container",
				Description: "Start a stopped Docker container",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"container_id": {
							Type:        jsonschema.String,
							Description: "The container ID to start",
						},
					},
					Required: []string{"container_id"},
				},
			},
			openai.FunctionDefinition{
				Name:        "stop_container",
				Description: "Stop a running Docker container",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"container_id": {
							Type:        jsonschema.String,
							Description: "The container ID to stop",
						},
					},
					Required: []string{"container_id"},
				},
			},
			openai.FunctionDefinition{
				Name:        "restart_container",
				Description: "Restart a Docker container",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"container_id": {
							Type:        jsonschema.String,
							Description: "The container ID to restart",
						},
					},
					Required: []string{"container_id"},
				},
			},
		)
	}

	return tools
}
