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
	// ImageDigest is k8s only, and k8s never runs behind an agent.
	// Protobuf hands an empty slice or map back as nil, which assert.Equal reads as a
	// difference, so at least one element keeps a zero-volume roll from failing at random.
	faker.FakeData(&expected,
		options.WithFieldsToIgnore("Stats", "MountStats", "ImageDigest"),
		options.WithRandomMapAndSliceMinSize(1),
	)
	expected.FinishedAt = expected.FinishedAt.UTC()
	expected.Created = expected.Created.UTC()
	expected.StartedAt = expected.StartedAt.UTC()
	expected.Stats = utils.NewRingBuffer[container.ContainerStat](300)

	pb := containerToProto(expected)
	actual := containerFromProto(&pb)

	assert.Equal(t, expected, actual)

	expected.SizeRw = nil
	pb = containerToProto(expected)
	assert.Nil(t, containerFromProto(&pb).SizeRw, "a container not measured yet stays unknown, not 0")
}
