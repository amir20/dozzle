package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	docker_types "github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// dozzleUpdate is e made by a Dozzle update, the only kind the checkbox covers.
func dozzleUpdate(e container.ContainerUpdateEvent) container.ContainerUpdateEvent {
	e.Source = container.UpdateSourceDozzle
	return e
}

func newTestWatchList() (*updateWatchList, *time.Time) {
	now := time.Date(2026, 10, 3, 3, 0, 0, 0, time.UTC)
	return &updateWatchList{at: make(map[string]time.Time), now: func() time.Time { return now }}, &now
}

func TestUpdateWatch_MatchesTheReplacedContainer(t *testing.T) {
	w, _ := newTestWatchList()
	// A request may carry the full id; events carry the store's 12 characters.
	w.set("nas", "abcdef1234567890", true)

	assert.True(t, w.watched(dozzleUpdate(container.ContainerUpdateEvent{Host: "nas", OldID: "abcdef123456", NewID: "999999999999"})))
	assert.False(t, w.watched(dozzleUpdate(container.ContainerUpdateEvent{Host: "other", OldID: "abcdef123456"})))
	assert.False(t, w.watched(dozzleUpdate(container.ContainerUpdateEvent{Host: "nas", OldID: "999999999999"})))
}

// Each update records its own choice: an unticked retry of a container whose
// watched update rolled back is not watched.
func TestUpdateWatch_UntickedRetryClears(t *testing.T) {
	w, _ := newTestWatchList()
	w.set("nas", "abcdef123456", true)
	w.set("nas", "abcdef123456", false)
	assert.False(t, w.watched(dozzleUpdate(container.ContainerUpdateEvent{Host: "nas", OldID: "abcdef123456"})))
}

// The checkbox covers the update it was ticked for. Watchtower or compose
// replacing the same container later is not that update.
func TestUpdateWatch_OnlyDozzleUpdates(t *testing.T) {
	w, _ := newTestWatchList()
	w.set("nas", "abcdef123456", true)
	for _, source := range []string{container.UpdateSourceWatchtower, container.UpdateSourceExternal, container.UpdateSourceSchedule, container.UpdateSourceCloud, ""} {
		assert.False(t, w.watched(container.ContainerUpdateEvent{Host: "nas", OldID: "abcdef123456", Source: source}), source)
	}
	assert.True(t, w.watched(dozzleUpdate(container.ContainerUpdateEvent{Host: "nas", OldID: "abcdef123456"})))
}

// An update that ends without a swap produces no event, so its choice is
// dropped rather than left waiting under a container that is still running.
func TestUpdateWatch_SettleDropsUpdatesWithoutASwap(t *testing.T) {
	for _, status := range []string{container.UpdateUpToDate, "error", "pulling", ""} {
		w, _ := newTestWatchList()
		w.set("nas", "abcdef123456", true)
		w.settle("nas", "abcdef1234567890", status)
		assert.False(t, w.watched(dozzleUpdate(container.ContainerUpdateEvent{Host: "nas", OldID: "abcdef123456"})), status)
	}
	for _, status := range []string{container.UpdateDone, container.UpdateRolledBack} {
		w, _ := newTestWatchList()
		w.set("nas", "abcdef123456", true)
		w.settle("nas", "abcdef123456", status)
		assert.True(t, w.watched(dozzleUpdate(container.ContainerUpdateEvent{Host: "nas", OldID: "abcdef123456"})), status)
	}
}

func TestUpdateWatch_Expires(t *testing.T) {
	w, now := newTestWatchList()
	w.set("nas", "abcdef123456", true)
	*now = now.Add(updateWatchTTL + time.Minute)
	assert.False(t, w.watched(dozzleUpdate(container.ContainerUpdateEvent{Host: "nas", OldID: "abcdef123456"})))
	// Expired choices are dropped on the next write.
	w.set("nas", "000000000000", false)
	assert.Empty(t, w.at)
}

// A bulk run refused as busy puts back the choices of the run that is busy.
func TestUpdateWatch_SetAllRestores(t *testing.T) {
	w, _ := newTestWatchList()
	w.set("nas", "aaaaaaaaaaaa", true)
	services := []*container.ContainerService{
		container.NewContainerService(nil, container.Container{Host: "nas", ID: "aaaaaaaaaaaa"}),
		container.NewContainerService(nil, container.Container{Host: "nas", ID: "bbbbbbbbbbbb"}),
	}

	restore := w.setAll(services, false)
	assert.False(t, w.watched(dozzleUpdate(container.ContainerUpdateEvent{Host: "nas", OldID: "aaaaaaaaaaaa"})))
	restore()
	assert.True(t, w.watched(dozzleUpdate(container.ContainerUpdateEvent{Host: "nas", OldID: "aaaaaaaaaaaa"})))
	assert.False(t, w.watched(dozzleUpdate(container.ContainerUpdateEvent{Host: "nas", OldID: "bbbbbbbbbbbb"})))

	restore = w.setAll(services, true)
	assert.True(t, w.watched(dozzleUpdate(container.ContainerUpdateEvent{Host: "nas", OldID: "bbbbbbbbbbbb"})))
	restore()
	assert.False(t, w.watched(dozzleUpdate(container.ContainerUpdateEvent{Host: "nas", OldID: "bbbbbbbbbbbb"})))
}

func Test_handler_containerUpdate_watchInCloud(t *testing.T) {
	t.Cleanup(func() { updateWatches.at = make(map[string]time.Time) })

	watched := func() bool {
		updateWatches.mu.Lock()
		defer updateWatches.mu.Unlock()
		for key := range updateWatches.at {
			if strings.HasSuffix(key, "/123") {
				return true
			}
		}
		return false
	}

	// duringUpdate is whether the choice was recorded while the update ran.
	var duringUpdate bool
	run := func(body string) {
		mockedClient := mockedClient()
		mockedClient.On("ContainerInspect", mock.Anything, "123").Return(docker_types.InspectResponse{
			Image:  "sha256:current",
			Config: &docker_types.Config{Image: "test:v1"},
		}, nil)
		mockedClient.On("ImagePull", mock.Anything, "test:v1").Run(func(mock.Arguments) { duringUpdate = watched() }).Return(io.NopCloser(strings.NewReader(`{"status":"Status: Image is up to date for test:v1"}`+"\n")), nil)
		mockedClient.On("ImageID", mock.Anything, "test:v1").Return("sha256:current", nil)

		handler := createHandler(mockedClient, nil, Config{Base: "/", EnableActions: true, Authorization: Authorization{Provider: NONE}})
		var reader io.Reader
		if body != "" {
			reader = strings.NewReader(body)
		}
		req, err := http.NewRequest("POST", "/api/hosts/localhost/containers/123/actions/update", reader)
		require.NoError(t, err)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		require.Equal(t, 200, rr.Code)
	}

	run(`{"watchInCloud":true}`)
	assert.True(t, duringUpdate)
	// The image was already current, so no event will come for the choice.
	assert.False(t, watched())
	// No body is an older UI or a script: not watched.
	run("")
	assert.False(t, duringUpdate)
}
