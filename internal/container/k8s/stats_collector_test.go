package k8s

import (
	"context"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/stretchr/testify/assert"
)

// newTestCollector builds a collector with no namespaces, so each tick polls
// nothing and the metrics client is never touched.
func newTestCollector() *StatsCollector {
	return &StatsCollector{
		client:         &Client{},
		subscribers:    xsync.NewMap[context.Context, chan<- container.ContainerStat](),
		metricsFailing: xsync.NewMap[string, bool](),
	}
}

// running reports whether a collector is live, read under the same lock Start
// and forceStop use.
func running(sc *StatsCollector) bool {
	return sc.lifecycle.Running()
}

func TestStopBeforeStartDoesNotLeakCollector(t *testing.T) {
	collector := newTestCollector()
	collector.Stop()

	done := make(chan bool)
	go func() { done <- collector.Start(t.Context()) }()

	select {
	case started := <-done:
		assert.False(t, started, "start after its own stop should not run a collector")
	case <-time.After(2 * time.Second):
		t.Fatal("Start blocked, so it started a collector nobody holds")
	}
	assert.False(t, running(collector), "no collector should be running")
	assert.Equal(t, 0, collector.lifecycle.Holders())
}

func TestStopTimerEndsStartedCollector(t *testing.T) {
	old := timeToStop
	timeToStop = 10 * time.Millisecond
	t.Cleanup(func() { timeToStop = old })

	collector := newTestCollector()
	done := make(chan bool)
	go func() { done <- collector.Start(t.Context()) }()

	assert.Eventually(t, func() bool { return running(collector) }, 2*time.Second, time.Millisecond)
	collector.Stop()

	select {
	case stopped := <-done:
		assert.True(t, stopped, "the stop timer should end the collector")
	case <-time.After(2 * time.Second):
		t.Fatal("stop timer never ended the collector")
	}
	assert.False(t, running(collector))
}
