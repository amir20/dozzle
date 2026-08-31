package container

import (
	"testing"

	"github.com/amir20/dozzle/internal/utils"
	"github.com/go-faker/faker/v4"
	"github.com/go-faker/faker/v4/pkg/options"
	"github.com/stretchr/testify/assert"
)

func TestProto(t *testing.T) {
	expected := Container{}
	faker.FakeData(&expected, options.WithFieldsToIgnore("Stats", "MountStats"))
	expected.FinishedAt = expected.FinishedAt.UTC()
	expected.Created = expected.Created.UTC()
	expected.StartedAt = expected.StartedAt.UTC()
	expected.Stats = utils.NewRingBuffer[ContainerStat](300)

	pb := expected.ToProto()
	actual := FromProto(&pb)

	assert.Equal(t, expected, actual)

}

// Empty repeated fields decode from proto as nil; FromProto must normalize
// them back so the round trip is lossless even when the slice is empty.
func TestProtoEmptyNetworks(t *testing.T) {
	expected := Container{
		Networks: []string{},
		Stats:    utils.NewRingBuffer[ContainerStat](300),
	}

	pb := expected.ToProto()
	actual := FromProto(&pb)

	assert.Equal(t, expected.Networks, actual.Networks)
}
