package cloud

import (
	"context"
	"fmt"

	pb "github.com/amir20/dozzle/proto/cloud"
	"google.golang.org/grpc/metadata"
)

// FindingsResult is what the findings list answers, in the web layer's shape.
type FindingsResult struct {
	Findings []Finding `json:"findings"`
	// FixLocked means the viewer's plan withholds the fix. Decided in Cloud;
	// never infer it from an empty Fix.
	FixLocked bool `json:"fixLocked"`
	Truncated bool `json:"truncated,omitempty"`
}

// FindingContainer is one container on screen that a finding is about.
type FindingContainer struct {
	Host string `json:"host"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Finding is one finding from Cloud's scheduled log review. Cloud records
// container names only, so ContainerNames is what it matched; the web layer
// maps those back to the containers it asked about and fills Containers.
type Finding struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Status   string `json:"status"`
	// Locked means the review has not shown it to the user yet. Headline, Why
	// and Fix are then empty.
	Locked   bool   `json:"locked"`
	Headline string `json:"headline"`
	Why      string `json:"why,omitempty"`
	Fix      string `json:"fix,omitempty"`

	ContainerNames []string           `json:"-"`
	Containers     []FindingContainer `json:"containers"`

	DaysOpen    int32 `json:"daysOpen"`
	FirstSeenAt int64 `json:"firstSeenAt"`
	LastSeenAt  int64 `json:"lastSeenAt"`
	ResolvedAt  int64 `json:"resolvedAt,omitempty"`
	// LogFirstAt and LogLastAt are the oldest and newest log line behind the
	// finding, unix nanoseconds. Zero when the review did not record them.
	LogFirstAt int64  `json:"logFirstAt,omitempty"`
	LogLastAt  int64  `json:"logLastAt,omitempty"`
	IgnoredAt  int64  `json:"ignoredAt,omitempty"`
	URL        string `json:"url,omitempty"`
}

// GetFindings asks Cloud what its scheduled review found about these
// containers, by name. status is "open" (open and ongoing), "" (every status)
// or "resolved"; ignored returns only ignored findings.
//
// The caller is responsible for sending only names the requesting user may see:
// Cloud scopes to the instance, not to a Dozzle user.
func (c *Client) GetFindings(ctx context.Context, containerNames []string, status string, ignored bool, limit int32) (*FindingsResult, error) {
	client, callCtx, err := c.findingsCall(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := client.GetFindings(callCtx, &pb.GetFindingsRequest{
		ContainerNames: containerNames,
		Status:         status,
		Ignored:        ignored,
		Limit:          limit,
	})
	if err != nil {
		return nil, fmt.Errorf("cloud: findings: %w", err)
	}

	findings := make([]Finding, 0, len(resp.GetFindings()))
	for _, f := range resp.GetFindings() {
		findings = append(findings, findingFromProto(f))
	}
	return &FindingsResult{Findings: findings, FixLocked: resp.GetFixLocked(), Truncated: resp.GetTruncated()}, nil
}

// IgnoreFinding ignores one finding account-wide, or takes that back with undo.
func (c *Client) IgnoreFinding(ctx context.Context, findingID string, undo bool) (*Finding, error) {
	client, callCtx, err := c.findingsCall(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := client.IgnoreFinding(callCtx, &pb.IgnoreFindingRequest{FindingId: findingID, Undo: undo})
	if err != nil {
		return nil, fmt.Errorf("cloud: ignore finding: %w", err)
	}

	f := findingFromProto(resp.GetFinding())
	return &f, nil
}

func (c *Client) findingsCall(ctx context.Context) (pb.CloudToolServiceClient, context.Context, error) {
	apiKey := c.apiKeyFunc()
	if apiKey == "" {
		return nil, nil, ErrNotConfigured
	}

	client, err := c.unaryServiceClient()
	if err != nil {
		return nil, nil, err
	}

	mdPairs := []string{"x-api-key", apiKey}
	if c.instanceID != "" {
		mdPairs = append(mdPairs, "x-instance-id", c.instanceID)
	}
	return client, metadata.NewOutgoingContext(ctx, metadata.Pairs(mdPairs...)), nil
}

func findingFromProto(f *pb.Finding) Finding {
	return Finding{
		ID:             f.GetFindingId(),
		Severity:       f.GetSeverity(),
		Status:         f.GetStatus(),
		Locked:         f.GetLocked(),
		Headline:       f.GetHeadline(),
		Why:            f.GetWhy(),
		Fix:            f.GetFix(),
		ContainerNames: f.GetContainerNames(),
		Containers:     []FindingContainer{},
		DaysOpen:       f.GetDaysOpen(),
		FirstSeenAt:    f.GetFirstSeenAtNs(),
		LastSeenAt:     f.GetLastSeenAtNs(),
		ResolvedAt:     f.GetResolvedAtNs(),
		LogFirstAt:     f.GetLogFirstAtNs(),
		LogLastAt:      f.GetLogLastAtNs(),
		IgnoredAt:      f.GetIgnoredAtNs(),
		URL:            httpURL(f.GetUrl()),
	}
}
