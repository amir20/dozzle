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

// ViewLogLine is one line as Dozzle rendered it: ANSI already stripped and the
// message already truncated, because this is what the user saw rather than what
// the container wrote.
type ViewLogLine struct {
	Timestamp   string `json:"timestamp,omitempty"`
	Level       string `json:"level,omitempty"`
	ContainerID string `json:"containerId,omitempty"`
	Message     string `json:"message"`
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
	// Lines is the window on screen, oldest first. Sent so the assistant reads
	// what the person is looking at instead of fetching its own slice.
	Lines []ViewLogLine `json:"lines,omitempty"`
	// Focused is the one line the user pointed at, when they asked from a log
	// row rather than from the composer.
	Focused *ViewLogLine `json:"focused,omitempty"`
}

// ChatEvent is one thing that happened during a turn, on its way to the
// browser. Kind is "status", "delta", "reset", "done" or "error".
type ChatEvent struct {
	Kind string `json:"kind"`
	Text string `json:"text,omitempty"`
	// Activity names what Dozzle is doing right now, for a status the browser
	// renders in the reader's own language. A token rather than a sentence:
	// this side of the wire has no locale, and cloud's own status lines are
	// already prose it wrote.
	Activity  string `json:"activity,omitempty"`
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
			if t.Delta.GetReset_() {
				emit(ChatEvent{Kind: "reset"})
			}
			if text := t.Delta.GetText(); text != "" {
				emit(ChatEvent{Kind: "delta", Text: text})
			}
		case *pb.ChatServerEvent_Done:
			emit(ChatEvent{Kind: "done", SessionID: t.Done.GetSessionId()})
			return nil
		case *pb.ChatServerEvent_Error:
			emit(ChatEvent{Kind: "error", Text: t.Error.GetMessage(), Code: t.Error.GetCode()})
			return nil
		case *pb.ChatServerEvent_ToolCall:
			resp := c.chatToolResponse(ctx, t.ToolCall, deps, emit)
			if err := stream.Send(&pb.ChatClientEvent{
				Type: &pb.ChatClientEvent_ToolResult{ToolResult: resp},
			}); err != nil {
				log.Warn().Err(err).Msg("cloud: chat tool result send failed")
				return err
			}
		}
	}
}

// chatToolResponse answers one tool request from a chat turn.
//
// Cloud drives discovery down this same stream: it sends ListTools and waits
// for the reply before the assistant has any tools at all. Dropping that
// request is why a turn sat on "Thinking…" while cloud logged "tool discovery
// failed; continuing without live tools".
func (c *Client) chatToolResponse(ctx context.Context, req *pb.ToolRequest, deps ToolDeps, emit func(ChatEvent)) *pb.ToolResponse {
	resp := &pb.ToolResponse{RequestId: req.GetRequestId()}

	switch t := req.GetType().(type) {
	case *pb.ToolRequest_ListTools:
		// Scoped to the person asking rather than the instance-wide cache, so
		// the assistant is never offered a tool this user would be denied.
		resp.Type = &pb.ToolResponse_ListTools{ListTools: &pb.ListToolsResponse{
			Tools:   AvailableTools(deps.EnableActions, deps.Principal),
			Version: c.version,
		}}

	case *pb.ToolRequest_CallTool:
		name := t.CallTool.GetName()
		// A turn is one round trip, so the long-lived streaming tool has nowhere
		// to stream to. It takes the same arguments as the snapshot, which is
		// what the model wanted anyway; answering "unknown tool" just gets it
		// called again.
		if name == toolStreamLogs {
			name = toolFetchContainerLogs
		}
		// Cloud sends its own status lines, but only Dozzle knows a tool is
		// running right now. Without this a multi-round investigation shows
		// nothing but "Thinking…" and reads as hung.
		emit(ChatEvent{Kind: "status", Activity: toolActivity(name)})
		resp.Type = &pb.ToolResponse_CallTool{
			CallTool: ExecuteTool(ctx, name, t.CallTool.GetArgumentsJson(), deps),
		}

	default:
		// Nothing on a chat stream is long lived, so cancel_stream (and anything
		// newer than this build) has no meaning here. Cloud still gets an
		// answer rather than waiting out its timeout.
		resp.Type = &pb.ToolResponse_CallTool{CallTool: &pb.CallToolResponse{
			Success: false,
			Error:   "unsupported request on a chat stream",
		}}
	}

	return resp
}

// toolActivity is the token for what is happening while a tool runs. Coarser
// than the tool list on purpose: the reader wants to know the assistant is
// doing something, not which RPC it picked, and the browser has one phrase to
// translate per activity rather than one per tool.
func toolActivity(name string) string {
	switch name {
	case toolFetchContainerLogs:
		return "logs"
	case toolListHosts, toolFindContainers, toolListRunningContainers, toolListAllContainers:
		return "containers"
	case toolGetRunningContainerStats:
		return "stats"
	case toolInspectContainer:
		return "inspect"
	case toolListNotifications, toolCreateLogNotification, toolCreateMetricNotification, toolCreateEventNotification:
		return "notifications"
	case toolStartContainer, toolStopContainer, toolRestartContainer, toolRemoveContainer, toolUpdateContainer:
		return "action"
	default:
		return "working"
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

	lines := make([]*pb.ViewLogLine, 0, len(v.Lines))
	for _, l := range v.Lines {
		lines = append(lines, lineToProto(l))
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
		Lines:       lines,
		Focused:     focusedToProto(v.Focused),
	}
}

func lineToProto(l ViewLogLine) *pb.ViewLogLine {
	var ts int64
	if l.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, l.Timestamp); err == nil {
			ts = t.UnixNano()
		}
	}
	return &pb.ViewLogLine{
		TimestampNs: ts,
		Level:       l.Level,
		ContainerId: l.ContainerID,
		Message:     l.Message,
	}
}

func focusedToProto(l *ViewLogLine) *pb.ViewLogLine {
	if l == nil {
		return nil
	}
	return lineToProto(*l)
}
