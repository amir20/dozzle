package cloud

import (
	"context"
	"testing"

	"github.com/amir20/dozzle/internal/agent/pb"
	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewClient_DefaultURL(t *testing.T) {
	t.Setenv("DOLIGENCE_URL", "")
	client := NewClient("test-key", true, nil, nil)
	assert.Equal(t, "doligence.dozzle.dev:443", client.target)
}

func TestNewClient_CustomURL(t *testing.T) {
	t.Setenv("DOLIGENCE_URL", "https://custom.cloud.dev")
	client := NewClient("test-key", true, nil, nil)
	assert.Equal(t, "custom.cloud.dev:443", client.target)
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
	assert.Len(t, listResp.ToolsJson, 4) // list_containers + 3 actions
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
	assert.Len(t, listResp.ToolsJson, 1) // only list_containers
}

func TestHandleRequest_CallTool_ListContainers(t *testing.T) {
	mockHost := &MockHostService{}
	mockHost.On("ListAllContainers", container.ContainerLabels(nil)).Return([]container.Container{
		{ID: "abc", Name: "nginx", Image: "nginx:latest", State: "running", Host: "local"},
	}, nil)

	client := &Client{
		hostService: mockHost,
	}

	req := &pb.ToolRequest{
		RequestId: "req-3",
		Type: &pb.ToolRequest_CallTool{
			CallTool: &pb.CallToolRequest{
				Name:          "list_containers",
				ArgumentsJson: "",
			},
		},
	}

	resp := client.handleRequest(context.Background(), req)

	callResp := resp.GetCallTool()
	assert.True(t, callResp.Success)
	assert.Contains(t, callResp.ResultJson, "nginx")
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
	mockActioner := &MockContainerActioner{}
	mockActioner.On("Action", mock.Anything, container.Restart).Return(nil)

	mockHost := &MockHostService{}
	mockHost.On("FindContainer", "", "abc123", container.ContainerLabels(nil)).Return(mockActioner, nil)

	client := &Client{
		hostService: mockHost,
	}

	req := &pb.ToolRequest{
		RequestId: "req-5",
		Type: &pb.ToolRequest_CallTool{
			CallTool: &pb.CallToolRequest{
				Name:          "restart_container",
				ArgumentsJson: `{"container_id": "abc123"}`,
			},
		},
	}

	resp := client.handleRequest(context.Background(), req)

	callResp := resp.GetCallTool()
	assert.True(t, callResp.Success)
	mockActioner.AssertCalled(t, "Action", mock.Anything, container.Restart)
}
