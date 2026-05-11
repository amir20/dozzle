package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/rs/zerolog/log"
)

// streamSender is a function that sends a ToolResponse to the cloud.
type streamSender func(resp *pb.ToolResponse) error

func parseStreamArgs(argsJSON string) (*fetchLogsArgs, *regexp.Regexp, error) {
	var args fetchLogsArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, nil, fmt.Errorf("failed to parse arguments: %w", err)
	}
	if args.ContainerID == "" || args.Host == "" {
		return nil, nil, fmt.Errorf("container_id and host_id are required")
	}

	var re *regexp.Regexp
	if args.Regex != "" {
		var err error
		re, err = regexp.Compile(args.Regex)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}
	return &args, re, nil
}

func matchesFilters(event *container.LogEvent, args *fetchLogsArgs, re *regexp.Regexp) (string, bool) {
	if args.Level != "" && !strings.EqualFold(event.Level, args.Level) {
		return "", false
	}

	msg := event.RawMessage
	if msg == "" {
		msg = fmt.Sprintf("%v", event.Message)
	}

	if args.Query != "" {
		matched := containsIgnoreCase(msg, args.Query)
		if matched == args.Inverse {
			return "", false
		}
	}
	if re != nil {
		matched := re.MatchString(msg)
		if matched == args.Inverse {
			return "", false
		}
	}

	return msg, true
}

func executeStreamLogs(ctx context.Context, requestID string, argsJSON string, deps ToolDeps, send streamSender) error {
	args, re, err := parseStreamArgs(argsJSON)
	if err != nil {
		return err
	}

	cs, err := deps.HostService.FindContainer(args.Host, args.ContainerID, deps.Labels)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}

	events := make(chan *container.LogEvent, 100)

	go func() {
		defer close(events)
		if err := cs.StreamLogs(ctx, time.Now().Add(-30*time.Second), container.STDOUT|container.STDERR, events); err != nil {
			log.Debug().Err(err).Str("container", cs.Container.Name).Msg("StreamLogs ended with error")
		}
	}()

	sendBatch := func(entries []*pb.LogEntry, endStream bool) error {
		if len(entries) == 0 && !endStream {
			return nil
		}
		resp := &pb.ToolResponse{
			RequestId: requestID,
			Type: &pb.ToolResponse_CallTool{
				CallTool: &pb.CallToolResponse{
					Success:   true,
					Stream:    !endStream,
					EndStream: endStream,
					Result: &pb.CallToolResponse_FetchLogs{
						FetchLogs: &pb.FetchLogsResult{
							ContainerName: cs.Container.Name,
							Entries:       entries,
						},
					},
				},
			},
		}
		return send(resp)
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	const batchSize = 50
	batch := make([]*pb.LogEntry, 0, batchSize)

	flush := func(endStream bool) error {
		if err := sendBatch(batch, endStream); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}

	for {
		select {
		case event, ok := <-events:
			if !ok {
				// Channel closed — drain and send end_stream
				return flush(true)
			}
			msg, matches := matchesFilters(event, args, re)
			if !matches {
				continue
			}
			batch = append(batch, &pb.LogEntry{
				Timestamp: event.Timestamp,
				Message:   msg,
				Stream:    event.Stream,
				Level:     event.Level,
			})
			if len(batch) >= batchSize {
				if err := flush(false); err != nil {
					return err
				}
			}

		case <-ticker.C:
			if len(batch) > 0 {
				if err := flush(false); err != nil {
					return err
				}
			}

		case <-ctx.Done():
			if err := flush(true); err != nil {
				log.Debug().Err(err).Msg("failed to send end_stream on cancel")
			}
			return ctx.Err()
		}
	}
}

