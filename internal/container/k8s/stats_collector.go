package k8s

import (
	"context"
	"time"

	"github.com/amir20/dozzle/internal/container"
	lop "github.com/samber/lo/parallel"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

var timeToStop = 2 * time.Hour

type StatsCollector struct {
	client      *Client
	metrics     *metricsclient.Clientset
	subscribers *xsync.Map[context.Context, chan<- container.ContainerStat]
	lifecycle   container.CollectorLifecycle
	// metricsFailing keeps a missing metrics-server to one warning instead of one a
	// second. Per namespace, since RBAC can allow metrics in one and deny another.
	metricsFailing *xsync.Map[string, bool]
	labels         container.ContainerLabels
}

func NewStatsCollector(client *Client, labels container.ContainerLabels) (*StatsCollector, error) {
	metricsClient, err := metricsclient.NewForConfig(client.config)
	if err != nil {
		return nil, err
	}
	return &StatsCollector{
		subscribers:    xsync.NewMap[context.Context, chan<- container.ContainerStat](),
		metricsFailing: xsync.NewMap[string, bool](),
		client:         client,
		labels:         labels,
		metrics:        metricsClient,
	}, nil
}

func (c *StatsCollector) Subscribe(ctx context.Context, stats chan<- container.ContainerStat) {
	c.subscribers.Store(ctx, stats)
	go func() {
		<-ctx.Done()
		c.subscribers.Delete(ctx)
	}()
}

func (c *StatsCollector) Stop() {
	c.lifecycle.Release(timeToStop)
}

// Start starts the stats collector and blocks until it's stopped. It returns true if the collector was stopped, false if it was already running
func (sc *StatsCollector) Start(parentCtx context.Context) bool {
	ctx, run := sc.lifecycle.Acquire(parentCtx, timeToStop)
	if !run {
		return false
	}

	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ticker.C:
			lop.ForEach(sc.client.namespace, func(item string, index int) {
				metricList, err := sc.metrics.MetricsV1beta1().PodMetricses(item).List(ctx, metav1.ListOptions{})
				if err != nil {
					// Most often metrics-server is not installed. Logs work without it, so
					// warn once and keep polling in case it shows up.
					if _, failing := sc.metricsFailing.LoadOrStore(item, true); ctx.Err() == nil && !failing {
						log.Warn().Err(err).Str("namespace", item).Msg("could not read pod metrics, is metrics-server installed? CPU and memory will be empty")
					}
					return
				}
				if _, failing := sc.metricsFailing.LoadAndDelete(item); failing {
					log.Info().Str("namespace", item).Msg("pod metrics are available again")
				}
				for _, pod := range metricList.Items {
					for _, c := range pod.Containers {
						stat := container.ContainerStat{
							ID:             pod.Namespace + ":" + pod.Name + ":" + c.Name,
							CPUPercent:     float64(c.Usage.Cpu().MilliValue()) / 1000 * 100,
							MemoryUsage:    c.Usage.Memory().AsApproximateFloat64(),
							NetworkRxTotal: 0, // K8s metrics API doesn't expose network stats by default
							NetworkTxTotal: 0, // Would require custom metrics or cAdvisor integration
						}
						log.Trace().Interface("stat", stat).Msg("k8s stats")
						sc.subscribers.Range(func(c context.Context, stats chan<- container.ContainerStat) bool {
							select {
							case stats <- stat:
							case <-c.Done():
								sc.subscribers.Delete(c)
							}
							return true
						})
					}
				}
			})
		case <-ctx.Done():
			return true
		}
	}
}
