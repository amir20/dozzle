package k8s

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// The kubelet's /stats/summary is where network and disk live: the metrics API only
// carries CPU and memory. It is read from each kubelet directly, the way
// metrics-server does, which needs only get on nodes/stats. Going through the API
// server's node proxy would need nodes/proxy, and that one allows exec into any
// container.

const kubeletTimeout = 5 * time.Second

// errKubeletRefused is a kubelet that will never answer this service account: RBAC
// denies nodes/stats, or its serving certificate is not signed by the cluster CA.
// Neither changes by retrying.
var errKubeletRefused = errors.New("kubelet refused stats")

// kubeletSummary is the part of /stats/summary Dozzle reads.
type kubeletSummary struct {
	Pods []kubeletPodStats `json:"pods"`
}

type kubeletPodStats struct {
	PodRef struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"podRef"`
	Network *struct {
		Time    metav1.Time `json:"time"`
		RxBytes *uint64     `json:"rxBytes"`
		TxBytes *uint64     `json:"txBytes"`
	} `json:"network"`
	Containers []struct {
		Name   string `json:"name"`
		Rootfs *struct {
			UsedBytes *uint64 `json:"usedBytes"`
		} `json:"rootfs"`
	} `json:"containers"`
	Volume []struct {
		Name      string  `json:"name"`
		UsedBytes *uint64 `json:"usedBytes"`
	} `json:"volume"`
}

// newKubeletHTTPClient authenticates to kubelets the way the client authenticates to
// the API server, and trusts the same CA. The API server's ServerName is dropped, since
// each kubelet presents its own certificate.
//
// A kubeconfig that skips TLS verification gets no client, so kubelet stats are off:
// a node writes its own addresses, and one could point them at a host that collects
// the credentials.
func newKubeletHTTPClient(config *rest.Config) (*http.Client, error) {
	if config.Insecure {
		return nil, nil
	}
	kubelet := rest.CopyConfig(config)
	kubelet.ServerName = ""
	kubelet.Timeout = kubeletTimeout
	return rest.HTTPClientFor(kubelet)
}

// kubeletURL is the node's stats endpoint at its internal address, falling back to
// any address it has.
func kubeletURL(node *corev1.Node) (string, bool) {
	var address string
	for _, a := range node.Status.Addresses {
		if a.Type == corev1.NodeInternalIP {
			address = a.Address
			break
		}
		if address == "" {
			address = a.Address
		}
	}
	if address == "" {
		return "", false
	}
	port := node.Status.DaemonEndpoints.KubeletEndpoint.Port
	if port == 0 {
		port = 10250
	}
	return "https://" + net.JoinHostPort(address, strconv.Itoa(int(port))) + "/stats/summary", true
}

func (k *Client) nodeSummary(ctx context.Context, node *corev1.Node) ([]kubeletPodStats, error) {
	url, ok := kubeletURL(node)
	if !ok {
		return nil, fmt.Errorf("node %s has no address", node.Name)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := k.kubelet.Do(req)
	if err != nil {
		var unknownAuthority x509.UnknownAuthorityError
		var hostname x509.HostnameError
		var verification *tls.CertificateVerificationError
		if errors.As(err, &unknownAuthority) || errors.As(err, &hostname) || errors.As(err, &verification) {
			return nil, fmt.Errorf("%w: node %s: %w", errKubeletRefused, node.Name, err)
		}
		return nil, err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("%w: node %s answered %s, grant get on nodes/stats", errKubeletRefused, node.Name, resp.Status)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("node %s answered %s", node.Name, resp.Status)
	}
	var summary kubeletSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, fmt.Errorf("node %s: %w", node.Name, err)
	}
	return summary.Pods, nil
}

// kubeletStats reads every node's summary at once. A node that fails leaves its pods
// out and complete false; err is set only when no node answered.
func (k *Client) kubeletStats(ctx context.Context) (pods []kubeletPodStats, complete bool, err error) {
	if k.kubelet == nil {
		return nil, false, errKubeletRefused
	}
	nodes, err := k.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{ResourceVersion: "0"})
	if err != nil {
		return nil, false, err
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	var errs []error
	for i := range nodes.Items {
		wg.Go(func() {
			summary, err := k.nodeSummary(ctx, &nodes.Items[i])
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			pods = append(pods, summary...)
		})
	}
	wg.Wait()
	if len(errs) == len(nodes.Items) && len(errs) > 0 {
		return nil, false, errors.Join(errs...)
	}
	k.rootfs.Store(&rootfsCache{at: time.Now(), sizes: rootfsSizes(pods)})
	return pods, len(errs) == 0, nil
}
