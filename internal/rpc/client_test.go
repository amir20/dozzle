package rpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	assert.NotNil(t, client, "Client should not be nil")
}

func TestFindContainer(t *testing.T) {
	client := NewClient()
	container, err := client.FindContainer("57dbe50682eb")
	assert.Nil(t, err, "Error should be nil")
	assert.NotNil(t, container, "Container should not be nil")
	assert.Equal(t, "57dbe50682eb", container.ID, "Container ID should be 57dbe50682eb")
}



func TestStreamLogs(t *testing.T) {
	client := NewClient()
	_, err := client.ContainerLogs("57dbe50682eb", nil, "stdout")
	assert.Nil(t, err, "Error should be nil")
}
