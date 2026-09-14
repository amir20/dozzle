package agent

import (
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/utils"
	"github.com/go-faker/faker/v4"
	"github.com/go-faker/faker/v4/pkg/options"
	"github.com/stretchr/testify/assert"
)

func TestContainerProtoRoundTrip(t *testing.T) {
	expected := container.Container{}
	faker.FakeData(&expected, options.WithFieldsToIgnore("Stats", "MountStats"))
	expected.FinishedAt = expected.FinishedAt.UTC()
	expected.Created = expected.Created.UTC()
	expected.StartedAt = expected.StartedAt.UTC()
	expected.Stats = utils.NewRingBuffer[container.ContainerStat](300)

	pb := containerToProto(expected)
	actual := containerFromProto(&pb)

	assert.Equal(t, expected, actual)
}
