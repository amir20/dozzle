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
