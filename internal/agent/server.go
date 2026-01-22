package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"

	"encoding/json"

	"time"

	"github.com/amir20/dozzle/internal/agent/pb"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"google.golang.org/grpc/status"
)

// NotificationConfigHandler handles notification config updates received from the main server
type NotificationConfigHandler interface {
	HandleNotificationConfig(subscriptions []types.SubscriptionConfig, dispatchers []types.DispatcherConfig) error
}

// ClientService is the interface for container operations used by the agent server
type ClientService interface {
	FindContainer(ctx context.Context, id string, labels container.ContainerLabels) (container.Container, error)
	ListContainers(ctx context.Context, filter container.ContainerLabels) ([]container.Container, error)
	Host(ctx context.Context) (container.Host, error)
	ContainerAction(ctx context.Context, container container.Container, action container.ContainerAction) error
	LogsBetweenDates(ctx context.Context, container container.Container, from time.Time, to time.Time, stdTypes container.StdType) (<-chan *container.LogEvent, error)
	RawLogs(ctx context.Context, container container.Container, from time.Time, to time.Time, stdTypes container.StdType) (io.ReadCloser, error)
	SubscribeStats(context.Context, chan<- container.ContainerStat)
	SubscribeEvents(context.Context, chan<- container.ContainerEvent)
	SubscribeContainersStarted(context.Context, chan<- container.Container)
	StreamLogs(context.Context, container.Container, time.Time, container.StdType, chan<- *container.LogEvent) error
	Attach(context.Context, container.Container, container.ExecEventReader, io.Writer) error
	Exec(context.Context, container.Container, []string, container.ExecEventReader, io.Writer) error
}

type server struct {
	service                   ClientService
	version                   string
	notificationConfigHandler NotificationConfigHandler

	pb.UnimplementedAgentServiceServer
}

func newServer(service ClientService, dozzleVersion string, notificationHandler NotificationConfigHandler) pb.AgentServiceServer {
	return &server{
		service:                   service,
		version:                   dozzleVersion,
		notificationConfigHandler: notificationHandler,
	}
}

func (s *server) StreamLogs(in *pb.StreamLogsRequest, out pb.AgentService_StreamLogsServer) error {
	since := time.Time{}
	if in.Since != nil {
		since = in.Since.AsTime()
	}

	c, err := s.service.FindContainer(out.Context(), in.ContainerId, container.ContainerLabels{})
	if err != nil {
		return err
	}

	events := make(chan *container.LogEvent)
	go func() {
		defer close(events)
		s.service.StreamLogs(out.Context(), c, since, container.StdType(in.StreamTypes), events)
	}()

	for event := range events {
		if event != nil {
			out.Send(&pb.StreamLogsResponse{
				Event: logEventToPb(event),
			})
		}
	}

	return nil
}

func (s *server) LogsBetweenDates(in *pb.LogsBetweenDatesRequest, out pb.AgentService_LogsBetweenDatesServer) error {
	c, err := s.service.FindContainer(out.Context(), in.ContainerId, container.ContainerLabels{})
	if err != nil {
		return err
	}

	events, err := s.service.LogsBetweenDates(out.Context(), c, in.Since.AsTime(), in.Until.AsTime(), container.StdType(in.StreamTypes))
	if err != nil {
		return err
	}

	for {
		select {
		case event, ok := <-events:
			if !ok {
				// Channel closed, exit cleanly
				return nil
			}
			out.Send(&pb.StreamLogsResponse{
				Event: logEventToPb(event),
			})
		case <-out.Context().Done():
			return nil
		}
	}
}

func (s *server) StreamRawBytes(in *pb.StreamRawBytesRequest, out pb.AgentService_StreamRawBytesServer) error {
	c, err := s.service.FindContainer(out.Context(), in.ContainerId, container.ContainerLabels{})
	if err != nil {
		return err
	}

	reader, err := s.service.RawLogs(out.Context(), c, in.Since.AsTime(), in.Until.AsTime(), container.StdType(in.StreamTypes))
	if err != nil {
		return err
	}
	defer reader.Close()

	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if n == 0 {
			break
		}

		if err := out.Send(&pb.StreamRawBytesResponse{
			Data: buf[:n],
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *server) StreamEvents(in *pb.StreamEventsRequest, out pb.AgentService_StreamEventsServer) error {
	events := make(chan container.ContainerEvent)

	s.service.SubscribeEvents(out.Context(), events)

	for {
		select {
		case event := <-events:
			out.Send(&pb.StreamEventsResponse{
				Event: &pb.ContainerEvent{
					ActorId:   event.ActorID,
					Name:      event.Name,
					Host:      event.Host,
					Timestamp: timestamppb.New(event.Time),
				},
			})
		case <-out.Context().Done():
			return nil
		}
	}
}

func (s *server) StreamStats(in *pb.StreamStatsRequest, out pb.AgentService_StreamStatsServer) error {
	stats := make(chan container.ContainerStat)

	s.service.SubscribeStats(out.Context(), stats)

	for {
		select {
		case stat := <-stats:
			out.Send(&pb.StreamStatsResponse{
				Stat: &pb.ContainerStat{
					Id:             stat.ID,
					CpuPercent:     stat.CPUPercent,
					MemoryPercent:  stat.MemoryPercent,
					MemoryUsage:    stat.MemoryUsage,
					NetworkRxTotal: stat.NetworkRxTotal,
					NetworkTxTotal: stat.NetworkTxTotal,
				},
			})
		case <-out.Context().Done():
			return nil
		}
	}
}

func (s *server) FindContainer(ctx context.Context, in *pb.FindContainerRequest) (*pb.FindContainerResponse, error) {
	labels := make(container.ContainerLabels)
	if in.GetFilter() != nil {
		for k, v := range in.GetFilter() {
			labels[k] = append(labels[k], v.GetValues()...)
		}
	}

	c, err := s.service.FindContainer(ctx, in.ContainerId, labels)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	proto := c.ToProto()
	return &pb.FindContainerResponse{
		Container: &proto,
	}, nil
}

func (s *server) ListContainers(ctx context.Context, in *pb.ListContainersRequest) (*pb.ListContainersResponse, error) {
	labels := make(container.ContainerLabels)
	if in.GetFilter() != nil {
		for k, v := range in.GetFilter() {
			labels[k] = append(labels[k], v.GetValues()...)
		}
	}

	containers, err := s.service.ListContainers(ctx, labels)
	if err != nil {
		return nil, err
	}

	var pbContainers []*pb.Container
	for _, c := range containers {
		proto := c.ToProto()
		pbContainers = append(pbContainers, &proto)
	}

	return &pb.ListContainersResponse{
		Containers: pbContainers,
	}, nil
}

func (s *server) HostInfo(ctx context.Context, in *pb.HostInfoRequest) (*pb.HostInfoResponse, error) {
	host, err := s.service.Host(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.HostInfoResponse{
		Host: &pb.Host{
			Id:            host.ID,
			Name:          host.Name,
			CpuCores:      uint32(host.NCPU),
			Memory:        uint64(host.MemTotal),
			DockerVersion: host.DockerVersion,
			AgentVersion:  s.version,
		},
	}, nil
}

func (s *server) StreamContainerStarted(in *pb.StreamContainerStartedRequest, out pb.AgentService_StreamContainerStartedServer) error {
	containers := make(chan container.Container)

	go s.service.SubscribeContainersStarted(out.Context(), containers)

	for {
		select {
		case container := <-containers:
			c := container.ToProto()
			out.Send(&pb.StreamContainerStartedResponse{
				Container: &c,
			})
		case <-out.Context().Done():
			return nil
		}
	}
}

func (s *server) ContainerAction(ctx context.Context, in *pb.ContainerActionRequest) (*pb.ContainerActionResponse, error) {
	var action container.ContainerAction
	switch in.Action {
	case pb.ContainerAction_Start:
		action = container.Start

	case pb.ContainerAction_Stop:
		action = container.Stop

	case pb.ContainerAction_Restart:
		action = container.Restart

	default:
		return nil, status.Error(codes.InvalidArgument, "invalid action")
	}

	c, err := s.service.FindContainer(ctx, in.ContainerId, container.ContainerLabels{})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	err = s.service.ContainerAction(ctx, c, action)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.ContainerActionResponse{}, nil
}

// terminalMessage represents a message from a terminal gRPC stream (exec or attach)
type terminalMessage interface {
	GetStdin() []byte
	GetResize() *pb.ResizePayload
}

// protoEventReader converts gRPC protobuf messages directly to ExecEvents (no JSON)
type protoEventReader struct {
	recv func() (terminalMessage, error)
}

func (r *protoEventReader) ReadEvent() (*container.ExecEvent, error) {
	msg, err := r.recv()
	if err != nil {
		return nil, err
	}

	if stdin := msg.GetStdin(); stdin != nil {
		return &container.ExecEvent{Type: "userinput", Data: string(stdin)}, nil
	} else if resize := msg.GetResize(); resize != nil {
		return &container.ExecEvent{Type: "resize", Width: uint(resize.Width), Height: uint(resize.Height)}, nil
	}

	// Skip unknown message types
	return r.ReadEvent()
}

// terminalStreamWriter adapts a gRPC terminal stream to io.Writer
type terminalStreamWriter struct {
	send func([]byte) error
}

func (w *terminalStreamWriter) Write(p []byte) (int, error) {
	if err := w.send(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (s *server) ContainerExec(stream pb.AgentService_ContainerExecServer) error {
	request, err := stream.Recv()
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	c, err := s.service.FindContainer(stream.Context(), request.ContainerId, container.ContainerLabels{})
	if err != nil {
		return status.Error(codes.NotFound, err.Error())
	}

	reader := &protoEventReader{recv: func() (terminalMessage, error) { return stream.Recv() }}
	writer := &terminalStreamWriter{send: func(p []byte) error { return stream.Send(&pb.ContainerExecResponse{Stdout: p}) }}

	if err := s.service.Exec(stream.Context(), c, request.Command, reader, writer); err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}

func (s *server) ContainerAttach(stream pb.AgentService_ContainerAttachServer) error {
	request, err := stream.Recv()
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	c, err := s.service.FindContainer(stream.Context(), request.ContainerId, container.ContainerLabels{})
	if err != nil {
		return status.Error(codes.NotFound, err.Error())
	}

	reader := &protoEventReader{recv: func() (terminalMessage, error) { return stream.Recv() }}
	writer := &terminalStreamWriter{send: func(p []byte) error { return stream.Send(&pb.ContainerAttachResponse{Stdout: p}) }}

	if err := s.service.Attach(stream.Context(), c, reader, writer); err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}

func (s *server) UpdateNotificationConfig(ctx context.Context, req *pb.UpdateNotificationConfigRequest) (*pb.UpdateNotificationConfigResponse, error) {
	if s.notificationConfigHandler == nil {
		log.Warn().Msg("No notification config handler registered, ignoring config update")
		return &pb.UpdateNotificationConfigResponse{}, nil
	}

	// Convert proto subscriptions to types
	subscriptions := make([]types.SubscriptionConfig, len(req.Subscriptions))
	for i, sub := range req.Subscriptions {
		subscriptions[i] = types.SubscriptionConfig{
			ID:                  int(sub.Id),
			Name:                sub.Name,
			Enabled:             sub.Enabled,
			DispatcherID:        int(sub.DispatcherId),
			LogExpression:       sub.LogExpression,
			ContainerExpression: sub.ContainerExpression,
		}
	}

	// Convert proto dispatchers to types
	dispatchers := make([]types.DispatcherConfig, len(req.Dispatchers))
	for i, d := range req.Dispatchers {
		dispatchers[i] = types.DispatcherConfig{
			ID:       int(d.Id),
			Name:     d.Name,
			Type:     d.Type,
			URL:      d.Url,
			Template: d.Template,
		}
	}

	// Call the handler
	if err := s.notificationConfigHandler.HandleNotificationConfig(subscriptions, dispatchers); err != nil {
		log.Error().Err(err).Msg("Failed to handle notification config")
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Info().Int("subscriptions", len(subscriptions)).Int("dispatchers", len(dispatchers)).Msg("Updated notification config from main server")
	return &pb.UpdateNotificationConfigResponse{}, nil
}

func NewServer(service ClientService, certificates tls.Certificate, dozzleVersion string, notificationHandler NotificationConfigHandler) (*grpc.Server, error) {
	caCertPool := x509.NewCertPool()
	c, err := x509.ParseCertificate(certificates.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	caCertPool.AddCert(c)

	// Create the TLS configuration
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificates},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert, // Require client certificates
	}

	// Create the gRPC server with the credentials
	creds := credentials.NewTLS(tlsConfig)

	grpcServer := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterAgentServiceServer(grpcServer, newServer(service, dozzleVersion, notificationHandler))

	return grpcServer, nil
}

func logEventToPb(event *container.LogEvent) *pb.LogEvent {
	var message *anypb.Any

	if event.Message == nil {
		log.Fatal().Interface("event", event).Msg("agent server: message is nil. This should not happen.")
	}

	switch data := event.Message.(type) {
	case string:
		message, _ = anypb.New(&pb.SingleMessage{
			Message: data,
		})

	case []container.LogFragment:
		fragments := make([]*pb.LogFragment, len(data))
		for i, f := range data {
			fragments[i] = &pb.LogFragment{
				Message: f.Message,
			}
		}
		message, _ = anypb.New(&pb.GroupMessage{
			Fragments: fragments,
		})

	case *orderedmap.OrderedMap[string, any]:
		message, _ = anypb.New(&pb.ComplexMessage{
			Data: orderedMapToJSONBytes(data),
		})
	case *orderedmap.OrderedMap[string, string]:
		message, _ = anypb.New(&pb.ComplexMessage{
			Data: orderedMapToJSONBytes(data),
		})

	default:
		log.Error().Type("message", event.Message).Msg("agent server: unknown message type")
	}

	return &pb.LogEvent{
		Message:     message,
		Timestamp:   timestamppb.New(time.Unix(event.Timestamp, 0)),
		Id:          event.Id,
		ContainerId: event.ContainerID,
		Level:       event.Level,
		Stream:      event.Stream,
		Type:        string(event.Type),
		RawMessage:  string(event.RawMessage),
	}
}

func orderedMapToJSONBytes[T any](data *orderedmap.OrderedMap[string, T]) []byte {
	bytes := bytes.Buffer{}
	json.NewEncoder(&bytes).Encode(data)
	return bytes.Bytes()
}
