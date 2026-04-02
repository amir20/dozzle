package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/agent/pb"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const (
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
	backoffFactor  = 2
	jitterFraction = 0.1
)

// Client manages the gRPC connection to Dozzle Cloud
type Client struct {
	apiKey        string
	enableActions bool
	labels        container.ContainerLabels
	hostService   ToolHostService
	target        string
}

// NewClient creates a new cloud gRPC client
func NewClient(apiKey string, enableActions bool, labels container.ContainerLabels, hostService ToolHostService) *Client {
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
		apiKey:        apiKey,
		enableActions: enableActions,
		labels:        labels,
		hostService:   hostService,
		target:        target,
	}
}

// Run starts the cloud client loop. It connects to the cloud gRPC endpoint
// and processes tool requests. Reconnects automatically on failure.
// Blocks until ctx is cancelled.
func (c *Client) Run(ctx context.Context) {
	backoff := initialBackoff

	for {
		err := c.connect(ctx)
		if ctx.Err() != nil {
			log.Debug().Msg("cloud client stopped")
			return
		}

		if err != nil {
			log.Warn().Err(err).Dur("backoff", backoff).Msg("cloud connection failed, reconnecting")
		} else {
			// Reset backoff after a successful connection that later disconnected
			backoff = initialBackoff
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

func (c *Client) connect(ctx context.Context) error {
	conn, err := grpc.NewClient(c.target,
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
		grpc.WithUserAgent(dispatcher.UserAgent),
	)
	if err != nil {
		return fmt.Errorf("failed to dial cloud: %w", err)
	}
	defer conn.Close()

	client := pb.NewCloudToolServiceClient(conn)

	md := metadata.Pairs("x-api-key", c.apiKey)
	streamCtx := metadata.NewOutgoingContext(ctx, md)

	stream, err := client.ToolStream(streamCtx)
	if err != nil {
		return fmt.Errorf("failed to open tool stream: %w", err)
	}

	log.Info().Str("target", c.target).Msg("connected to cloud tool service")

	// Use stream context so tool executions are tied to the stream lifetime
	streamLifetime := stream.Context()

	for {
		req, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("stream recv error: %w", err)
		}

		resp := c.handleRequest(streamLifetime, req)

		if err := stream.Send(resp); err != nil {
			return fmt.Errorf("stream send error: %w", err)
		}
	}
}

// CheckProPlan checks if the cloud API key has a pro plan by calling the status endpoint.
func CheckProPlan(ctx context.Context, apiKey string) (bool, error) {
	cloudURL := os.Getenv("DOLIGENCE_URL")
	if cloudURL == "" {
		cloudURL = "https://doligence.dozzle.dev"
	}

	statusURL := fmt.Sprintf("%s/api/status", cloudURL)
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create status request: %w", err)
	}
	req.Header.Set("User-Agent", dispatcher.UserAgent)
	req.Header.Set("X-API-Key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check cloud status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var status struct {
		Plan string `json:"plan"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return false, fmt.Errorf("failed to decode status response: %w", err)
	}

	return status.Plan == "pro", nil
}

func (c *Client) handleRequest(ctx context.Context, req *pb.ToolRequest) *pb.ToolResponse {
	resp := &pb.ToolResponse{
		RequestId: req.RequestId,
	}

	switch t := req.Type.(type) {
	case *pb.ToolRequest_ListTools:
		tools := AvailableTools(c.enableActions)
		toolsJSON := make([]string, len(tools))
		for i, tool := range tools {
			data, err := json.Marshal(tool)
			if err != nil {
				log.Error().Err(err).Str("tool", tool.Name).Msg("failed to marshal tool definition")
				continue
			}
			toolsJSON[i] = string(data)
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
