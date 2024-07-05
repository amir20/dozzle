package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/agent/pb"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/amir20/dozzle/internal/utils"
	log "github.com/sirupsen/logrus"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Client struct {
	client pb.AgentServiceClient
	host   docker.Host
}

func NewClient(endpoint string, certificates tls.Certificate, opts ...grpc.DialOption) (*Client, error) {
	caCertPool := x509.NewCertPool()
	c, err := x509.ParseCertificate(certificates.Certificate[0])
	if err != nil {
		log.Fatalf("failed to parse certificate: %v", err)
	}
	caCertPool.AddCert(c)
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{certificates},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true, // Set to true if the server's hostname does not match the certificate
	}

	// Create the gRPC transport credentials
	creds := credentials.NewTLS(tlsConfig)

	opts = append(opts, grpc.WithTransportCredentials(creds))
	conn, err := grpc.NewClient(endpoint, opts...)
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)
	}

	client := pb.NewAgentServiceClient(conn)
	info, err := client.HostInfo(context.Background(), &pb.HostInfoRequest{})
	if err != nil {
		return nil, err
	}

	return &Client{
		client: client,

		host: docker.Host{
			ID:       info.Host.Id,
			Name:     info.Host.Name,
			NCPU:     int(info.Host.CpuCores),
			MemTotal: int64(info.Host.Memory),
			Endpoint: endpoint,
		},
	}, nil
}

func rpcErrToErr(err error) error {
	status, ok := status.FromError(err)
	if !ok {
		return err
	}

	if status.Code() == codes.Unknown && status.Message() == "EOF" {
		return fmt.Errorf("found EOF while streaming logs: %w", io.EOF)
	}

	switch status.Code() {
	case codes.Canceled:
		return fmt.Errorf("canceled: %v with %w", status.Message(), context.Canceled)
	case codes.DeadlineExceeded:
		return fmt.Errorf("deadline exceeded: %v with %w", status.Message(), context.DeadlineExceeded)
	case codes.Unknown:
		return fmt.Errorf("unknown error: %v with %w", status.Message(), err)
	default:
		return fmt.Errorf("unknown error: %v with %w", status.Message(), err)
	}
}

func (c *Client) LogsBetweenDates(ctx context.Context, containerID string, since time.Time, until time.Time, std docker.StdType) (<-chan *docker.LogEvent, error) {
	stream, err := c.client.LogsBetweenDates(ctx, &pb.LogsBetweenDatesRequest{
		ContainerId: containerID,
		Since:       timestamppb.New(since),
		Until:       timestamppb.New(until),
		StreamTypes: int32(std),
	})

	if err != nil {
		return nil, err
	}

	events := make(chan *docker.LogEvent)

	go func() {
		sendLogs(stream, events)
		close(events)
	}()

	return events, nil
}

func (c *Client) StreamContainerLogs(ctx context.Context, containerID string, since time.Time, std docker.StdType, events chan<- *docker.LogEvent) error {
	stream, err := c.client.StreamLogs(ctx, &pb.StreamLogsRequest{
		ContainerId: containerID,
		Since:       timestamppb.New(since),
		StreamTypes: int32(std),
	})

	if err != nil {
		return err
	}

	return sendLogs(stream, events)
}

func sendLogs(stream pb.AgentService_StreamLogsClient, events chan<- *docker.LogEvent) error {
	for {
		resp, err := stream.Recv()
		if err != nil {
			return rpcErrToErr(err)
		}

		m, err := resp.Event.Message.UnmarshalNew()
		if err != nil {
			log.Fatalf("cannot unpack message %v", err)
		}

		var message any
		switch m := m.(type) {
		case *pb.SimpleMessage:
			message = m.Message

		case *pb.ComplexMessage:
			message = jsonBytesToOrderedMap(m.Data)
		default:
			log.Fatalf("agent client: unknown type %T", m)
		}

		events <- &docker.LogEvent{
			Id:          resp.Event.Id,
			ContainerID: resp.Event.ContainerId,
			Message:     message,
			Timestamp:   resp.Event.Timestamp.AsTime().Unix(),
			Position:    docker.LogPosition(resp.Event.Position),
			Level:       resp.Event.Level,
			Stream:      resp.Event.Stream,
		}
	}
}

func (c *Client) StreamRawBytes(ctx context.Context, containerID string, since time.Time, until time.Time, std docker.StdType) (io.ReadCloser, error) {
	out, err := c.client.StreamRawBytes(context.Background(), &pb.StreamRawBytesRequest{
		ContainerId: containerID,
		Since:       timestamppb.New(since),
		Until:       timestamppb.New(until),
		StreamTypes: int32(std),
	})

	if err != nil {
		return nil, err
	}

	r, w := io.Pipe()

	go func() {
		defer w.Close()
		for {
			resp, err := out.Recv()
			err = rpcErrToErr(err)
			if err != nil {
				if err == io.EOF || err == context.Canceled {
					return
				} else {
					log.Warnf("error while streaming raw bytes %v", err)
					return
				}
			}

			w.Write(resp.Data)
		}
	}()

	return r, nil
}

func (c *Client) StreamStats(ctx context.Context, stats chan<- docker.ContainerStat) error {
	stream, err := c.client.StreamStats(ctx, &pb.StreamStatsRequest{})
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return rpcErrToErr(err)
		}

		stats <- docker.ContainerStat{
			CPUPercent:    resp.Stat.CpuPercent,
			MemoryPercent: resp.Stat.MemoryPercent,
			MemoryUsage:   resp.Stat.MemoryUsage,
			ID:            resp.Stat.Id,
		}
	}
}

func (c *Client) StreamEvents(ctx context.Context, events chan<- docker.ContainerEvent) error {
	stream, err := c.client.StreamEvents(ctx, &pb.StreamEventsRequest{})
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return rpcErrToErr(err)
		}

		events <- docker.ContainerEvent{
			ActorID: resp.Event.ActorId,
			Name:    resp.Event.Name,
			Host:    resp.Event.Host,
		}
	}
}

func (c *Client) StreamNewContainers(ctx context.Context, containers chan<- docker.Container) error {
	stream, err := c.client.StreamContainerStarted(ctx, &pb.StreamContainerStartedRequest{})
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return rpcErrToErr(err)
		}

		started := resp.Container.Started.AsTime()

		containers <- docker.Container{
			ID:        resp.Container.Id,
			Name:      resp.Container.Name,
			Image:     resp.Container.Image,
			Labels:    resp.Container.Labels,
			Group:     resp.Container.Group,
			ImageID:   resp.Container.ImageId,
			Created:   resp.Container.Created.AsTime(),
			State:     resp.Container.State,
			Status:    resp.Container.Status,
			Health:    resp.Container.Health,
			Host:      resp.Container.Host,
			Tty:       resp.Container.Tty,
			StartedAt: &started,
			Command:   resp.Container.Command,
		}
	}
}

func (c *Client) FindContainer(containerID string) (docker.Container, error) {
	response, err := c.client.FindContainer(context.Background(), &pb.FindContainerRequest{ContainerId: containerID})
	if err != nil {
		return docker.Container{}, err
	}

	var stats []docker.ContainerStat

	for _, stat := range response.Container.Stats {
		stats = append(stats, docker.ContainerStat{
			ID:            stat.Id,
			CPUPercent:    stat.CpuPercent,
			MemoryPercent: stat.MemoryPercent,
			MemoryUsage:   stat.MemoryUsage,
		})
	}

	var startedAt *time.Time
	if response.Container.Started != nil {
		started := response.Container.Started.AsTime()
		startedAt = &started
	}

	return docker.Container{
		ID:        response.Container.Id,
		Name:      response.Container.Name,
		Image:     response.Container.Image,
		Labels:    response.Container.Labels,
		Group:     response.Container.Group,
		ImageID:   response.Container.ImageId,
		Created:   response.Container.Created.AsTime(),
		State:     response.Container.State,
		Status:    response.Container.Status,
		Health:    response.Container.Health,
		Host:      response.Container.Host,
		Tty:       response.Container.Tty,
		Command:   response.Container.Command,
		Stats:     utils.RingBufferFrom(300, stats),
		StartedAt: startedAt,
	}, nil
}

func (c *Client) ListContainers() ([]docker.Container, error) {
	response, err := c.client.ListContainers(context.Background(), &pb.ListContainersRequest{})
	if err != nil {
		return nil, err
	}

	containers := make([]docker.Container, 0)
	for _, container := range response.Containers {
		var stats []docker.ContainerStat
		for _, stat := range container.Stats {
			stats = append(stats, docker.ContainerStat{
				ID:            stat.Id,
				CPUPercent:    stat.CpuPercent,
				MemoryPercent: stat.MemoryPercent,
				MemoryUsage:   stat.MemoryUsage,
			})
		}

		var startedAt *time.Time
		if container.Started != nil {
			started := container.Started.AsTime()
			startedAt = &started
		}

		containers = append(containers, docker.Container{
			ID:        container.Id,
			Name:      container.Name,
			Image:     container.Image,
			Labels:    container.Labels,
			Group:     container.Group,
			ImageID:   container.ImageId,
			Created:   container.Created.AsTime(),
			State:     container.State,
			Status:    container.Status,
			Health:    container.Health,
			Host:      container.Host,
			Tty:       container.Tty,
			Stats:     utils.RingBufferFrom(300, stats),
			Command:   container.Command,
			StartedAt: startedAt,
		})
	}

	return containers, nil
}

func (c *Client) Host() docker.Host {
	return c.host
}

func jsonBytesToOrderedMap(b []byte) *orderedmap.OrderedMap[string, any] {
	var data *orderedmap.OrderedMap[string, any]
	reader := bytes.NewReader(b)
	json.NewDecoder(reader).Decode(&data)
	return data
}
