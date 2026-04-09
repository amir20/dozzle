package cloud

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/amir20/dozzle/internal/support/cli"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	initialBackoff        = 1 * time.Second
	maxBackoff            = 30 * time.Second
	backoffFactor         = 2
	jitterFraction        = 0.1
	maxConcurrent         = 5
	unauthenticatedPause  = 1 * time.Hour
)

// Client manages the gRPC connection to Dozzle Cloud
type Client struct {
	enableActions   bool
	labels          container.ContainerLabels
	hostService     ToolHostService
	apiKeyFunc      func() string
	target          string
	plaintext       bool
	toolSem         *semaphore.Weighted
	cachedTools     []*pb.ToolDefinition
	toolsOnce       sync.Once
	startCh         chan struct{}
	activeStreams    sync.Map // requestID -> context.CancelFunc
}

// NewClient creates a new cloud gRPC client.
// apiKeyFunc is called to get the current cloud API key — it may return ""
// if no cloud dispatcher is configured yet, in which case the client waits.
func NewClient(enableActions bool, labels container.ContainerLabels, hostService ToolHostService, apiKeyFunc func() string) *Client {
	cloudURL := os.Getenv("AGENT_URL")
	if cloudURL == "" {
		cloudURL = "https://agent.doligence.dozzle.dev"
	}

	// Support plaintext for local dev (AGENT_URL=http://localhost:7008)
	plaintext := strings.HasPrefix(cloudURL, "http://")

	target := cloudURL
	target = strings.TrimPrefix(target, "https://")
	target = strings.TrimPrefix(target, "http://")
	if !strings.Contains(target, ":") {
		if plaintext {
			target = target + ":80"
		} else {
			target = target + ":443"
		}
	}

	return &Client{
		enableActions: enableActions,
		labels:        labels,
		hostService:   hostService,
		apiKeyFunc:    apiKeyFunc,
		target:        target,
		plaintext:     plaintext,
		toolSem:       semaphore.NewWeighted(maxConcurrent),
		startCh:       make(chan struct{}, 1),
	}
}

// Notify signals the client to attempt a connection. Safe to call multiple times.
// Use this when a cloud dispatcher is added or when the status page is viewed.
func (c *Client) Notify() {
	select {
	case c.startCh <- struct{}{}:
	default:
	}
}

// Run blocks until signaled via Notify(), then connects to the cloud gRPC endpoint
// and processes tool requests. Reconnects automatically on failure.
// Does nothing until Notify() is called — zero overhead for non-cloud users.
// Blocks until ctx is cancelled.
func (c *Client) Run(ctx context.Context) {
	// Wait for signal to start
	select {
	case <-ctx.Done():
		return
	case <-c.startCh:
	}

	backoff := initialBackoff

	backoffTimer := time.NewTimer(0)
	backoffTimer.Stop()
	defer backoffTimer.Stop()

	for {
		apiKey := c.apiKeyFunc()
		if apiKey == "" {
			// Cloud dispatcher was removed — go back to waiting for signal
			select {
			case <-ctx.Done():
				return
			case <-c.startCh:
			}
			continue
		}

		wasConnected, err := c.connect(ctx, apiKey)
		if ctx.Err() != nil {
			return
		}

		if wasConnected {
			backoff = initialBackoff
		}

		if err != nil {
			if isPermissionDenied(err) {
				log.Debug().Msg("cloud account does not have pro plan, stopping cloud client")
				return
			}
			if isUnauthenticated(err) {
				log.Warn().Err(err).Dur("pause", unauthenticatedPause).Msg("invalid API key, pausing before retry")
				backoffTimer.Reset(unauthenticatedPause)
				select {
				case <-ctx.Done():
					return
				case <-backoffTimer.C:
				}
				backoff = initialBackoff
				continue
			}
			log.Warn().Err(err).Dur("backoff", backoff).Msg("cloud connection failed, reconnecting")
		}

		jitter := time.Duration(float64(backoff) * jitterFraction * rand.Float64())
		backoffTimer.Reset(backoff + jitter)
		select {
		case <-ctx.Done():
			return
		case <-backoffTimer.C:
		}

		backoff = min(backoff*backoffFactor, maxBackoff)
	}
}

// connect establishes a gRPC stream to the cloud and processes requests.
// Returns wasConnected=true if the stream was successfully established and
// at least one message was received before disconnecting.
func (c *Client) connect(ctx context.Context, apiKey string) (wasConnected bool, err error) {
	var creds grpc.DialOption
	if c.plaintext {
		creds = grpc.WithTransportCredentials(insecure.NewCredentials())
	} else {
		creds = grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, ""))
	}

	conn, err := grpc.NewClient(c.target, creds, grpc.WithUserAgent(dispatcher.UserAgent),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
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

	streamLifetime, streamCancel := context.WithCancel(stream.Context())
	var sendMu sync.Mutex
	var wg sync.WaitGroup

	sendResp := func(resp *pb.ToolResponse) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		return stream.Send(resp)
	}

	defer func() {
		// Cancel all active log streams before shutting down
		c.activeStreams.Range(func(key, value any) bool {
			if cancel, ok := value.(context.CancelFunc); ok {
				cancel()
			}
			c.activeStreams.Delete(key)
			return true
		})
		streamCancel()
		wg.Wait()
	}()

	for {
		req, err := stream.Recv()
		if err != nil {
			return wasConnected, fmt.Errorf("stream recv error: %w", err)
		}
		wasConnected = true

		// Handle cancel stream request
		if cancelReq, ok := req.Type.(*pb.ToolRequest_CancelStream); ok {
			streamReqID := cancelReq.CancelStream.StreamRequestId
			if cancel, loaded := c.activeStreams.LoadAndDelete(streamReqID); loaded {
				if cancelFn, ok := cancel.(context.CancelFunc); ok {
					cancelFn()
				}
			}
			continue
		}

		// Handle stream_logs specially — long-lived, not subject to the semaphore
		if callReq, ok := req.Type.(*pb.ToolRequest_CallTool); ok && callReq.CallTool.Name == "stream_logs" {
			streamCtx, streamCancel := context.WithCancel(streamLifetime)
			c.activeStreams.Store(req.RequestId, streamCancel)
			reqID := req.RequestId
			argsJSON := callReq.CallTool.ArgumentsJson
			wg.Go(func() {
				defer func() {
					c.activeStreams.Delete(reqID)
					streamCancel()
				}()
				log.Debug().Str("request_id", reqID).Msg("starting stream_logs")
				if err := executeStreamLogs(streamCtx, reqID, argsJSON, c.hostService, c.labels, sendResp); err != nil {
					if streamLifetime.Err() != nil {
						return
					}
					log.Debug().Err(err).Str("request_id", reqID).Msg("stream_logs ended")
				}
			})
			continue
		}

		// List tools is fast — handle inline. Tool calls run concurrently with a semaphore.
		if _, ok := req.Type.(*pb.ToolRequest_CallTool); ok {
			if !c.toolSem.TryAcquire(1) {
				resp := &pb.ToolResponse{
					RequestId: req.RequestId,
					Type: &pb.ToolResponse_CallTool{
						CallTool: &pb.CallToolResponse{
							Success: false,
							Error:   "too many concurrent tool calls",
						},
					},
				}
				if err := sendResp(resp); err != nil {
					return wasConnected, fmt.Errorf("stream send error: %w", err)
				}
				continue
			}
			wg.Go(func() {
				defer c.toolSem.Release(1)
				resp := c.handleRequest(streamLifetime, req)
				if streamLifetime.Err() != nil {
					return
				}
				if err := sendResp(resp); err != nil {
					log.Debug().Err(err).Msg("failed to send tool response")
				}
			})
		} else {
			resp := c.handleRequest(streamLifetime, req)
			if err := sendResp(resp); err != nil {
				return wasConnected, fmt.Errorf("stream send error: %w", err)
			}
		}
	}
}

func (c *Client) handleRequest(ctx context.Context, req *pb.ToolRequest) *pb.ToolResponse {
	resp := &pb.ToolResponse{
		RequestId: req.RequestId,
	}

	switch t := req.Type.(type) {
	case *pb.ToolRequest_ListTools:
		log.Debug().Str("request_id", req.RequestId).Msg("cloud requested tool list")
		resp.Type = &pb.ToolResponse_ListTools{
			ListTools: &pb.ListToolsResponse{
				Tools:   c.tools(),
				Version: cli.Version,
			},
		}

	case *pb.ToolRequest_CallTool:
		log.Debug().Str("request_id", req.RequestId).Str("tool", t.CallTool.Name).Str("args", t.CallTool.ArgumentsJson).Msg("cloud tool call received")
		callResp := ExecuteTool(ctx, t.CallTool.Name, t.CallTool.ArgumentsJson, c.enableActions, c.hostService, c.labels)
		if !callResp.Success {
			log.Debug().Str("error", callResp.Error).Str("request_id", req.RequestId).Str("tool", t.CallTool.Name).Msg("cloud tool call failed")
		} else {
			log.Debug().Str("request_id", req.RequestId).Str("tool", t.CallTool.Name).Msg("cloud tool call completed")
		}
		resp.Type = &pb.ToolResponse_CallTool{
			CallTool: callResp,
		}

	default:
		log.Warn().Msg("received unknown tool request type")
		resp.Type = &pb.ToolResponse_CallTool{
			CallTool: &pb.CallToolResponse{
				Success: false,
				Error:   "unknown request type",
			},
		}
	}

	return resp
}

func (c *Client) tools() []*pb.ToolDefinition {
	c.toolsOnce.Do(func() {
		c.cachedTools = AvailableTools(c.enableActions)
	})
	return c.cachedTools
}

func isPermissionDenied(err error) bool {
	return hasGRPCCode(err, codes.PermissionDenied)
}

func isUnauthenticated(err error) bool {
	return hasGRPCCode(err, codes.Unauthenticated)
}

func hasGRPCCode(err error, code codes.Code) bool {
	for e := err; e != nil; e = errors.Unwrap(e) {
		if s, ok := status.FromError(e); ok && s.Code() == code {
			return true
		}
	}
	return false
}
