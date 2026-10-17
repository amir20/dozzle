package cloud

import (
	"context"
	"sync"
	"time"

	pb "github.com/amir20/dozzle/proto/cloud"
	"google.golang.org/protobuf/proto"
)

const (
	// inspectBatchDeadline bounds one inspect_containers pass. Most containers
	// are already in the store, so this only matters when many have to be
	// read from the engine; whatever is not reached in time is left out.
	inspectBatchDeadline = 45 * time.Second
	// inspectBatchWorkers reads this many containers from the engine at once.
	// It is local work, not a cloud round trip, so it can go wider than the
	// cloud's own fan-out did.
	inspectBatchWorkers = 8
	// inspectBatchBytes keeps the response well under gRPC's 4 MiB frame.
	inspectBatchBytes = 3 * 1024 * 1024
)

// executeInspectContainers runs inspect_container for every container the
// principal can see and answers in one response. A container that cannot be
// found or is not reached before the deadline is left out, never guessed.
func executeInspectContainers(ctx context.Context, deps ToolDeps) (*pb.CallToolResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, inspectBatchDeadline)
	defer cancel()

	scoped := deps.scoped()
	list, errs := scoped.ListAllContainers()
	logHostErrors(errs)
	hostNames := buildHostNameMap(deps.HostService)

	var (
		mu     sync.Mutex
		out    = make([]*pb.InspectContainerResult, len(list))
		closed bool
		wg     sync.WaitGroup
		jobs   = make(chan int)
	)
	for range inspectBatchWorkers {
		wg.Go(func() {
			for i := range jobs {
				cs, err := scoped.FindContainer(list[i].Host, list[i].ID)
				if err != nil {
					continue
				}
				r := inspectResult(cs.Container, hostNames)
				mu.Lock()
				if !closed {
					out[i] = r
				}
				mu.Unlock()
			}
		})
	}
	go func() {
		defer close(jobs)
		for i := range list {
			select {
			case jobs <- i:
			case <-ctx.Done():
				return
			}
		}
	}()

	// A slow engine read must not hold the response past the deadline, so
	// stragglers are abandoned rather than waited for.
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
	}
	mu.Lock()
	closed = true
	results := out
	mu.Unlock()

	res := &pb.InspectContainersResult{DeadlineExceeded: ctx.Err() != nil}
	size := 0
	for _, r := range results {
		if r == nil {
			continue
		}
		if size += proto.Size(r); size > inspectBatchBytes {
			res.Truncated = true
			break
		}
		res.Containers = append(res.Containers, r)
	}
	return &pb.CallToolResponse{
		Success: true,
		Result:  &pb.CallToolResponse_InspectContainers{InspectContainers: res},
	}, nil
}
