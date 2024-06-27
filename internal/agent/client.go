package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"time"

	"github.com/amir20/dozzle/internal/agent/pb"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/amir20/dozzle/internal/utils"
	log "github.com/sirupsen/logrus"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Client struct {
	client pb.AgentServiceClient
	host   docker.Host
}

func NewClient(endpoint string, certificates tls.Certificate) (*Client, error) {
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

	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(creds))
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

func (c *Client) StreamContainerLogs(ctx context.Context, containerID string, since time.Time, until time.Time, std docker.StdType, events chan<- *docker.LogEvent) error {
	stream, err := c.client.StreamLogs(ctx, &pb.StreamLogsRequest{
		ContainerId: containerID,
		Since:       timestamppb.New(since),
		StreamTypes: int32(std),
	})

	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return err
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
		}
	}
}

func jsonBytesToOrderedMap(b []byte) *orderedmap.OrderedMap[string, any] {
	var data *orderedmap.OrderedMap[string, any]
	reader := bytes.NewReader(b)
	json.NewDecoder(reader).Decode(&data)
	return data
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
			status, ok := status.FromError(err)
			if status.Message() == "EOF" || ok {
				return
			} else {
				log.Errorf("cannot unpack message %v", err)
			}

			w.Write(resp.Data)
		}
	}()

	return r, nil
}

func (c *Client) FindContainer(containerID string) (docker.Container, error) {
	response, err := c.client.FindContainer(context.Background(), &pb.FindContainerRequest{ContainerId: containerID})
	if err != nil {
		return docker.Container{}, err
	}

	// TODO: convert response to docker.Container
	return docker.Container{
		ID:      response.Container.Id,
		Name:    response.Container.Name,
		Image:   response.Container.Image,
		Labels:  response.Container.Labels,
		Group:   response.Container.Group,
		ImageID: response.Container.ImageId,
		Created: response.Container.Created.AsTime(),
		State:   response.Container.State,
		Status:  response.Container.Status,
		Health:  response.Container.Health,
		Host:    response.Container.Host,
		Tty:     response.Container.Tty,
		Stats:   nil,
	}, nil
}

func (c *Client) ListContainers() ([]docker.Container, error) {
	response, err := c.client.ListContainers(context.Background(), &pb.ListContainersRequest{})
	if err != nil {
		return nil, err
	}

	containers := make([]docker.Container, 0)
	for _, container := range response.Containers {
		containers = append(containers, docker.Container{
			ID:      container.Id,
			Name:    container.Name,
			Image:   container.Image,
			Labels:  container.Labels,
			Group:   container.Group,
			ImageID: container.ImageId,
			Created: container.Created.AsTime(),
			State:   container.State,
			Status:  container.Status,
			Health:  container.Health,
			Host:    container.Host,
			Tty:     container.Tty,
			Stats:   utils.NewRingBuffer[docker.ContainerStat](300),
		})
	}

	return containers, nil
}

func (c *Client) Host() docker.Host {
	return c.host
}
