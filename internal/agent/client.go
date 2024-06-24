package agent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"os"
	"time"

	"github.com/amir20/dozzle/internal/agent/pb"
	"github.com/amir20/dozzle/internal/docker"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Client struct {
	client pb.StreamServiceClient
}

func NewClient() *Client {

	cert, err := tls.LoadX509KeyPair("shared_cert.pem", "shared_key.pem")
	if err != nil {
		log.Fatalf("failed to load client certificate: %v", err)
	}

	// Load the CA certificate from disk
	caCert, err := os.ReadFile("shared_cert.pem")
	if err != nil {
		log.Fatalf("failed to read CA certificate: %v", err)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		log.Fatalf("failed to add CA certificate to pool")
	}

	// Create the TLS configuration
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true, // Set to true if the server's hostname does not match the certificate
	}

	// Create the gRPC transport credentials
	creds := credentials.NewTLS(tlsConfig)

	conn, err := grpc.NewClient("localhost:7007", grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)
	}

	client := pb.NewStreamServiceClient(conn)
	return &Client{client: client}
}

func (c *Client) StreamContainerLogs(ctx context.Context, containerID string, since time.Time, until time.Time, std docker.StdType, events chan<- *docker.LogEvent) error {
	stream, err := c.client.StreamLogs(ctx, &pb.StreamLogsRequest{
		ContainerId: containerID,
		Since:       timestamppb.New(since),
		Until:       timestamppb.New(until),
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
		default:
			log.Fatalf("unknown type %T", m)
		}

		events <- &docker.LogEvent{
			Id:          resp.Event.Id,
			ContainerID: resp.Event.ContainerId,
			Message:     message,
			Timestamp:   resp.Event.Timestamp.AsTime().Unix(),
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
			if err != nil {
				return
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
			Stats:   nil, // TODO: convert stats
		})
	}

	return containers, nil
}
