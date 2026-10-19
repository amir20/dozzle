package k8s

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	// networkRefresh is about how often a kubelet refreshes its own network numbers.
	networkRefresh = 10 * time.Second
	// kubeletRetry is how long a refused kubelet is left alone: RBAC or a certificate
	// changes rarely, and each attempt is one request per node.
	kubeletRetry = 5 * time.Minute
)

type netSample struct {
	rx, tx  uint64
	kubelet time.Time // the kubelet's own timestamp, to tell a new reading from a repeat
	at      time.Time // when Dozzle received it
}

// podNetwork is one pod's network counters. Every container in a pod shares its
// network namespace, so there is no per-container number to report.
type podNetwork struct {
	base, prev, last netSample
}

// observe takes a summary's reading, keeping the two most recent distinct ones.
func (p podNetwork) observe(s netSample) podNetwork {
	switch {
	case s.rx < p.last.rx || s.tx < p.last.tx:
		// the counters started over: a new sandbox under the same pod name
		return podNetwork{base: s, prev: s, last: s}
	case s.kubelet.After(p.last.kubelet):
		p.prev, p.last = p.last, s
	}
	return p
}

// totals are the pod's counters since Dozzle first saw it, as of one kubelet interval
// ago. The table turns consecutive one-second totals into a rate, so reporting the
// kubelet's numbers as they step every ~10s would read as zero and then a spike.
// Walking from the previous reading to the last one over the time between them keeps
// the rate smooth; counting from the first reading keeps the first one from reading
// as everything the pod ever sent.
func (p podNetwork) totals(now time.Time) (rx, tx uint64) {
	f := 1.0
	if span := p.last.at.Sub(p.prev.at); span > 0 {
		f = min(max(float64(now.Sub(p.last.at))/float64(span), 0), 1)
	}
	lerp := func(a, b uint64) uint64 { return a + uint64(f*float64(b-a)) }
	return lerp(p.prev.rx, p.last.rx) - p.base.rx, lerp(p.prev.tx, p.last.tx) - p.base.tx
}

// maybeRefreshNetwork starts a summary read when one is due and none is running. It
// runs beside the ticks, since a slow kubelet must not hold up CPU and memory.
func (sc *StatsCollector) maybeRefreshNetwork(ctx context.Context) {
	if time.Now().UnixNano() < sc.networkNext.Load() || !sc.networkBusy.CompareAndSwap(false, true) {
		return
	}
	sc.networkNext.Store(time.Now().Add(networkRefresh).UnixNano())
	go func() {
		defer sc.networkBusy.Store(false)
		sc.refreshNetwork(ctx)
	}()
}

func (sc *StatsCollector) refreshNetwork(ctx context.Context) {
	pods, complete, err := sc.client.kubeletStats(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		if errors.Is(err, errKubeletRefused) {
			sc.networkNext.Store(time.Now().Add(kubeletRetry).UnixNano())
			if !sc.kubeletRefused.Swap(true) {
				log.Info().Err(err).Msg("could not read kubelet stats, network and disk will be empty")
			}
			return
		}
		log.Debug().Err(err).Msg("could not read kubelet stats")
		return
	}
	if sc.kubeletRefused.Swap(false) {
		log.Info().Msg("kubelet stats are available again")
	}

	now := time.Now()
	var old map[string]podNetwork
	if p := sc.network.Load(); p != nil {
		old = *p
	}
	next := make(map[string]podNetwork, len(pods))
	for _, pod := range pods {
		n := pod.Network
		if n == nil || n.RxBytes == nil || n.TxBytes == nil {
			continue
		}
		key := pod.PodRef.Namespace + ":" + pod.PodRef.Name
		s := netSample{rx: *n.RxBytes, tx: *n.TxBytes, kubelet: n.Time.Time, at: now}
		if p, seen := old[key]; seen {
			next[key] = p.observe(s)
		} else {
			next[key] = podNetwork{base: s, prev: s, last: s}
		}
	}
	// A node that did not answer keeps its pods' readings, or their totals would drop
	// to zero and come back as one huge jump.
	if !complete {
		for key, p := range old {
			if _, ok := next[key]; !ok {
				next[key] = p
			}
		}
	}
	sc.network.Store(&next)
}

// podNetworkTotals are a pod's totals for this tick, zero before any reading.
func (sc *StatsCollector) podNetworkTotals(namespace, pod string, now time.Time) (rx, tx uint64) {
	p := sc.network.Load()
	if p == nil {
		return 0, 0
	}
	n, ok := (*p)[namespace+":"+pod]
	if !ok {
		return 0, 0
	}
	return n.totals(now)
}
