package cloud

import (
	"context"
	"fmt"

	pb "github.com/amir20/dozzle/proto/cloud"
	"google.golang.org/grpc/metadata"
)

// AlertResult is the JSON-friendly response shape returned to the Dozzle web
// layer. Mirrors the proto GetAlertsResponse but lives in this package so
// callers don't have to import the proto package directly.
type AlertResult struct {
	Hits []AlertHit `json:"hits"`
	// Truncated means the window held more anchors than the limit allowed, so
	// the viewer is not seeing all of them.
	Truncated bool `json:"truncated,omitempty"`
}

// AlertHit is one alert anchored inside the requested window, scoped
// server-side to the connecting instance's (user_id, api_key_id) — Cloud
// derives those from the auth metadata, never the request body.
type AlertHit struct {
	AlertID     int64  `json:"alertId"`
	ContainerID string `json:"containerId"`
	HostID      string `json:"hostId"`
	// LogID is Cloud's copy of Dozzle's FNV-32a hash of the line that
	// triggered the alert — the same id LogEvent.Id carries, so the viewer can
	// splice the alert in directly after that line. Omitted (0) for metric and
	// event alerts, which have no triggering line and anchor on Ts instead.
	LogID uint32 `json:"logId,omitempty"`
	Ts    int64  `json:"ts"`

	Headline        string `json:"headline"`
	Level           string `json:"level"`
	EventCount      int32  `json:"eventCount"`
	SuppressedCount int32  `json:"suppressedCount,omitempty"`
	// ContainerCount > 1 means the incident is wider than the container being
	// viewed.
	ContainerCount int32 `json:"containerCount,omitempty"`
	// Summary is the alert's description, shown in Details. Present even when
	// triage wrote no investigation.
	Summary       string `json:"summary,omitempty"`
	Investigation string `json:"investigation,omitempty"`
	TriageAction  string `json:"triageAction,omitempty"`

	CreatedAt      int64 `json:"createdAt"`
	LastActivityAt int64 `json:"lastActivityAt,omitempty"`
	// IsOrigin marks the anchor where the incident FIRST fired. Cloud appends
	// every folded batch's events to the original alert, so one incident can
	// have activity in many scroll windows; the viewer renders a full card at
	// the origin and a compact "still happening" marker at follow-ups.
	IsOrigin bool `json:"isOrigin"`

	// URL deep-links to the alert page in Dozzle Cloud. Empty when the API key
	// has no app_url configured.
	URL string `json:"url,omitempty"`
}

// GetAlerts fetches the alerts that fired on these containers inside a window,
// for merging into the local log stream on scrollback. Reuses the long-lived
// unary conn so a scroll doesn't pay a TLS handshake.
//
// Identity (user, instance) is enforced server-side from the authenticated
// metadata; this client passes only the per-request fields below. Unlike
// SearchLogs this does not depend on the streamLogs opt-in — alerts live in
// Cloud's own database rather than the log store.
func (c *Client) GetAlerts(ctx context.Context, containerIDs []string, hostID string, fromNs, toNs int64, limit int32, includeFollowUps bool) (*AlertResult, error) {
	apiKey := c.apiKeyFunc()
	if apiKey == "" {
		return nil, ErrNotConfigured
	}

	client, err := c.unaryServiceClient()
	if err != nil {
		return nil, err
	}

	mdPairs := []string{"x-api-key", apiKey}
	if c.instanceID != "" {
		mdPairs = append(mdPairs, "x-instance-id", c.instanceID)
	}
	callCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(mdPairs...))

	resp, err := client.GetAlerts(callCtx, &pb.GetAlertsRequest{
		ContainerIds:     containerIDs,
		HostId:           hostID,
		FromTsNs:         fromNs,
		ToTsNs:           toNs,
		Limit:            limit,
		IncludeFollowUps: includeFollowUps,
	})
	if err != nil {
		return nil, fmt.Errorf("cloud: alerts: %w", err)
	}

	hits := make([]AlertHit, 0, len(resp.GetHits()))
	for _, h := range resp.GetHits() {
		hits = append(hits, AlertHit{
			AlertID:         h.GetAlertId(),
			ContainerID:     h.GetContainerId(),
			HostID:          h.GetHostId(),
			LogID:           h.GetLogId(),
			Ts:              h.GetAnchorTsNs(),
			Headline:        h.GetHeadline(),
			Level:           h.GetLevel(),
			EventCount:      h.GetEventCount(),
			SuppressedCount: h.GetSuppressedCount(),
			ContainerCount:  h.GetContainerCount(),
			Summary:         h.GetSummary(),
			Investigation:   h.GetInvestigation(),
			TriageAction:    h.GetTriageAction(),
			CreatedAt:       h.GetCreatedAtNs(),
			LastActivityAt:  h.GetLastActivityAtNs(),
			IsOrigin:        h.GetIsOrigin(),
			URL:             h.GetUrl(),
		})
	}
	return &AlertResult{Hits: hits, Truncated: resp.GetTruncated()}, nil
}
