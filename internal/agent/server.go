package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net"
	"time"

	"github.com/amir20/dozzle/internal/agent/pb"
	"github.com/amir20/dozzle/internal/docker"
	orderedmap "github.com/wk8/go-ordered-map/v2"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"google.golang.org/grpc/status"
)

type server struct {
	client docker.Client
	store  *docker.ContainerStore

	pb.UnimplementedAgentServiceServer
}

func NewServer(client docker.Client) pb.AgentServiceServer {
	return &server{
		client: client,
		store:  docker.NewContainerStore(context.Background(), client),
	}
}

func (s *server) StreamLogs(in *pb.StreamLogsRequest, out pb.AgentService_StreamLogsServer) error {
	since := time.Time{}
	if in.Since != nil {
		since = in.Since.AsTime()
	}

	reader, err := s.client.ContainerLogs(out.Context(), in.ContainerId, since, docker.StdType(in.StreamTypes))
	if err != nil {
		return err
	}

	container, err := s.store.FindContainer(in.ContainerId)
	if err != nil {
		return err
	}

	g := docker.NewEventGenerator(reader, container)

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

func (s *server) LogsBetweenDates(in *pb.LogsBetweenDatesRequest, out pb.AgentService_LogsBetweenDatesServer) error {
	reader, err := s.client.ContainerLogsBetweenDates(out.Context(), in.ContainerId, in.Since.AsTime(), in.Until.AsTime(), docker.StdType(in.StreamTypes))
	if err != nil {
		return err
	}

	container, err := s.client.FindContainer(in.ContainerId)
	if err != nil {
		return err
	}

	g := docker.NewEventGenerator(reader, container)

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
	reader, err := s.client.ContainerLogsBetweenDates(out.Context(), in.ContainerId, in.Since.AsTime(), in.Until.AsTime(), docker.StdType(in.StreamTypes))

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
	events := make(chan docker.ContainerEvent)

	s.store.SubscribeEvents(out.Context(), events)

	for {
		select {
		case event := <-events:
			out.Send(&pb.StreamEventsResponse{
				Event: &pb.ContainerEvent{
					ActorId: event.ActorID,
					Name:    event.Name,
					Host:    event.Host,
				},
			})
		case <-out.Context().Done():
			return nil
		}
	}
}

func (s *server) StreamStats(in *pb.StreamStatsRequest, out pb.AgentService_StreamStatsServer) error {
	stats := make(chan docker.ContainerStat)

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
	container, err := s.store.FindContainer(in.ContainerId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &pb.FindContainerResponse{
		Container: &pb.Container{
			Id:      container.ID,
			Name:    container.Name,
			Image:   container.Image,
			ImageId: container.ImageID,
			Command: container.Command,
			Created: timestamppb.New(container.Created),
			State:   container.State,
			Status:  container.Status,
			Health:  container.Health,
			Host:    container.Host,
			Tty:     container.Tty,
			Labels:  container.Labels,
			Group:   container.Group,
			Started: timestamppb.New(*container.StartedAt),
		},
	}, nil
}

func (s *server) ListContainers(ctx context.Context, in *pb.ListContainersRequest) (*pb.ListContainersResponse, error) {
	containers, err := s.store.ListContainers()
	if err != nil {
		return nil, err
	}

	var pbContainers []*pb.Container

	for _, container := range containers {
		var pbStats []*pb.ContainerStat
		for _, stat := range container.Stats.Data() {
			pbStats = append(pbStats, &pb.ContainerStat{
				Id:            stat.ID,
				CpuPercent:    stat.CPUPercent,
				MemoryPercent: stat.MemoryPercent,
				MemoryUsage:   stat.MemoryUsage,
			})
		}

		var startedAt *timestamppb.Timestamp
		if container.StartedAt != nil {
			startedAt = timestamppb.New(*container.StartedAt)
		}

		pbContainers = append(pbContainers, &pb.Container{
			Id:      container.ID,
			Name:    container.Name,
			Image:   container.Image,
			ImageId: container.ImageID,
			Created: timestamppb.New(container.Created),
			State:   container.State,
			Status:  container.Status,
			Health:  container.Health,
			Host:    container.Host,
			Tty:     container.Tty,
			Labels:  container.Labels,
			Group:   container.Group,
			Started: startedAt,
			Stats:   pbStats,
			Command: container.Command,
		})
	}

	return &pb.ListContainersResponse{
		Containers: pbContainers,
	}, nil
}

func (s *server) HostInfo(ctx context.Context, in *pb.HostInfoRequest) (*pb.HostInfoResponse, error) {
	host := s.client.Host()
	return &pb.HostInfoResponse{
		Host: &pb.Host{
			Id:       host.ID,
			Name:     host.Name,
			CpuCores: uint32(host.NCPU),
			Memory:   uint32(host.MemTotal),
		},
	}, nil
}

func (s *server) StreamContainerStarted(in *pb.StreamContainerStartedRequest, out pb.AgentService_StreamContainerStartedServer) error {
	containers := make(chan docker.Container)

	go s.store.SubscribeNewContainers(out.Context(), containers)

	for {
		select {
		case container := <-containers:
			out.Send(&pb.StreamContainerStartedResponse{
				Container: &pb.Container{
					Id:      container.ID,
					Name:    container.Name,
					Image:   container.Image,
					ImageId: container.ImageID,
					Created: timestamppb.New(container.Created),
					State:   container.State,
					Status:  container.Status,
					Health:  container.Health,
					Host:    container.Host,
					Tty:     container.Tty,
					Labels:  container.Labels,
					Group:   container.Group,
					Started: timestamppb.New(*container.StartedAt),
				},
			})
		case <-out.Context().Done():
			return nil
		}
	}
}

func RunServer(client docker.Client, certificates tls.Certificate, listener net.Listener) {
	caCertPool := x509.NewCertPool()
	c, err := x509.ParseCertificate(certificates.Certificate[0])
	if err != nil {
		log.Fatalf("failed to parse certificate: %v", err)
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
	pb.RegisterAgentServiceServer(grpcServer, NewServer(client))

	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Infof("gRPC server listening on %s", listener.Addr().String())
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func logEventToPb(event *docker.LogEvent) *pb.LogEvent {
	var message *anypb.Any
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
		log.Fatalf("agent server: unknown type %T", event.Message)
	}

	return &pb.LogEvent{
		Message:     message,
		Timestamp:   timestamppb.New(time.Unix(event.Timestamp, 0)),
		Id:          event.Id,
		ContainerId: event.ContainerID,
		Level:       event.Level,
		Stream:      event.Stream,
		Position:    string(event.Position),
	}
}

func orderedMapToJSONBytes[T any](data *orderedmap.OrderedMap[string, T]) []byte {
	bytes := bytes.Buffer{}
	json.NewEncoder(&bytes).Encode(data)
	return bytes.Bytes()
}
