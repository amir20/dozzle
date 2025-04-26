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

type K8sStatsCollector struct {
	client       *K8sClient
	metrics      *metricsclient.Clientset
	subscribers  *xsync.Map[context.Context, chan<- container.ContainerStat]
	stopper      context.CancelFunc
	timer        *time.Timer
	mu           sync.Mutex
	totalStarted atomic.Int32
	labels       container.ContainerLabels
}

func NewK8sStatsCollector(client *K8sClient, labels container.ContainerLabels) (*K8sStatsCollector, error) {
	metricsClient, err := metricsclient.NewForConfig(client.config)
	if err != nil {
		return nil, err
	}
	return &K8sStatsCollector{
		subscribers: xsync.NewMap[context.Context, chan<- container.ContainerStat](),
		client:      client,
		labels:      labels,
		metrics:     metricsClient,
	}, nil
}

func (c *K8sStatsCollector) Subscribe(ctx context.Context, stats chan<- container.ContainerStat) {
	c.subscribers.Store(ctx, stats)
	go func() {
		<-ctx.Done()
		c.subscribers.Delete(ctx)
	}()
}

func (c *K8sStatsCollector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.totalStarted.Add(-1) == 0 {
		c.timer = time.AfterFunc(timeToStop, func() {
			c.forceStop()
		})
	}
}

func (c *K8sStatsCollector) forceStop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopper != nil {
		c.stopper()
		c.stopper = nil
		log.Debug().Msg("stopped container k8s stats collector")
	}
}

func (c *K8sStatsCollector) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.timer != nil {
		c.timer.Stop()
	}
	c.timer = nil
}

// Start starts the stats collector and blocks until it's stopped. It returns true if the collector was stopped, false if it was already running
func (sc *K8sStatsCollector) Start(parentCtx context.Context) bool {
	sc.reset()
	sc.totalStarted.Add(1)

	sc.mu.Lock()
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
					log.Panic().Err(err).Msg("failed to get pod metrics")
				}
				for _, pod := range metricList.Items {
					for _, c := range pod.Containers {
						stat := container.ContainerStat{
							ID:          pod.Namespace + ":" + pod.Name + ":" + c.Name,
							CPUPercent:  float64(c.Usage.Cpu().MilliValue()) / 1000 * 100,
							MemoryUsage: c.Usage.Memory().AsApproximateFloat64(),
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
