package cloud

import (
	"context"
	"fmt"
	"time"

	"github.com/amir20/dozzle/internal/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"google.golang.org/grpc/metadata"
)

// RecentDeploy is one update Dozzle Cloud recorded, with its verdict. The
// update is the record this instance pushed, so it reads like one Dozzle keeps
// itself, except that Cloud's digests are bare sha256:... without the repo.
type RecentDeploy struct {
	container.UpdateRecord
	Verdict *DeployHit `json:"verdict,omitempty"`
}

// GetRecentDeploys fetches the updates Dozzle Cloud recorded since sinceNs,
// newest first, for the megaphone. They outlive a restart, which Dozzle's own
// records do not.
//
// The caller is responsible for confining the result to what the requesting
// user may see — Cloud scopes to the instance, not to a Dozzle user.
func (c *Client) GetRecentDeploys(ctx context.Context, sinceNs int64, limit int32) ([]RecentDeploy, error) {
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

	resp, err := client.GetRecentDeploys(callCtx, &pb.GetRecentDeploysRequest{SinceTsNs: sinceNs, Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("cloud: recent deploys: %w", err)
	}

	deploys := make([]RecentDeploy, 0, len(resp.GetDeploys()))
	for _, d := range resp.GetDeploys() {
		if d.GetUpdate() == nil {
			continue
		}
		deploys = append(deploys, RecentDeploy{
			UpdateRecord: updateRecordFromProto(d.GetUpdate()),
			Verdict:      deployHitFromProto(d.GetVerdict()),
		})
	}
	return deploys, nil
}

// updateRecordFromProto is containerUpdateProto the other way round.
func updateRecordFromProto(u *pb.ContainerUpdate) container.UpdateRecord {
	return container.UpdateRecord{
		Host:         u.GetHost(),
		Name:         u.GetName(),
		OldID:        u.GetOldContainerId(),
		NewID:        u.GetNewContainerId(),
		ImageRef:     u.GetImageRef(),
		FromDigest:   u.GetFromDigest(),
		ToDigest:     u.GetToDigest(),
		FromImageID:  u.GetFromImageId(),
		ToImageID:    u.GetToImageId(),
		OldStartedAt: fromUnixNano(u.GetOldStartedAt()),
		At:           fromUnixNano(u.GetAt()),
		Source:       u.GetSource(),
		RolledBack:   u.GetRolledBack(),
	}
}

func fromUnixNano(ns int64) time.Time {
	if ns == 0 {
		return time.Time{}
	}
	return time.Unix(0, ns).UTC()
}
