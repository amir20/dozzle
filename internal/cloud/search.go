package cloud

import (
	"context"
	"errors"
	"fmt"
	"time"

	pb "github.com/amir20/dozzle/proto/cloud"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// SearchLogResult is the JSON-friendly response shape returned to the Dozzle
// web layer. Mirrors the proto SearchLogsResponse but lives in this package
// so callers don't have to import the proto package directly.
type SearchLogResult struct {
	Hits    []SearchLogHit `json:"hits"`
	HasMore bool           `json:"hasMore"`
}

// SearchLogHit is one matched log line, scoped server-side to the connecting
// instance's (user_id, api_key_id) — Cloud derives those from the auth
// metadata, never the request body.
type SearchLogHit struct {
	TimestampNs   int64  `json:"ts"`
	HostID        string `json:"hostId"`
	ContainerID   string `json:"containerId"`
	ContainerName string `json:"containerName"`
	Message       string `json:"message"`
	Stream        string `json:"stream"`
	Level         string `json:"level"`
}

// ErrNotConfigured is returned when SearchLogs is called but no Cloud API key
// is available (the user hasn't linked Cloud yet). Callers map this to a 503.
var ErrNotConfigured = errors.New("cloud: no API key configured")

// SearchLogs runs a Cloud-side log search against the existing gRPC service.
// Opens a fresh, short-lived connection per call — search is rare (debounced
// UI input) so the simplicity outweighs reusing the long-running ToolStream
// connection. Identity (user, instance) is enforced server-side from the
// authenticated metadata; this client passes none of those fields itself.
func (c *Client) SearchLogs(ctx context.Context, query string, limit int32, hostID, containerID string) (*SearchLogResult, error) {
	apiKey := c.apiKeyFunc()
	if apiKey == "" {
		return nil, ErrNotConfigured
	}

	var creds grpc.DialOption
	if c.plaintext {
		creds = grpc.WithTransportCredentials(insecure.NewCredentials())
	} else {
		creds = grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, ""))
	}

	conn, err := grpc.NewClient(c.target, creds,
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("cloud: dial: %w", err)
	}
	defer conn.Close()

	mdPairs := []string{"x-api-key", apiKey}
	if c.instanceID != "" {
		mdPairs = append(mdPairs, "x-instance-id", c.instanceID)
	}
	callCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(mdPairs...))

	resp, err := pb.NewCloudToolServiceClient(conn).SearchLogs(callCtx, &pb.SearchLogsRequest{
		Query:       query,
		Limit:       limit,
		HostId:      hostID,
		ContainerId: containerID,
	})
	if err != nil {
		return nil, fmt.Errorf("cloud: search: %w", err)
	}

	hits := make([]SearchLogHit, 0, len(resp.GetHits()))
	for _, h := range resp.GetHits() {
		hits = append(hits, SearchLogHit{
			TimestampNs:   h.GetTimestampNs(),
			HostID:        h.GetHostId(),
			ContainerID:   h.GetContainerId(),
			ContainerName: h.GetContainerName(),
			Message:       h.GetMessage(),
			Stream:        h.GetStream(),
			Level:         h.GetLevel(),
		})
	}
	return &SearchLogResult{Hits: hits, HasMore: resp.GetHasMore()}, nil
}
