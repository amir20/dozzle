package cloud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/metadata"
)

// ViewContainer is one container the user can currently see.
type ViewContainer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Host string `json:"host"`
}

// ViewContext is what the user was looking at when they asked.
type ViewContext struct {
	Kind       string          `json:"kind"`
	Target     string          `json:"target,omitempty"`
	Containers []ViewContainer `json:"containers,omitempty"`
	Hosts      []string        `json:"hosts,omitempty"`
	Search     string          `json:"search,omitempty"`
	Levels     []string        `json:"levels,omitempty"`
	// VisibleAt is the moment in view, RFC 3339. A string rather than nanos
	// because that is what the browser has and what a human reads in a log; it
	// becomes nanoseconds on the wire to cloud.
	VisibleAt  string `json:"visibleAt,omitempty"`
	Historical bool   `json:"historical,omitempty"`
}

// ChatEvent is one thing that happened during a turn, on its way to the
// browser. Kind is "status", "delta", "done" or "error".
type ChatEvent struct {
	Kind      string `json:"kind"`
	Text      string `json:"text,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	Code      string `json:"code,omitempty"`
}

// ChatCredentials is resolved per turn rather than read from the client's
// stored key. Today it always answers with the instance key. When a user can
// attach a personal key, this is the only thing that changes: the turn is
// already the unit that has a user attached, and gRPC takes per-call
// credentials on a shared connection.
type ChatCredentials func() string

// Chat runs one assistant turn and emits everything it produces.
//
// The stream is opened here, inside the caller's request context, which is what
// makes the scoping work: cloud sends this turn's tool calls back down this
// stream, and they execute against the principal the caller resolved from their
// own HTTP request. Nothing about the user's identity crosses to cloud beyond
// the opaque userRef.
func (c *Client) Chat(
	ctx context.Context,
	message string,
	view ViewContext,
	userRef string,
	principal Principal,
	creds ChatCredentials,
	emit func(ChatEvent),
) error {
	apiKey := c.apiKeyFunc()
	if creds != nil {
		apiKey = creds()
	}
	if apiKey == "" {
		return ErrNotConfigured
	}

	client, err := c.unaryServiceClient()
	if err != nil {
		return err
	}

	mdPairs := []string{"x-api-key", apiKey}
	if c.instanceID != "" {
		mdPairs = append(mdPairs, "x-instance-id", c.instanceID)
	}
	stream, err := client.Chat(metadata.NewOutgoingContext(ctx, metadata.Pairs(mdPairs...)))
	if err != nil {
		return fmt.Errorf("cloud: chat: %w", err)
	}

	if err := stream.Send(&pb.ChatClientEvent{
		Type: &pb.ChatClientEvent_Turn{Turn: &pb.ChatTurn{
			Message: message,
			View:    viewToProto(view),
			UserRef: userRef,
		}},
	}); err != nil {
		return fmt.Errorf("cloud: chat send: %w", err)
	}

	// The turn's own deps: everything a tool does here happens as the person
	// who asked, not as the instance.
	deps := c.deps
	deps.Principal = principal

	for {
		event, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			// A cancelled request is the pane closing, which is not a failure.
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("cloud: chat recv: %w", err)
		}

		switch t := event.GetType().(type) {
		case *pb.ChatServerEvent_Status:
			emit(ChatEvent{Kind: "status", Text: t.Status.GetText()})
		case *pb.ChatServerEvent_Delta:
			emit(ChatEvent{Kind: "delta", Text: t.Delta.GetText()})
		case *pb.ChatServerEvent_Done:
			emit(ChatEvent{Kind: "done", SessionID: t.Done.GetSessionId()})
			return nil
		case *pb.ChatServerEvent_Error:
			emit(ChatEvent{Kind: "error", Text: t.Error.GetMessage(), Code: t.Error.GetCode()})
			return nil
		case *pb.ChatServerEvent_ToolCall:
			call := t.ToolCall.GetCallTool()
			if call == nil {
				continue
			}
			resp := ExecuteTool(ctx, call.GetName(), call.GetArgumentsJson(), deps)
			if err := stream.Send(&pb.ChatClientEvent{
				Type: &pb.ChatClientEvent_ToolResult{ToolResult: &pb.ToolResponse{
					RequestId: t.ToolCall.GetRequestId(),
					Type:      &pb.ToolResponse_CallTool{CallTool: resp},
				}},
			}); err != nil {
				log.Warn().Err(err).Msg("cloud: chat tool result send failed")
				return err
			}
		}
	}
}

func viewToProto(v ViewContext) *pb.ViewContext {
	containers := make([]*pb.ViewContainer, 0, len(v.Containers))
	for _, c := range v.Containers {
		containers = append(containers, &pb.ViewContainer{Id: c.ID, Name: c.Name, Host: c.Host})
	}
	// An unparseable or absent timestamp sends 0, which cloud reads as "no
	// particular moment". Refusing the whole turn over it would be worse.
	var visibleAtNs int64
	if v.VisibleAt != "" {
		if t, err := time.Parse(time.RFC3339, v.VisibleAt); err == nil {
			visibleAtNs = t.UnixNano()
		}
	}

	return &pb.ViewContext{
		Kind:        v.Kind,
		Target:      v.Target,
		Containers:  containers,
		Hosts:       v.Hosts,
		Search:      v.Search,
		Levels:      v.Levels,
		VisibleAtNs: visibleAtNs,
		Historical:  v.Historical,
	}
}
