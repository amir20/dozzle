package k8s

import (
	"context"
	"sync"
	"sync/atomic"
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
	client       *Client
	metrics      *metricsclient.Clientset
	subscribers  *xsync.Map[context.Context, chan<- container.ContainerStat]
	stopper      context.CancelFunc
	timer        *time.Timer
	mu           sync.Mutex
	totalStarted atomic.Int32
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
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.totalStarted.Add(-1) == 0 {
		c.timer = time.AfterFunc(timeToStop, func() {
			c.forceStop()
		})
	}
}

func (c *StatsCollector) forceStop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopper != nil {
		c.stopper()
		c.stopper = nil
		log.Debug().Msg("stopped container k8s stats collector")
	}
}

// Start starts the stats collector and blocks until it's stopped. It returns true if the collector was stopped, false if it was already running
func (sc *StatsCollector) Start(parentCtx context.Context) bool {
	sc.mu.Lock()
	if sc.timer != nil {
		sc.timer.Stop()
		sc.timer = nil
	}
	// Callers run Start and Stop in separate goroutines, so a subscriber whose
	// ctx is already done can Stop first. A count still <= 0 means that Stop
	// already ran: starting here would leave a collector nobody holds and no
	// timer to end it.
	if sc.totalStarted.Add(1) <= 0 {
		if sc.stopper != nil {
			sc.timer = time.AfterFunc(timeToStop, sc.forceStop)
		}
		sc.mu.Unlock()
		return false
	}
	if sc.stopper != nil {
		sc.mu.Unlock()
		return false
	}
	var ctx context.Context
	ctx, sc.stopper = context.WithCancel(parentCtx)
	sc.mu.Unlock()

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
