package container

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The UI shows ErrNotRunning translated by matching its text, so the two must
// not drift apart.
func TestErrNotRunningMatchesTheUI(t *testing.T) {
	source, err := os.ReadFile("../../assets/composable/containers/containerActions.ts")
	require.NoError(t, err)
	assert.Contains(t, string(source), "NOT_RUNNING_ERROR = "+strconv.Quote(ErrNotRunning.Error()))
}
