package k8s

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amir20/dozzle/internal/container"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
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
	// memoryLimits is what each container's memory percent is measured against: its
	// own limit, or its node's capacity without one, as Docker does. The metrics API
	// carries usage only, so pods and nodes are listed again every limitsRefresh.
	memoryLimits    map[string]float64
	limitsRefreshed time.Time

	// network is each pod's readings from the kubelets, keyed namespace:pod and
	// replaced whole on every read; see stats_network.go.
	network        atomic.Pointer[map[string]podNetwork]
	networkNext    atomic.Int64 // UnixNano when the next read is due
	networkBusy    atomic.Bool
	kubeletRefused atomic.Bool
}

const limitsRefresh = 30 * time.Second

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
			if time.Since(sc.limitsRefreshed) > limitsRefresh {
				sc.refreshMemoryLimits(ctx)
			}
			sc.maybeRefreshNetwork(ctx)
			now := time.Now()
			var wg sync.WaitGroup
			for _, item := range sc.client.namespace {
				wg.Go(func() {
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
						rx, tx := sc.podNetworkTotals(pod.Namespace, pod.Name, now)
						for i, c := range pod.Containers {
							// The pod's traffic goes on its first container only: a merged view
							// adds containers up, and a sidecar would count it twice.
							if i > 0 {
								rx, tx = 0, 0
							}
							id := pod.Namespace + ":" + pod.Name + ":" + c.Name
							memory := c.Usage.Memory().AsApproximateFloat64()
							var memoryPercent float64
							if limit := sc.memoryLimits[id]; limit > 0 {
								memoryPercent = memory / limit * 100
							}
							stat := container.ContainerStat{
								ID:             id,
								CPUPercent:     float64(c.Usage.Cpu().MilliValue()) / 1000 * 100,
								MemoryPercent:  memoryPercent,
								MemoryUsage:    memory,
								NetworkRxTotal: rx,
								NetworkTxTotal: tx,
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
			}
			wg.Wait()
		case <-ctx.Done():
			return true
		}
	}
}

// refreshMemoryLimits runs between ticks, so the pollers only ever read the map. A
// failed list keeps the previous limits rather than emptying every memory bar.
func (sc *StatsCollector) refreshMemoryLimits(ctx context.Context) {
	sc.limitsRefreshed = time.Now()
	// ResourceVersion "0" is served from the API server's cache, not etcd.
	nodes, err := sc.client.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{ResourceVersion: "0"})
	if err != nil {
		log.Debug().Err(err).Msg("could not list nodes for memory limits")
		return
	}
	nodeMemory := make(map[string]float64, len(nodes.Items))
	for _, node := range nodes.Items {
		nodeMemory[node.Name] = node.Status.Capacity.Memory().AsApproximateFloat64()
	}

	var pods []corev1.Pod
	for _, namespace := range sc.client.namespace {
		list, err := sc.client.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector:   sc.client.labelSelector,
			ResourceVersion: "0",
		})
		if err != nil {
			log.Debug().Err(err).Str("namespace", namespace).Msg("could not list pods for memory limits")
			return
		}
		pods = append(pods, list.Items...)
	}
	sc.memoryLimits = memoryLimits(pods, nodeMemory)
}

func memoryLimits(pods []corev1.Pod, nodeMemory map[string]float64) map[string]float64 {
	limits := make(map[string]float64)
	for _, pod := range pods {
		for _, c := range slices.Concat(pod.Spec.InitContainers, pod.Spec.Containers) {
			limit := c.Resources.Limits.Memory().AsApproximateFloat64()
			if limit == 0 {
				limit = nodeMemory[pod.Spec.NodeName]
			}
			limits[pod.Namespace+":"+pod.Name+":"+c.Name] = limit
		}
	}
	return limits
}
