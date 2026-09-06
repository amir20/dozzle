package web

import (
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/amir20/dozzle/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func compiledSub(t *testing.T, sub *notification.Subscription) *notification.Subscription {
	t.Helper()
	require.NoError(t, sub.CompileExpressions())
	return sub
}

func containerWithStats(id, name string, cpu ...float64) container.Container {
	stats := utils.NewRingBuffer[container.ContainerStat](300)
	for _, c := range cpu {
		stats.Push(container.ContainerStat{CPUPercent: c, MemoryPercent: c / 2})
	}
	return container.Container{ID: id, Name: name, State: "running", Host: "localhost", Stats: stats}
}

func Test_previewEventSamples_marks_matching_events(t *testing.T) {
	sub := compiledSub(t, &notification.Subscription{
		ContainerExpression: "true",
		EventExpression:     `name == "health_status" && attributes["healthStatus"] == "unhealthy"`,
	})

	samples := previewEventSamples(sub)
	require.Len(t, samples, len(previewEventCatalog))

	matched := make([]string, 0)
	for _, s := range samples {
		if s.Matches {
			matched = append(matched, s.Name+":"+s.Attributes["healthStatus"])
		}
	}
	assert.Equal(t, []string{"health_status:unhealthy"}, matched)
}

func Test_previewEventSamples_handles_events_without_attributes(t *testing.T) {
	// `attributes["exitCode"]` must not error on start/restart, which carry no attributes.
	sub := compiledSub(t, &notification.Subscription{
		ContainerExpression: "true",
		EventExpression:     `attributes["exitCode"] == "1"`,
	})

	samples := previewEventSamples(sub)
	matched := 0
	for _, s := range samples {
		if s.Matches {
			matched++
		}
	}
	assert.Equal(t, 1, matched, "only die with exitCode 1 should match")
}

func Test_previewEventSamples_nil_without_expression(t *testing.T) {
	assert.Nil(t, previewEventSamples(compiledSub(t, &notification.Subscription{ContainerExpression: "true"})))
}

func Test_previewMetricSamples_reports_window_ratio(t *testing.T) {
	sub := compiledSub(t, &notification.Subscription{
		ContainerExpression: "true",
		MetricExpression:    "cpu > 80",
	})

	// 5 samples, 4 over the threshold: 80% of a full window, which is exactly what fires.
	hot := containerWithStats("a", "hot", 90, 90, 90, 10, 95)
	// 5 samples, 1 over: matches right now but nowhere near the window threshold.
	spiky := containerWithStats("b", "spiky", 10, 10, 10, 10, 95)
	cold := containerWithStats("c", "cold", 1, 2, 3, 4, 5)

	samples := previewMetricSamples(sub, []container.Container{cold, spiky, hot}, 5)
	require.Len(t, samples, 3)

	// Containers closest to firing sort first.
	assert.Equal(t, "hot", samples[0].Name)
	assert.True(t, samples[0].WouldTrigger)
	assert.Equal(t, 4, samples[0].MatchedSamples)
	assert.Equal(t, 5, samples[0].TotalSamples)
	assert.InDelta(t, 95, samples[0].CPU, 0.001, "cpu should be the latest sample")

	assert.Equal(t, "spiky", samples[1].Name)
	assert.True(t, samples[1].Matches, "latest sample is over the threshold")
	assert.False(t, samples[1].WouldTrigger, "one spike is not a full window")

	assert.Equal(t, "cold", samples[2].Name)
	assert.False(t, samples[2].Matches)
}

func Test_previewMetricSamples_needs_a_full_window(t *testing.T) {
	sub := compiledSub(t, &notification.Subscription{
		ContainerExpression: "true",
		MetricExpression:    "cpu > 80",
	})

	// Every sample matches, but there are fewer of them than the window asks for.
	samples := previewMetricSamples(sub, []container.Container{containerWithStats("a", "hot", 90, 90)}, 15)
	require.Len(t, samples, 1)
	assert.True(t, samples[0].Matches)
	assert.False(t, samples[0].WouldTrigger)
	assert.Equal(t, 2, samples[0].TotalSamples)
}

func Test_previewMetricSamples_handles_containers_without_stats(t *testing.T) {
	sub := compiledSub(t, &notification.Subscription{
		ContainerExpression: "true",
		MetricExpression:    "cpu > 80",
	})

	samples := previewMetricSamples(sub, []container.Container{
		{ID: "a", Name: "no-stats", State: "running"},
		containerWithStats("b", "empty"),
	}, 15)

	require.Len(t, samples, 2)
	for _, s := range samples {
		assert.Zero(t, s.TotalSamples)
		assert.False(t, s.WouldTrigger)
	}
}

func Test_previewMetricSamples_nil_without_expression(t *testing.T) {
	sub := compiledSub(t, &notification.Subscription{ContainerExpression: "true"})
	assert.Nil(t, previewMetricSamples(sub, []container.Container{containerWithStats("a", "x", 5)}, 15))
}
