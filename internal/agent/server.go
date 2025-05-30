package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"

	"encoding/json"

	"time"

	"github.com/amir20/dozzle/internal/agent/pb"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/rs/zerolog/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"google.golang.org/grpc/status"
)

type server struct {
	client  container.Client
	store   *container.ContainerStore
	version string

	pb.UnimplementedAgentServiceServer
}

func newServer(client container.Client, dozzleVersion string, labels container.ContainerLabels) pb.AgentServiceServer {
	statsCollector := docker.NewDockerStatsCollector(client, labels)
	return &server{
		client:  client,
		version: dozzleVersion,

		store: container.NewContainerStore(context.Background(), client, statsCollector, labels),
	}
}

func (s *server) StreamLogs(in *pb.StreamLogsRequest, out pb.AgentService_StreamLogsServer) error {
	since := time.Time{}
	if in.Since != nil {
		since = in.Since.AsTime()
	}

	c, err := s.store.FindContainer(in.ContainerId, container.ContainerLabels{})
	if err != nil {
		return err
	}

	reader, err := s.client.ContainerLogs(out.Context(), in.ContainerId, since, container.StdType(in.StreamTypes))
	if err != nil {
		return err
	}

	dockerReader := docker.NewLogReader(reader, c.Tty)
	g := container.NewEventGenerator(out.Context(), dockerReader, c)

	for event := range g.Events {
		out.Send(&pb.StreamLogsResponse{
			Event: logEventToPb(event),
		})
	}

	select {
	case e := <-g.Errors:
		return e
	default:
		return nil
	}
}

func (s *server) LogsBetweenDates(in *pb.LogsBetweenDatesRequest, out pb.AgentService_LogsBetweenDatesServer) error {
	reader, err := s.client.ContainerLogsBetweenDates(out.Context(), in.ContainerId, in.Since.AsTime(), in.Until.AsTime(), container.StdType(in.StreamTypes))
	if err != nil {
		return err
	}

	c, err := s.client.FindContainer(out.Context(), in.ContainerId)
	if err != nil {
		return err
	}

	dockerReader := docker.NewLogReader(reader, c.Tty)
	g := container.NewEventGenerator(out.Context(), dockerReader, c)

	for {
		select {
		case event := <-g.Events:
			out.Send(&pb.StreamLogsResponse{
				Event: logEventToPb(event),
			})
		case e := <-g.Errors:
			return e
		case <-out.Context().Done():
			return nil
		}
	}
}

func (s *server) StreamRawBytes(in *pb.StreamRawBytesRequest, out pb.AgentService_StreamRawBytesServer) error {
	reader, err := s.client.ContainerLogsBetweenDates(out.Context(), in.ContainerId, in.Since.AsTime(), in.Until.AsTime(), container.StdType(in.StreamTypes))

	if err != nil {
		return err
	}

	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if err != nil {
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

	s.store.SubscribeEvents(out.Context(), events)

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

	s.store.SubscribeStats(out.Context(), stats)

	for {
		select {
		case stat := <-stats:
			out.Send(&pb.StreamStatsResponse{
				Stat: &pb.ContainerStat{
					Id:            stat.ID,
					CpuPercent:    stat.CPUPercent,
					MemoryPercent: stat.MemoryPercent,
					MemoryUsage:   stat.MemoryUsage,
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

	container, err := s.store.FindContainer(in.ContainerId, labels)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	c := container.ToProto()
	return &pb.FindContainerResponse{
		Container: &c,
	}, nil
}

func (s *server) ListContainers(ctx context.Context, in *pb.ListContainersRequest) (*pb.ListContainersResponse, error) {
	labels := make(container.ContainerLabels)
	if in.GetFilter() != nil {
		for k, v := range in.GetFilter() {
			labels[k] = append(labels[k], v.GetValues()...)
		}
	}

	containers, err := s.store.ListContainers(labels)
	if err != nil {
		return nil, err
	}

	var pbContainers []*pb.Container
	for _, container := range containers {
		c := container.ToProto()
		pbContainers = append(pbContainers, &c)
	}

	return &pb.ListContainersResponse{
		Containers: pbContainers,
	}, nil
}

func (s *server) HostInfo(ctx context.Context, in *pb.HostInfoRequest) (*pb.HostInfoResponse, error) {
	host := s.client.Host()
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

	go s.store.SubscribeNewContainers(out.Context(), containers)

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

	err := s.client.ContainerActions(ctx, action, in.ContainerId)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.ContainerActionResponse{}, nil
}

func (s *server) ContainerExec(stream pb.AgentService_ContainerExecServer) error {
	request, err := stream.Recv()
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	cancelCtx, cancel := context.WithCancel(stream.Context())
	containerWriter, containerReader, err := s.client.ContainerExec(cancelCtx, request.ContainerId, request.Command)
	if err != nil {
		cancel()
		return status.Error(codes.Internal, err.Error())
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		defer cancel()
		defer containerWriter.Close()
		for {
			stdinReq, err := stream.Recv()
			if err != nil {
				return
			}

			if _, err := containerWriter.Write(stdinReq.Stdin); err != nil {
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		defer cancel()
		buffer := make([]byte, 1024)
		for {
			n, err := containerReader.Read(buffer)
			if err != nil {
				return
			}

			if err := stream.Send(&pb.ContainerExecResponse{Stdout: buffer[:n]}); err != nil {
				return
			}
		}
	}()

	wg.Wait()

	return nil
}

func NewServer(client container.Client, certificates tls.Certificate, dozzleVersion string, labels container.ContainerLabels) (*grpc.Server, error) {
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
	pb.RegisterAgentServiceServer(grpcServer, newServer(client, dozzleVersion, labels))

	return grpcServer, nil
}

func logEventToPb(event *container.LogEvent) *pb.LogEvent {
	var message *anypb.Any

	if event.Message == nil {
		log.Fatal().Interface("event", event).Msg("agent server: message is nil. This should not happen.")
	}

	switch data := event.Message.(type) {
	case string:
		message, _ = anypb.New(&pb.SimpleMessage{
			Message: data,
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
		Position:    string(event.Position),
		RawMessage:  string(event.RawMessage),
	}
}

func orderedMapToJSONBytes[T any](data *orderedmap.OrderedMap[string, T]) []byte {
	bytes := bytes.Buffer{}
	json.NewEncoder(&bytes).Encode(data)
	return bytes.Bytes()
}
