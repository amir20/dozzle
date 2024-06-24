package agent

import (
	"context"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/amir20/dozzle/internal/docker"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	assert.NotNil(t, client, "Client should not be nil")
}

func TestFindContainer(t *testing.T) {
	client := NewClient()
	container, err := client.FindContainer("57dbe50682eb")
	assert.Nil(t, err, "Error should be nil. Got: %v", err)
	assert.NotNil(t, container, "Container should not be nil")
	assert.Equal(t, "57dbe50682eb", container.ID, "Container ID should be 57dbe50682eb")
}

func TestStreamLogs(t *testing.T) {
	client := NewClient()
	events := make(chan docker.LogEvent)
	go func() {
		for event := range events {
			assert.NotNil(t, event, "Event should not be nil")
			log.Printf("Event: %+v", event)
		}
	}()

	err := client.ContainerLogs(context.Background(), "57dbe50682eb", nil, docker.STDALL, events)
	assert.Nil(t, err, "Error should be nil")

}
