package cloud

import (
	"context"
	"errors"
	"fmt"

	pb "github.com/amir20/dozzle/proto/cloud"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// SearchLogResult is the JSON-friendly response shape returned to the Dozzle
// web layer. Mirrors the proto SearchLogsResponse but lives in this package
// so callers don't have to import the proto package directly.
type SearchLogResult struct {
	Hits    []SearchLogHit `json:"hits"`
	HasMore bool           `json:"hasMore"`
	// NextBefore is the cursor to pass back as `before` (HTTP) /
	// before_ts_ns (gRPC) to fetch the next older page. 0 when HasMore
	// is false.
	NextBefore int64 `json:"nextBefore,omitempty"`
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
	// LogID is Dozzle's FNV-32a hash of the original line. Lets the UI
	// build deep-links matching "Copy permalink" output. Omitted when the
	// row predates indexing (older Dozzle clients sent 0).
	LogID uint32 `json:"logId,omitempty"`
}

// ErrNotConfigured is returned when SearchLogs is called but no Cloud API key
// is available (the user hasn't linked Cloud yet). Callers map this to a 503.
var ErrNotConfigured = errors.New("cloud: no API key configured")

// searchServiceClient returns a (lazily dialed) reusable gRPC client. The
// underlying conn is shared across all SearchLogs calls so we pay the TLS
// handshake once per process — not once per keystroke.
func (c *Client) searchServiceClient() (pb.CloudToolServiceClient, error) {
	c.searchConnMu.Lock()
	defer c.searchConnMu.Unlock()
	if c.searchClient != nil {
		return c.searchClient, nil
	}
	var creds grpc.DialOption
	if c.plaintext {
		creds = grpc.WithTransportCredentials(insecure.NewCredentials())
	} else {
		creds = grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, ""))
	}
	conn, err := grpc.NewClient(c.target, creds)
	if err != nil {
		return nil, fmt.Errorf("cloud: dial: %w", err)
	}
	c.searchConn = conn
	c.searchClient = pb.NewCloudToolServiceClient(conn)
	return c.searchClient, nil
}

// SearchLogs runs a Cloud-side log search against the existing gRPC service.
// Reuses a long-lived gRPC conn (lazily dialed on first call) so the
// 500ms search timeout isn't burned on a TLS handshake per keystroke.
// Identity (user, instance) is enforced server-side from the authenticated
// metadata; this client passes only the per-request fields below.
func (c *Client) SearchLogs(ctx context.Context, query string, limit int32, hostID, containerID string, before int64) (*SearchLogResult, error) {
	apiKey := c.apiKeyFunc()
	if apiKey == "" {
		return nil, ErrNotConfigured
	}

	client, err := c.searchServiceClient()
	if err != nil {
		return nil, err
	}

	mdPairs := []string{"x-api-key", apiKey}
	if c.instanceID != "" {
		mdPairs = append(mdPairs, "x-instance-id", c.instanceID)
	}
	callCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(mdPairs...))

	resp, err := client.SearchLogs(callCtx, &pb.SearchLogsRequest{
		Query:       query,
		Limit:       limit,
		HostId:      hostID,
		ContainerId: containerID,
		BeforeTsNs:  before,
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
			LogID:         h.GetLogId(),
		})
	}
	return &SearchLogResult{
		Hits:       hits,
		HasMore:    resp.GetHasMore(),
		NextBefore: resp.GetNextBeforeTsNs(),
	}, nil
}
