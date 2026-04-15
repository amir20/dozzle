package cloud

import (
	"context"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewClient_DefaultURL(t *testing.T) {
	t.Setenv("AGENT_URL", "")
	client := NewClient(true, nil, nil, func() string { return "test-key" })
	assert.Equal(t, "agent.doligence.dozzle.dev:443", client.target)
}

func TestNewClient_CustomURL(t *testing.T) {
	t.Setenv("AGENT_URL", "https://custom.cloud.dev")
	client := NewClient(true, nil, nil, func() string { return "test-key" })
	assert.Equal(t, "custom.cloud.dev:443", client.target)
	assert.False(t, client.plaintext)
}

func TestNewClient_PlaintextURL(t *testing.T) {
	t.Setenv("AGENT_URL", "http://localhost:7008")
	client := NewClient(true, nil, nil, func() string { return "test-key" })
	assert.Equal(t, "localhost:7008", client.target)
	assert.True(t, client.plaintext)
}

func TestHandleRequest_ListTools(t *testing.T) {
	client := &Client{
		enableActions: true,
	}

	req := &pb.ToolRequest{
		RequestId: "req-1",
		Type: &pb.ToolRequest_ListTools{
			ListTools: &pb.ListToolsRequest{},
		},
	}

	resp := client.handleRequest(context.Background(), req)

	assert.Equal(t, "req-1", resp.RequestId)
	listResp := resp.GetListTools()
	assert.NotNil(t, listResp)
	assert.Len(t, listResp.Tools, 15) // list_hosts + find_containers + list_running/all + get_stats + fetch_logs + stream_logs + inspect_container + 3 actions + update + deploy + list_versions + rollback
}

func TestHandleRequest_ListTools_ActionsDisabled(t *testing.T) {
	client := &Client{
		enableActions: false,
	}

	req := &pb.ToolRequest{
		RequestId: "req-2",
		Type: &pb.ToolRequest_ListTools{
			ListTools: &pb.ListToolsRequest{},
		},
	}

	resp := client.handleRequest(context.Background(), req)

	listResp := resp.GetListTools()
	assert.Len(t, listResp.Tools, 8) // list_hosts + find_containers + list_running/all + get_stats + fetch_logs + stream_logs + inspect_container
}

func TestHandleRequest_CallTool_ListContainers(t *testing.T) {
	mockHost := &MockHostService{}
	mockHost.On("ListAllContainers", container.ContainerLabels(nil)).Return([]container.Container{
		{ID: "abc", Name: "nginx", Image: "nginx:latest", State: "running", Host: "local"},
	}, nil)
	mockHost.On("Hosts").Return([]container.Host{{ID: "local", Name: "my-server"}})

	client := &Client{
		hostService: mockHost,
	}

	req := &pb.ToolRequest{
		RequestId: "req-3",
		Type: &pb.ToolRequest_CallTool{
			CallTool: &pb.CallToolRequest{
				Name:          "list_running_containers",
				ArgumentsJson: "",
			},
		},
	}

	resp := client.handleRequest(context.Background(), req)

	callResp := resp.GetCallTool()
	assert.True(t, callResp.Success)
	result := callResp.GetListContainers()
	assert.NotNil(t, result)
	assert.Len(t, result.Containers, 1)
	assert.Equal(t, "nginx", result.Containers[0].Name)
}

func TestHandleRequest_CallTool_UnknownTool(t *testing.T) {
	mockHost := &MockHostService{}

	client := &Client{
		hostService: mockHost,
	}

	req := &pb.ToolRequest{
		RequestId: "req-4",
		Type: &pb.ToolRequest_CallTool{
			CallTool: &pb.CallToolRequest{
				Name:          "nonexistent",
				ArgumentsJson: "",
			},
		},
	}

	resp := client.handleRequest(context.Background(), req)

	callResp := resp.GetCallTool()
	assert.False(t, callResp.Success)
	assert.Contains(t, callResp.Error, "unknown tool")
}

func TestHandleRequest_CallTool_RestartContainer(t *testing.T) {
	mockClient := &MockClientService{}
	mockClient.On("ContainerAction", mock.Anything, mock.Anything, container.Restart).Return(nil)

	cs := container_support.NewContainerService(mockClient, container.Container{ID: "abc123"})

	mockHost := &MockHostService{}
	mockHost.On("FindContainer", "local", "abc123", container.ContainerLabels(nil)).Return(cs, nil)

	client := &Client{
		hostService:   mockHost,
		enableActions: true,
	}

	req := &pb.ToolRequest{
		RequestId: "req-5",
		Type: &pb.ToolRequest_CallTool{
			CallTool: &pb.CallToolRequest{
				Name:          "restart_container",
				ArgumentsJson: `{"container_id": "abc123", "host_id": "local"}`,
			},
		},
	}

	resp := client.handleRequest(context.Background(), req)

	callResp := resp.GetCallTool()
	assert.True(t, callResp.Success)
	mockClient.AssertCalled(t, "ContainerAction", mock.Anything, mock.Anything, container.Restart)
}
