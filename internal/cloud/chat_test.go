package cloud

import (
	"context"
	"testing"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/stretchr/testify/assert"
)

func TestChatToolResponse_AnswersToolDiscovery(t *testing.T) {
	client := &Client{version: "test"}
	deps := ToolDeps{EnableActions: true, Principal: UserPrincipal(nil, auth.All)}

	resp := client.chatToolResponse(context.Background(), &pb.ToolRequest{
		RequestId: "req-1",
		Type:      &pb.ToolRequest_ListTools{ListTools: &pb.ListToolsRequest{}},
	}, deps, func(ChatEvent) {})

	assert.Equal(t, "req-1", resp.RequestId)
	list := resp.GetListTools()
	assert.NotNil(t, list, "cloud waits on this reply before the turn has any tools")
	assert.NotEmpty(t, list.Tools)
	assert.Equal(t, "test", list.Version)
}

func TestChatToolResponse_ScopesToolsToTheAsker(t *testing.T) {
	client := &Client{version: "test"}
	deps := ToolDeps{EnableActions: true, Principal: UserPrincipal(nil, auth.Role(0))}

	resp := client.chatToolResponse(context.Background(), &pb.ToolRequest{
		RequestId: "req-1",
		Type:      &pb.ToolRequest_ListTools{ListTools: &pb.ListToolsRequest{}},
	}, deps, func(ChatEvent) {})

	for _, tool := range resp.GetListTools().Tools {
		assert.NotEqual(t, toolStartContainer, tool.Name, "a user without actions is never offered one")
	}
}

func TestChatToolResponse_StreamLogsFallsBackToASnapshot(t *testing.T) {
	mockHost := &MockHostService{}
	mockHost.On("ListAllContainers", container.ContainerLabels(nil)).Return([]container.Container{}, nil)
	mockHost.On("Hosts").Return([]container.Host{{ID: "local", Name: "my-server"}})

	client := &Client{version: "test"}
	deps := ToolDeps{HostService: mockHost, Principal: UserPrincipal(nil, auth.All)}

	var activities []string
	resp := client.chatToolResponse(context.Background(), &pb.ToolRequest{
		RequestId: "req-2",
		Type: &pb.ToolRequest_CallTool{CallTool: &pb.CallToolRequest{
			Name:          toolStreamLogs,
			ArgumentsJson: `{"container_id":"nope"}`,
		}},
	}, deps, func(e ChatEvent) {
		if e.Kind == "status" {
			activities = append(activities, e.Activity)
		}
	})

	// The container does not exist, so this fails — but as the snapshot tool,
	// not as "unknown tool: stream_logs", which the model just calls again.
	assert.NotNil(t, resp.GetCallTool())
	assert.NotContains(t, resp.GetCallTool().Error, "unknown tool")
	assert.Equal(t, []string{"logs"}, activities)
}

func TestChatToolResponse_AnswersRequestsItCannotServe(t *testing.T) {
	client := &Client{version: "test"}

	resp := client.chatToolResponse(context.Background(), &pb.ToolRequest{
		RequestId: "req-3",
		Type:      &pb.ToolRequest_CancelStream{CancelStream: &pb.CancelStreamRequest{}},
	}, ToolDeps{}, func(ChatEvent) {})

	assert.Equal(t, "req-3", resp.RequestId)
	assert.False(t, resp.GetCallTool().Success)
}
