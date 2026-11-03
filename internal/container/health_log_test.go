package container

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type healthLogClient struct {
	ClientService
}

func (healthLogClient) HealthLog(_ context.Context, c Container) ([]HealthProbe, error) {
	return []HealthProbe{{ExitCode: 1, Output: c.ID}}, nil
}

func TestContainerServiceHealthLog(t *testing.T) {
	probes, err := NewContainerService(healthLogClient{}, Container{ID: "abc"}).HealthLog(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []HealthProbe{{ExitCode: 1, Output: "abc"}}, probes)

	_, err = NewContainerService(nil, Container{ID: "abc"}).HealthLog(context.Background())
	assert.ErrorIs(t, err, errors.ErrUnsupported)
}
