package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"encoding/json"

	"github.com/amir20/dozzle/internal/agent/pb"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Client struct {
	client   pb.AgentServiceClient
	conn     *grpc.ClientConn
	endpoint string
}

func NewClient(endpoint string, certificates tls.Certificate, opts ...grpc.DialOption) (*Client, error) {
	caCertPool := x509.NewCertPool()
	c, err := x509.ParseCertificate(certificates.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	caCertPool.AddCert(c)
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{certificates},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true, // Set to true if the server's hostname does not match the certificate
	}

	// Create the gRPC transport credentials
	creds := credentials.NewTLS(tlsConfig)

	opts = append(opts,
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(10*1024*1024), grpc.UseCompressor(gzip.Name)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	conn, err := grpc.NewClient(endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", endpoint, err)
	}

	client := pb.NewAgentServiceClient(conn)

	return &Client{
		client:   client,
		conn:     conn,
		endpoint: endpoint,
	}, nil
}

func rpcErrToErr(err error) error {
	if err == nil {
		return nil
	}

	status, ok := status.FromError(err)
	if !ok {
		return err
	}

	if status.Code() == codes.Unknown && status.Message() == "EOF" {
		return fmt.Errorf("EOF error while converting gRPC to error: %w", io.EOF)
	}

	switch status.Code() {
	case codes.Canceled:
		return fmt.Errorf("canceled: %v with %w", status.Message(), context.Canceled)
	case codes.DeadlineExceeded:
		return fmt.Errorf("deadline exceeded: %v with %w", status.Message(), context.DeadlineExceeded)
	case codes.Unknown:
		return fmt.Errorf("unknown error: %v with %w", status.Message(), err)
	case codes.OK:
		return nil
	default:
		return fmt.Errorf("unknown code: %v with %w", status.Code(), err)
	}
}

func (c *Client) LogsBetweenDates(ctx context.Context, containerID string, since time.Time, until time.Time, std container.StdType) (<-chan *container.LogEvent, error) {
	stream, err := c.client.LogsBetweenDates(ctx, &pb.LogsBetweenDatesRequest{
		ContainerId: containerID,
		Since:       timestamppb.New(since),
		Until:       timestamppb.New(until),
		StreamTypes: int32(std),
	})

	if err != nil {
		return nil, err
	}

	events := make(chan *container.LogEvent)

	go func() {
		sendLogs(stream, events)
		close(events)
	}()

	return events, nil
}

func (c *Client) StreamContainerLogs(ctx context.Context, containerID string, since time.Time, std container.StdType, events chan<- *container.LogEvent) error {
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

func sendLogs(stream pb.AgentService_StreamLogsClient, events chan<- *container.LogEvent) error {
	for {
		resp, err := stream.Recv()
		if err != nil {
			return rpcErrToErr(err)
		}

		m, err := resp.Event.Message.UnmarshalNew()
		if err != nil {
			log.Error().Err(err).Msg("agent client: failed to unmarshal message")
			continue
		}

		var message any
		var logType container.LogType
		switch m := m.(type) {
		case *pb.SingleMessage:
			message = m.Message
			logType = container.LogTypeSingle

		case *pb.GroupMessage:
			fragments := make([]container.LogFragment, len(m.Fragments))
			for i, f := range m.Fragments {
				fragments[i] = container.LogFragment{
					Message: f.Message,
				}
			}
			message = fragments
			logType = container.LogTypeGroup

		case *pb.ComplexMessage:
			message = jsonBytesToOrderedMap(m.Data)
			logType = container.LogTypeComplex

		default:
			log.Error().Type("message", m).Msg("agent client: unknown message type")
			continue
		}

		events <- &container.LogEvent{
			Id:          resp.Event.Id,
			ContainerID: resp.Event.ContainerId,
			Message:     message,
			Type:        logType,
			Timestamp:   resp.Event.Timestamp.AsTime().Unix(),
			Level:       resp.Event.Level,
			Stream:      resp.Event.Stream,
			RawMessage:  resp.Event.RawMessage,
		}
	}
}

func (c *Client) StreamRawBytes(ctx context.Context, containerID string, since time.Time, until time.Time, std container.StdType) (io.ReadCloser, error) {
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
			if err != nil {
				err = rpcErrToErr(err)
				e := errors.Unwrap(err)
				if e == io.EOF || e == context.Canceled {
					return
				} else {
					log.Error().Err(err).Msg("agent client: failed to receive raw bytes")
					return
				}
			}

			w.Write(resp.Data)
		}
	}()

	return r, nil
}

func (c *Client) StreamStats(ctx context.Context, stats chan<- container.ContainerStat) error {
	stream, err := c.client.StreamStats(ctx, &pb.StreamStatsRequest{})
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return rpcErrToErr(err)
		}

		stats <- container.ContainerStat{
			CPUPercent:    resp.Stat.CpuPercent,
			MemoryPercent: resp.Stat.MemoryPercent,
			MemoryUsage:   resp.Stat.MemoryUsage,
			ID:            resp.Stat.Id,
		}
	}
}

func (c *Client) StreamEvents(ctx context.Context, events chan<- container.ContainerEvent) error {
	stream, err := c.client.StreamEvents(ctx, &pb.StreamEventsRequest{})
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return rpcErrToErr(err)
		}

		events <- container.ContainerEvent{
			ActorID: resp.Event.ActorId,
			Name:    resp.Event.Name,
			Host:    resp.Event.Host,
			Time:    resp.Event.Timestamp.AsTime(),
		}
	}
}

func (c *Client) StreamNewContainers(ctx context.Context, containers chan<- container.Container) error {
	stream, err := c.client.StreamContainerStarted(ctx, &pb.StreamContainerStartedRequest{})
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return rpcErrToErr(err)
		}

		containers <- container.FromProto(resp.Container)
	}
}

func (c *Client) FindContainer(ctx context.Context, containerID string, labels container.ContainerLabels) (container.Container, error) {
	in := &pb.FindContainerRequest{ContainerId: containerID}

	if labels != nil {
		in.Filter = make(map[string]*pb.RepeatedString)
		for k, v := range labels {
			in.Filter[k] = &pb.RepeatedString{Values: v}
		}
	}

	response, err := c.client.FindContainer(ctx, in)
	if err != nil {
		return container.Container{}, err
	}

	return container.FromProto(response.Container), nil
}

func (c *Client) ListContainers(ctx context.Context, labels container.ContainerLabels) ([]container.Container, error) {
	in := &pb.ListContainersRequest{}

	if labels != nil {
		in.Filter = make(map[string]*pb.RepeatedString)
		for k, v := range labels {
			in.Filter[k] = &pb.RepeatedString{Values: v}
		}
	}

	response, err := c.client.ListContainers(ctx, in)
	if err != nil {
		return nil, err
	}

	containers := make([]container.Container, 0)
	for _, c := range response.Containers {
		containers = append(containers, container.FromProto(c))
	}

	return containers, nil
}

func (c *Client) Host(ctx context.Context) (container.Host, error) {
	info, err := c.client.HostInfo(ctx, &pb.HostInfoRequest{})
	if err != nil {
		return container.Host{
			Endpoint:  c.endpoint,
			Type:      "agent",
			Available: false,
		}, err
	}

	return container.Host{
		ID:            info.Host.Id,
		Name:          info.Host.Name,
		NCPU:          int(info.Host.CpuCores),
		MemTotal:      int64(info.Host.Memory),
		Endpoint:      c.endpoint,
		Type:          "agent",
		DockerVersion: info.Host.DockerVersion,
		AgentVersion:  info.Host.AgentVersion,
	}, nil
}

func (c *Client) ContainerAction(ctx context.Context, containerId string, action container.ContainerAction) error {
	var containerAction pb.ContainerAction
	switch action {
	case container.Start:
		containerAction = pb.ContainerAction_Start

	case container.Stop:
		containerAction = pb.ContainerAction_Stop

	case container.Restart:
		containerAction = pb.ContainerAction_Restart

	}

	_, err := c.client.ContainerAction(ctx, &pb.ContainerActionRequest{ContainerId: containerId, Action: containerAction})

	return err
}

func (c *Client) ContainerAttach(ctx context.Context, containerId string) (*container.ExecSession, error) {
	stream, err := c.client.ContainerAttach(ctx)
	if err != nil {
		return nil, err
	}

	if err = stream.Send(&pb.ContainerAttachRequest{
		ContainerId: containerId,
	}); err != nil {
		return nil, err
	}
	stdoutReader, stdoutWriter := io.Pipe()
	stdinReader, stdinWriter := io.Pipe()

	go func() {
		defer stdoutWriter.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := stream.Recv()
				if err != nil {
					return
				}

				stdoutWriter.Write(msg.Stdout)
			}
		}
	}()

	go func() {
		defer stdinReader.Close()
		buffer := make([]byte, 1024)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := stdinReader.Read(buffer)
				if err != nil {
					return
				}

				if err := stream.Send(&pb.ContainerAttachRequest{
					Payload: &pb.ContainerAttachRequest_Stdin{
						Stdin: buffer[:n],
					},
				}); err != nil {
					return
				}
			}
		}
	}()

	// Create resize closure that sends via gRPC
	resizeFn := func(width uint, height uint) error {
		return stream.Send(&pb.ContainerAttachRequest{
			Payload: &pb.ContainerAttachRequest_Resize{
				Resize: &pb.ResizePayload{
					Width:  uint32(width),
					Height: uint32(height),
				},
			},
		})
	}

	return &container.ExecSession{
		Writer: stdinWriter,
		Reader: stdoutReader,
		Resize: resizeFn,
	}, nil
}

func (c *Client) Exec(ctx context.Context, containerId string, cmd []string, events container.ExecEventReader, stdout io.Writer) error {
	stream, err := c.client.ContainerExec(ctx)
	if err != nil {
		return err
	}

	if err = stream.Send(&pb.ContainerExecRequest{
		ContainerId: containerId,
		Command:     cmd,
	}); err != nil {
		return err
	}

	var wg sync.WaitGroup

	// Read from gRPC stream and write to stdout
	wg.Go(func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				return
			}
			stdout.Write(msg.Stdout)
		}
	})

	// Read events and convert to gRPC messages
	wg.Go(func() {
		for {
			event, err := events.ReadEvent()
			if err != nil {
				return
			}

			switch event.Type {
			case "userinput":
				stream.Send(&pb.ContainerExecRequest{
					Payload: &pb.ContainerExecRequest_Stdin{
						Stdin: []byte(event.Data),
					},
				})
			case "resize":
				stream.Send(&pb.ContainerExecRequest{
					Payload: &pb.ContainerExecRequest_Resize{
						Resize: &pb.ResizePayload{
							Width:  uint32(event.Width),
							Height: uint32(event.Height),
						},
					},
				})
			}
		}
	})

	wg.Wait()
	return nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) UpdateNotificationConfig(ctx context.Context, subscriptions []types.SubscriptionConfig, dispatchers []types.DispatcherConfig) error {
	// Convert to proto
	pbSubs := make([]*pb.NotificationSubscription, len(subscriptions))
	for i, sub := range subscriptions {
		pbSubs[i] = &pb.NotificationSubscription{
			Id:                  int32(sub.ID),
			Name:                sub.Name,
			Enabled:             sub.Enabled,
			DispatcherId:        int32(sub.DispatcherID),
			LogExpression:       sub.LogExpression,
			ContainerExpression: sub.ContainerExpression,
		}
	}

	pbDispatchers := make([]*pb.NotificationDispatcher, len(dispatchers))
	for i, d := range dispatchers {
		pbDispatchers[i] = &pb.NotificationDispatcher{
			Id:       int32(d.ID),
			Name:     d.Name,
			Type:     d.Type,
			Url:      d.URL,
			Template: d.Template,
		}
	}

	_, err := c.client.UpdateNotificationConfig(ctx, &pb.UpdateNotificationConfigRequest{
		Subscriptions: pbSubs,
		Dispatchers:   pbDispatchers,
	})

	return err
}

func jsonBytesToOrderedMap(b []byte) *orderedmap.OrderedMap[string, any] {
	var data *orderedmap.OrderedMap[string, any]
	reader := bytes.NewReader(b)
	json.NewDecoder(reader).Decode(&data)
	return data
}
