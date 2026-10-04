package cloud

import (
	"testing"

	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/stretchr/testify/assert"
)

// The URL ends up in an href in the viewer, so only http(s) links survive.
func Test_alertResultFromProto_drops_non_http_urls(t *testing.T) {
	resp := &pb.GetAlertsResponse{Hits: []*pb.AlertHit{
		{AlertId: "a", Url: "https://cloud.dozzle.dev/alerts/a"},
		{AlertId: "b", Url: "javascript:alert(1)"},
		{AlertId: "c", Url: "JavaScript:alert(1)"},
		{AlertId: "d", Url: "data:text/html,<script>alert(1)</script>"},
		{AlertId: "e", Url: "//evil.example/a"},
	}}

	result := alertResultFromProto(resp)

	urls := make([]string, 0, len(result.Hits))
	for _, h := range result.Hits {
		urls = append(urls, h.URL)
	}
	assert.Equal(t, []string{"https://cloud.dozzle.dev/alerts/a", "", "", "", ""}, urls)
}

// A deploy event carries Cloud's verdict on the update that created the
// container; every other event carries none.
func Test_alertResultFromProto_maps_deploy_verdicts(t *testing.T) {
	resp := &pb.GetAlertsResponse{Events: []*pb.EventHit{
		{TsNs: 1, ContainerId: "abc123def456", HostId: "h1", Type: "deploy", Deploy: &pb.DeployHit{
			DeployId:   "x7",
			Container:  "immich",
			FromRef:    "ghcr.io/immich-app/immich-server:1.4.1",
			ToRef:      "ghcr.io/immich-app/immich-server:1.4.2",
			FromDigest: "sha256:aaa",
			ToDigest:   "sha256:bbb",
			Verdict:    "regressed",
			Reason:     "3 errors that never appeared before the update",
			Decision:   "kept",
			Url:        "javascript:alert(1)",
		}},
		{TsNs: 2, ContainerId: "abc123def456", Type: "log", LogId: 9},
	}}

	result := alertResultFromProto(resp)

	assert.Equal(t, &DeployHit{
		DeployID:   "x7",
		Container:  "immich",
		FromRef:    "ghcr.io/immich-app/immich-server:1.4.1",
		ToRef:      "ghcr.io/immich-app/immich-server:1.4.2",
		FromDigest: "sha256:aaa",
		ToDigest:   "sha256:bbb",
		Verdict:    "regressed",
		Reason:     "3 errors that never appeared before the update",
		Decision:   "kept",
	}, result.Events[0].Deploy)
	assert.Equal(t, "deploy", result.Events[0].Type)
	assert.Nil(t, result.Events[1].Deploy)
}
