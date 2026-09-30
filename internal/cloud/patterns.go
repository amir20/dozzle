package cloud

import (
	"context"
	"fmt"

	pb "github.com/amir20/dozzle/proto/cloud"
	"google.golang.org/grpc/metadata"
)

// PatternLine names one log line the viewer is showing.
type PatternLine struct {
	ContainerID string `json:"containerId"`
	LogID       uint32 `json:"logId"`
}

// PatternContext is what Cloud's error memory knows about one pattern among
// the lines asked about. Status is "new", "louder", "known" or "learning".
type PatternContext struct {
	ContainerID      string   `json:"containerId"`
	LogIDs           []uint32 `json:"logIds"`
	Pattern          string   `json:"pattern"`
	Level            string   `json:"level"`
	Status           string   `json:"status"`
	FirstSeenNs      int64    `json:"firstSeen,omitempty"`
	DaysSeen         int32    `json:"daysSeen,omitempty"`
	RatePerHour      float64  `json:"ratePerHour"`
	UsualRatePerHour float64  `json:"usualRatePerHour"`
}

// GetPatternContext asks Cloud's error memory about the lines on screen:
// whether each line's pattern is new for its container, louder than usual,
// or known. Cloud finds the lines by log_id in what it was streamed, so this
// needs the streamLogs opt-in. Identity comes from the connection metadata.
func (c *Client) GetPatternContext(ctx context.Context, lines []PatternLine, fromNs, toNs int64) ([]PatternContext, error) {
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

	refs := make([]*pb.PatternLineRef, len(lines))
	for i, l := range lines {
		refs[i] = &pb.PatternLineRef{ContainerId: l.ContainerID, LogId: l.LogID}
	}
	resp, err := client.GetPatternContext(callCtx, &pb.GetPatternContextRequest{Lines: refs, FromTsNs: fromNs, ToTsNs: toNs})
	if err != nil {
		return nil, fmt.Errorf("cloud: pattern context: %w", err)
	}

	out := make([]PatternContext, 0, len(resp.GetPatterns()))
	for _, p := range resp.GetPatterns() {
		out = append(out, PatternContext{
			ContainerID:      p.GetContainerId(),
			LogIDs:           p.GetLogIds(),
			Pattern:          p.GetPattern(),
			Level:            p.GetLevel(),
			Status:           patternStatus(p.GetStatus()),
			FirstSeenNs:      p.GetFirstSeenNs(),
			DaysSeen:         p.GetDaysSeen(),
			RatePerHour:      p.GetRatePerHour(),
			UsualRatePerHour: p.GetUsualRatePerHour(),
		})
	}
	return out, nil
}

func patternStatus(s pb.PatternStatus) string {
	switch s {
	case pb.PatternStatus_PATTERN_STATUS_NEW:
		return "new"
	case pb.PatternStatus_PATTERN_STATUS_LOUDER:
		return "louder"
	case pb.PatternStatus_PATTERN_STATUS_LEARNING:
		return "learning"
	default:
		return "known"
	}
}
