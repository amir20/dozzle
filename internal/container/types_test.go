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
	faker.FakeData(&expected, options.WithFieldsToIgnore("Stats"))
	expected.FinishedAt = expected.FinishedAt.UTC()
	expected.Created = expected.Created.UTC()
	expected.StartedAt = expected.StartedAt.UTC()
	expected.Stats = utils.NewRingBuffer[ContainerStat](300)

	pb := expected.ToProto()
	actual := FromProto(&pb)

	assert.Equal(t, expected, actual)

}
