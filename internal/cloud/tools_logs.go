package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/amir20/dozzle/internal/container"
	pb "github.com/amir20/dozzle/proto/cloud"
)

type fetchLogsArgs struct {
	ContainerID string `json:"container_id"`
	Host        string `json:"host_id"`
	Start       string `json:"start"`
	End         string `json:"end"`
	Level       string `json:"level"`
	Query       string `json:"query"`
	Regex       string `json:"regex"`
	Inverse     bool   `json:"inverse"`
}

func executeFetchContainerLogs(ctx context.Context, argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	var args fetchLogsArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}
	if args.ContainerID == "" || args.Host == "" {
		return nil, fmt.Errorf("container_id and host_id are required")
	}

	cs, err := deps.HostService.FindContainer(args.Host, args.ContainerID, deps.Labels)
	if err != nil {
		return nil, fmt.Errorf("container not found: %w", err)
	}

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()
	if args.Start != "" {
		t, err := time.Parse(time.RFC3339, args.Start)
		if err != nil {
			return nil, fmt.Errorf("invalid start time format (expected RFC3339): %w", err)
		}
		start = t
	}
	if args.End != "" {
		t, err := time.Parse(time.RFC3339, args.End)
		if err != nil {
			return nil, fmt.Errorf("invalid end time format (expected RFC3339): %w", err)
		}
		end = t
	}

	var re *regexp.Regexp
	if args.Regex != "" {
		var err error
		re, err = regexp.Compile(args.Regex)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logCh, err := cs.LogsBetweenDates(ctx, start, end, container.STDOUT|container.STDERR)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch logs: %w", err)
	}

	const maxLines = 100
	entries := make([]*pb.LogEntry, 0, maxLines)
	for event := range logCh {
		msg, matches := matchesFilters(event, &args, re)
		if !matches {
			continue
		}

		entries = append(entries, &pb.LogEntry{
			Timestamp: event.Timestamp,
			Message:   msg,
			Stream:    event.Stream,
			Level:     event.Level,
		})

		if len(entries) >= maxLines {
			break
		}
	}

	return &pb.CallToolResponse{
		Success: true,
		Result:  &pb.CallToolResponse_FetchLogs{FetchLogs: &pb.FetchLogsResult{ContainerName: cs.Container.Name, Entries: entries}},
	}, nil
}
