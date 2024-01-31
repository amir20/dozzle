package docker

import (
	"context"
)

type ContainerStore struct {
	containers     map[string]Container
	client         *Client
	StatsCollector *StatCollector
}

func NewContainerStore(client *Client) *ContainerStore {
	s := &ContainerStore{
		containers:     make(map[string]Container),
		client:         client,
		StatsCollector: NewStatCollector(client),
	}

	go s.init(context.Background())
	go s.StatsCollector.StartCollecting(context.Background())

	return s
}

func (s *ContainerStore) Get(id string) (Container, bool) {
	container, ok := s.containers[id]
	return container, ok
}

func (s *ContainerStore) List() []Container {
	containers := make([]Container, 0, len(s.containers))
	for _, c := range s.containers {
		containers = append(containers, c)
	}
	return containers
}

func (s *ContainerStore) Client() *Client {
	return s.client
}

func (s *ContainerStore) init(ctx context.Context) {
	containers, err := s.client.ListContainers()
	if err != nil {
		return
	}

	for _, c := range containers {
		s.containers[c.ID] = c
	}

	events := make(chan ContainerEvent)
	s.client.Events(ctx, events)

	stats := make(chan ContainerStat)
	s.StatsCollector.Subscribe(stats)
	defer s.StatsCollector.Unsubscribe(stats)

	for {
		select {
		case event := <-events:
			switch event.Name {
			case "start":
				if container, err := s.client.FindContainer(event.ActorID); err == nil {
					s.containers[container.ID] = container
				}
			case "die":
				delete(s.containers, event.ActorID)
			}
		case stat := <-stats:
			if container, ok := s.containers[stat.ID]; ok {
				container.Stats.Push(stat)

			}
		case <-ctx.Done():
			return
		}
	}
}
