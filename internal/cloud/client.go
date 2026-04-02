package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/cloud/pb"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const (
	initialBackoff  = 1 * time.Second
	maxBackoff      = 30 * time.Second
	backoffFactor   = 2
	jitterFraction  = 0.1
	apiKeyPollDelay = 30 * time.Second
)

// Client manages the gRPC connection to Dozzle Cloud
type Client struct {
	enableActions bool
	labels        container.ContainerLabels
	hostService   ToolHostService
	apiKeyFunc    func() string
	target        string
}

// NewClient creates a new cloud gRPC client.
// apiKeyFunc is called to get the current cloud API key — it may return ""
// if no cloud dispatcher is configured yet, in which case the client waits.
func NewClient(enableActions bool, labels container.ContainerLabels, hostService ToolHostService, apiKeyFunc func() string) *Client {
	cloudURL := os.Getenv("DOLIGENCE_URL")
	if cloudURL == "" {
		cloudURL = "https://doligence.dozzle.dev"
	}

	// Convert https://host to host:443 for gRPC dial target
	target := cloudURL
	target = strings.TrimPrefix(target, "https://")
	target = strings.TrimPrefix(target, "http://")
	if !strings.Contains(target, ":") {
		target = target + ":443"
	}

	return &Client{
		enableActions: enableActions,
		labels:        labels,
		hostService:   hostService,
		apiKeyFunc:    apiKeyFunc,
		target:        target,
	}
}

// Run starts the cloud client loop. It connects to the cloud gRPC endpoint
// and processes tool requests. Reconnects automatically on failure.
// If no cloud API key is configured, it polls until one appears.
// Blocks until ctx is cancelled.
func (c *Client) Run(ctx context.Context) {
	backoff := initialBackoff

	for {
		apiKey := c.apiKeyFunc()
		if apiKey == "" {
			// No cloud dispatcher configured yet, wait and check again
			log.Debug().Msg("no cloud API key found, waiting for cloud setup")
			select {
			case <-ctx.Done():
				return
			case <-time.After(apiKeyPollDelay):
			}
			continue
		}

		wasConnected, err := c.connect(ctx, apiKey)
		if ctx.Err() != nil {
			log.Debug().Msg("cloud client stopped")
			return
		}

		if wasConnected {
			// Was connected and then disconnected — reset backoff
			backoff = initialBackoff
		}

		if err != nil {
			log.Debug().Err(err).Dur("backoff", backoff).Msg("cloud connection failed, reconnecting")
		}

		jitter := time.Duration(float64(backoff) * jitterFraction * rand.Float64())
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff + jitter):
		}

		backoff = min(backoff*backoffFactor, maxBackoff)
	}
}

// connect establishes a gRPC stream to the cloud and processes requests.
// Returns wasConnected=true if the stream was successfully established and
// at least one message was received before disconnecting.
func (c *Client) connect(ctx context.Context, apiKey string) (wasConnected bool, err error) {
	// Uses system cert pool for TLS to the public cloud endpoint
	conn, err := grpc.NewClient(c.target,
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
		grpc.WithUserAgent(dispatcher.UserAgent),
	)
	if err != nil {
		return false, fmt.Errorf("failed to dial cloud: %w", err)
	}
	defer conn.Close()

	client := pb.NewCloudToolServiceClient(conn)

	md := metadata.Pairs("x-api-key", apiKey)
	streamCtx := metadata.NewOutgoingContext(ctx, md)

	stream, err := client.ToolStream(streamCtx)
	if err != nil {
		return false, fmt.Errorf("failed to open tool stream: %w", err)
	}

	log.Debug().Str("target", c.target).Msg("connected to cloud tool service")

	// Use stream context so tool executions are tied to the stream lifetime
	streamLifetime := stream.Context()

	for {
		req, err := stream.Recv()
		if err != nil {
			return wasConnected, fmt.Errorf("stream recv error: %w", err)
		}
		wasConnected = true

		resp := c.handleRequest(streamLifetime, req)

		if err := stream.Send(resp); err != nil {
			return wasConnected, fmt.Errorf("stream send error: %w", err)
		}
	}
}

func (c *Client) handleRequest(ctx context.Context, req *pb.ToolRequest) *pb.ToolResponse {
	resp := &pb.ToolResponse{
		RequestId: req.RequestId,
	}

	switch t := req.Type.(type) {
	case *pb.ToolRequest_ListTools:
		tools := AvailableTools(c.enableActions)
		var toolsJSON []string
		for _, tool := range tools {
			data, err := json.Marshal(tool)
			if err != nil {
				log.Error().Err(err).Str("tool", tool.Name).Msg("failed to marshal tool definition")
				continue
			}
			toolsJSON = append(toolsJSON, string(data))
		}
		resp.Type = &pb.ToolResponse_ListTools{
			ListTools: &pb.ListToolsResponse{
				ToolsJson: toolsJSON,
			},
		}

	case *pb.ToolRequest_CallTool:
		result, err := ExecuteTool(ctx, t.CallTool.Name, t.CallTool.ArgumentsJson, c.hostService, c.labels)
		if err != nil {
			resp.Type = &pb.ToolResponse_CallTool{
				CallTool: &pb.CallToolResponse{
					Success: false,
					Error:   err.Error(),
				},
			}
		} else {
			resp.Type = &pb.ToolResponse_CallTool{
				CallTool: &pb.CallToolResponse{
					Success:    true,
					ResultJson: result,
				},
			}
		}

	default:
		log.Warn().Msg("received unknown tool request type")
	}

	return resp
}
