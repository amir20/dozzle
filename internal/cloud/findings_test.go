package cloud

import (
	"context"
	"errors"
	"testing"

	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeFindingsClient struct {
	pb.CloudToolServiceClient
	getReq    *pb.GetFindingsRequest
	ignoreReq *pb.IgnoreFindingRequest
	md        metadata.MD
	err       error
}

func (f *fakeFindingsClient) GetFindings(ctx context.Context, req *pb.GetFindingsRequest, _ ...grpc.CallOption) (*pb.GetFindingsResponse, error) {
	f.getReq = req
	f.md, _ = metadata.FromOutgoingContext(ctx)
	if f.err != nil {
		return nil, f.err
	}
	return &pb.GetFindingsResponse{
		Findings: []*pb.Finding{{
			FindingId:      "k3Xa9",
			Severity:       "high",
			Status:         "ongoing",
			Headline:       "worker OOM-killed every night",
			Why:            "why",
			Fix:            "fix",
			ContainerNames: []string{"app-worker-1"},
			DaysOpen:       4,
			FirstSeenAtNs:  1,
			LastSeenAtNs:   2,
			LogFirstAtNs:   3,
			LogLastAtNs:    4,
			Url:            "https://cloud.dozzle.dev/findings/k3Xa9",
		}, {
			FindingId: "evil",
			Locked:    true,
			Url:       "javascript:alert(1)",
		}},
		FixLocked: true,
		Truncated: true,
	}, nil
}

func (f *fakeFindingsClient) IgnoreFinding(_ context.Context, req *pb.IgnoreFindingRequest, _ ...grpc.CallOption) (*pb.IgnoreFindingResponse, error) {
	f.ignoreReq = req
	if f.err != nil {
		return nil, f.err
	}
	return &pb.IgnoreFindingResponse{Finding: &pb.Finding{FindingId: req.GetFindingId(), IgnoredAtNs: 9}}, nil
}

func findingsClient(fake *fakeFindingsClient, key string) *Client {
	return &Client{apiKeyFunc: func() string { return key }, instanceID: "inst", unaryClient: fake}
}

func TestGetFindings(t *testing.T) {
	fake := &fakeFindingsClient{}
	result, err := findingsClient(fake, "key").GetFindings(t.Context(), []string{"app-worker-1"}, "open", false, 20)
	require.NoError(t, err)

	assert.Equal(t, []string{"app-worker-1"}, fake.getReq.GetContainerNames())
	assert.Equal(t, "open", fake.getReq.GetStatus())
	assert.Equal(t, int32(20), fake.getReq.GetLimit())
	assert.Equal(t, []string{"key"}, fake.md.Get("x-api-key"))
	assert.Equal(t, []string{"inst"}, fake.md.Get("x-instance-id"))

	assert.True(t, result.FixLocked)
	assert.True(t, result.Truncated)
	require.Len(t, result.Findings, 2)
	assert.Equal(t, Finding{
		ID:             "k3Xa9",
		Severity:       "high",
		Status:         "ongoing",
		Headline:       "worker OOM-killed every night",
		Why:            "why",
		Fix:            "fix",
		ContainerNames: []string{"app-worker-1"},
		Containers:     []FindingContainer{},
		DaysOpen:       4,
		FirstSeenAt:    1,
		LastSeenAt:     2,
		LogFirstAt:     3,
		LogLastAt:      4,
		URL:            "https://cloud.dozzle.dev/findings/k3Xa9",
	}, result.Findings[0])
	assert.True(t, result.Findings[1].Locked)
	assert.Empty(t, result.Findings[1].URL, "the url ends up in an href, so only http(s) survives")
}

func TestIgnoreFinding(t *testing.T) {
	fake := &fakeFindingsClient{}
	f, err := findingsClient(fake, "key").IgnoreFinding(t.Context(), "k3Xa9", true)
	require.NoError(t, err)
	assert.Equal(t, "k3Xa9", fake.ignoreReq.GetFindingId())
	assert.True(t, fake.ignoreReq.GetUndo())
	assert.Equal(t, int64(9), f.IgnoredAt)
}

func TestFindings_errors(t *testing.T) {
	_, err := findingsClient(&fakeFindingsClient{}, "").GetFindings(t.Context(), []string{"x"}, "", false, 1)
	assert.ErrorIs(t, err, ErrNotConfigured)
	_, err = findingsClient(&fakeFindingsClient{}, "").IgnoreFinding(t.Context(), "x", false)
	assert.ErrorIs(t, err, ErrNotConfigured)

	// The web layer branches on the gRPC code, so wrapping must keep it.
	fake := &fakeFindingsClient{err: status.Error(codes.NotFound, "no")}
	_, err = findingsClient(fake, "key").IgnoreFinding(t.Context(), "x", false)
	assert.Equal(t, codes.NotFound, status.Code(err))
	_, err = findingsClient(fake, "key").GetFindings(t.Context(), []string{"x"}, "", false, 1)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.False(t, errors.Is(err, ErrNotConfigured))
}
