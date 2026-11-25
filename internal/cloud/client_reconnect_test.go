package cloud

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// fakeCloud sends ListTools on every stream, then either holds it open or drops it.
type fakeCloud struct {
	pb.UnimplementedCloudToolServiceServer
	hold  bool
	mu    sync.Mutex
	dials []time.Time
}

func (f *fakeCloud) ToolStream(stream grpc.BidiStreamingServer[pb.ToolResponse, pb.ToolRequest]) error {
	f.mu.Lock()
	f.dials = append(f.dials, time.Now())
	f.mu.Unlock()
	if err := stream.Send(&pb.ToolRequest{RequestId: "1", Type: &pb.ToolRequest_ListTools{ListTools: &pb.ListToolsRequest{}}}); err != nil {
		return err
	}
	if f.hold {
		<-stream.Context().Done()
	} else {
		_, _ = stream.Recv() // the ListTools reply, so the client counts as connected
	}
	return nil
}

func (f *fakeCloud) dialTimes() []time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]time.Time(nil), f.dials...)
}

func startFakeCloud(t *testing.T, f *fakeCloud, streamLogs *atomic.Bool) *Client {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	pb.RegisterCloudToolServiceServer(srv, f)
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)

	t.Setenv("AGENT_URL", "http://"+lis.Addr().String())
	c := NewClient(func() string { return "key" }, "test", "v0", ToolDeps{})
	c.SetDeployment(deploymentAgent, "")
	c.SetStreamLogsFunc(streamLogs.Load)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go c.Run(ctx)
	c.Notify()
	return c
}

// An agent is re-sent the hub's cloud config whenever any host joins, so Refresh
// must leave an unchanged connection alone and drop one whose settings changed:
// otherwise turning log streaming off on the hub never reached a connected agent.
func TestRefreshRedialsOnlyWhenSettingsChange(t *testing.T) {
	f := &fakeCloud{hold: true}
	var streamLogs atomic.Bool
	streamLogs.Store(true)
	c := startFakeCloud(t, f, &streamLogs)

	require.Eventually(t, func() bool { return len(f.dialTimes()) == 1 }, 5*time.Second, 10*time.Millisecond)

	c.Refresh()
	time.Sleep(300 * time.Millisecond)
	assert.Len(t, f.dialTimes(), 1, "unchanged settings must keep the connection")

	streamLogs.Store(false)
	c.Refresh()
	require.Eventually(t, func() bool { return len(f.dialTimes()) == 2 }, 5*time.Second, 10*time.Millisecond)
}

// A cloud that accepts a stream, sends ListTools and drops it must not be
// redialed every second: backoff only resets after a stable connection.
func TestBackoffGrowsWhenStreamsDropQuickly(t *testing.T) {
	f := &fakeCloud{}
	var streamLogs atomic.Bool
	startFakeCloud(t, f, &streamLogs)

	require.Eventually(t, func() bool { return len(f.dialTimes()) >= 3 }, 10*time.Second, 10*time.Millisecond)
	d := f.dialTimes()
	assert.GreaterOrEqual(t, d[2].Sub(d[1]), 1800*time.Millisecond, "second retry should wait ~2s, not reset to 1s")
}
