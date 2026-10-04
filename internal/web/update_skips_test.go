package web

import (
	"context"
	"os"
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// checkingClientService answers image checks with a fixed result per container.
type checkingClientService struct {
	container.ClientService
	results map[string]imagecheck.Result
}

func (s *checkingClientService) CheckImageUpdate(_ context.Context, c container.Container, _ bool) (imagecheck.Result, error) {
	return s.results[c.ID], nil
}

// labelledHosts lists containers and finds them on one client service.
type labelledHosts struct {
	HostService
	containers []container.Container
	client     container.ClientService
}

func (h *labelledHosts) ListAllContainers(container.ContainerLabels) ([]container.Container, []error) {
	return h.containers, nil
}

func (h *labelledHosts) FindContainer(_ string, id string, _ container.ContainerLabels) (*container.ContainerService, error) {
	for _, c := range h.containers {
		if c.ID == id {
			return container.NewContainerService(h.client, c), nil
		}
	}
	return nil, container.ErrContainerNotFound
}

func outdatedNames(s *autoUpdateScheduler) []string {
	outdated, _ := s.outdatedAutoContainers(context.Background())
	var names []string
	for _, cs := range outdated {
		names = append(names, cs.Container.Name)
	}
	return names
}

// A container someone rolled back keeps being offered the image its tag still
// names. The schedule leaves it until a newer image is pushed.
func TestAutoUpdate_SkipsImageRolledBackFrom(t *testing.T) {
	setupTestEnv(t, true)
	labels := map[string]string{AutoUpdateLabel: "true"}
	immich := container.Container{ID: "aaa", Name: "immich", Host: "nas", Image: "immich:release", State: "running", Labels: labels, ImageDigest: "immich@sha256:broken"}
	redis := container.Container{ID: "bbb", Name: "redis", Host: "nas", Image: "redis:7", State: "running", Labels: labels}
	client := &checkingClientService{results: map[string]imagecheck.Result{
		"aaa": {Status: imagecheck.StatusUpdateAvailable, RemoteDigest: "sha256:broken"},
		"bbb": {Status: imagecheck.StatusUpdateAvailable, RemoteDigest: "sha256:redis"},
	}}
	s := &autoUpdateScheduler{config: &serverActions, hostService: &labelledHosts{containers: []container.Container{immich, redis}, client: client}}
	require.ElementsMatch(t, []string{"immich", "redis"}, outdatedNames(s))

	// Rolled back from sha256:broken, which immich's tag still names.
	RecordRolledBack(immich)
	assert.Equal(t, []string{"redis"}, outdatedNames(s))
	assert.Equal(t, []string{"redis"}, outdatedNames(s), "on every night after, too")

	client.results["aaa"] = imagecheck.Result{Status: imagecheck.StatusUpdateAvailable, RemoteDigest: "sha256:fixed"}
	assert.ElementsMatch(t, []string{"immich", "redis"}, outdatedNames(s), "a newer image is applied")
	client.results["aaa"] = imagecheck.Result{Status: imagecheck.StatusUpdateAvailable, RemoteDigest: "sha256:broken"}
	assert.ElementsMatch(t, []string{"immich", "redis"}, outdatedNames(s), "and the skip is gone with it")
}

func TestAutoUpdate_UpToDateClearsSkip(t *testing.T) {
	setupTestEnv(t, true)
	c := container.Container{ID: "aaa", Name: "immich", Host: "nas", Image: "immich:release", State: "running", Labels: map[string]string{AutoUpdateLabel: "true"}, ImageDigest: "immich@sha256:broken"}
	client := &checkingClientService{results: map[string]imagecheck.Result{"aaa": {Status: imagecheck.StatusUpToDate}}}
	s := &autoUpdateScheduler{config: &serverActions, hostService: &labelledHosts{containers: []container.Container{c}, client: client}}

	RecordRolledBack(c)
	assert.Empty(t, outdatedNames(s))
	client.results["aaa"] = imagecheck.Result{Status: imagecheck.StatusUpdateAvailable, RemoteDigest: "sha256:broken"}
	assert.Equal(t, []string{"immich"}, outdatedNames(s), "updated by hand since, so the skip no longer applies")
}

// A locally built image has no digest to skip, and is never offered an update.
func TestRecordRolledBack_LocalImage(t *testing.T) {
	setupTestEnv(t, true)
	RecordRolledBack(container.Container{Name: "app", Host: "nas"})
	_, err := os.Stat(updateSkipsPath())
	assert.ErrorIs(t, err, os.ErrNotExist)
}

// An older Dozzle wrote its own skip to auto-update-attempt. Upgrading keeps it.
func TestUpdateSkips_ReadsLegacyAttemptFile(t *testing.T) {
	setupTestEnv(t, true)
	require.NoError(t, os.WriteFile(autoUpdateAttemptPath(), []byte("sha256:broken\n"), 0644))

	assert.True(t, skippedUpdate(selfSkipKey, "sha256:broken"))
	recordUpdateSkip("nas/immich", "immich@sha256:x")
	_, err := os.Stat(autoUpdateAttemptPath())
	assert.ErrorIs(t, err, os.ErrNotExist, "folded into the skip list")
	assert.True(t, skippedUpdate(selfSkipKey, "sha256:broken"))
	assert.True(t, skippedUpdate("nas/immich", "sha256:x"))

	clearUpdateSkip(selfSkipKey)
	clearUpdateSkip("nas/immich")
	_, err = os.Stat(updateSkipsPath())
	assert.ErrorIs(t, err, os.ErrNotExist, "an empty list leaves no file")
}
