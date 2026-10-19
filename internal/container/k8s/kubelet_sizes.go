package k8s

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/amir20/dozzle/internal/container"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The Client is a container.SizeReader, so the store's size monitor fills the disk
// column for k8s the same way it does for Docker.

const sizeCacheTTL = time.Minute

// rootfsCache is the writable-layer sizes from the last summary read.
type rootfsCache struct {
	at    time.Time
	sizes map[string]int64
}

// ContainerSizes reads every container's writable layer (rootfs) from the kubelets.
// One that refuses turns sizes off for the host rather than being retried.
func (k *Client) ContainerSizes(ctx context.Context) (map[string]int64, error) {
	pods, _, err := k.kubeletStats(ctx)
	if err != nil {
		if errors.Is(err, errKubeletRefused) {
			return nil, fmt.Errorf("%w: %w", container.ErrLayerSizeUnsupported, err)
		}
		return nil, err
	}
	return rootfsSizes(pods), nil
}

func rootfsSizes(pods []kubeletPodStats) map[string]int64 {
	sizes := make(map[string]int64)
	for _, pod := range pods {
		for _, c := range pod.Containers {
			if c.Rootfs != nil && c.Rootfs.UsedBytes != nil {
				sizes[pod.PodRef.Namespace+":"+pod.PodRef.Name+":"+c.Name] = int64(*c.Rootfs.UsedBytes)
			}
		}
	}
	return sizes
}

// ContainerSize answers from the last summary read when it is recent: the stats
// collector reads every node's every networkRefresh while stats flow, which is also
// the only time the size monitor asks. Otherwise it asks the kubelet on the
// container's node.
func (k *Client) ContainerSize(ctx context.Context, id string) (int64, error) {
	if k.kubelet == nil {
		return 0, errKubeletRefused
	}
	if cached := k.rootfs.Load(); cached != nil && time.Since(cached.at) < sizeCacheTTL {
		if size, ok := cached.sizes[id]; ok {
			return size, nil
		}
	}
	namespace, podName, containerName := parsePodContainerID(id)
	pod, err := k.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{ResourceVersion: "0"})
	if err != nil {
		return 0, err
	}
	node, err := k.Clientset.CoreV1().Nodes().Get(ctx, pod.Spec.NodeName, metav1.GetOptions{ResourceVersion: "0"})
	if err != nil {
		return 0, err
	}
	stats, err := k.nodeSummary(ctx, node)
	if err != nil {
		return 0, err
	}
	for _, p := range stats {
		if p.PodRef.Namespace != namespace || p.PodRef.Name != podName {
			continue
		}
		for _, c := range p.Containers {
			if c.Name == containerName && c.Rootfs != nil && c.Rootfs.UsedBytes != nil {
				return int64(*c.Rootfs.UsedBytes), nil
			}
		}
	}
	return 0, fmt.Errorf("kubelet has no size for %s", id)
}

// MeasuresPeriodically marks the Client as a container.PeriodicSizeReader: k8s stats
// carry no disk write counter, and a kubelet answers from its cache.
func (k *Client) MeasuresPeriodically() {}

// DiskUsage reads the size of every PVC and emptyDir a container mounts. Secrets,
// config maps and projected tokens are left out, as Docker never counts bind mounts.
// Nothing on a node is reclaimable the way docker prune frees it, so Reclaimable
// stays zero.
func (k *Client) DiskUsage(ctx context.Context) (container.DiskUsage, error) {
	stats, _, err := k.kubeletStats(ctx)
	if err != nil {
		return container.DiskUsage{}, err
	}
	var pods []corev1.Pod
	for _, namespace := range k.namespace {
		list, err := k.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector:   k.labelSelector,
			ResourceVersion: "0",
		})
		if err != nil {
			return container.DiskUsage{}, err
		}
		pods = append(pods, list.Items...)
	}
	return container.DiskUsage{Volumes: volumeUsage(pods, stats)}, nil
}

func volumeUsage(pods []corev1.Pod, stats []kubeletPodStats) map[string][]container.VolumeUsage {
	used := make(map[string]int64) // namespace/pod/volume -> bytes
	for _, pod := range stats {
		for _, v := range pod.Volume {
			if v.UsedBytes != nil {
				used[pod.PodRef.Namespace+"/"+pod.PodRef.Name+"/"+v.Name] = int64(*v.UsedBytes)
			}
		}
	}

	type mount struct {
		id    string
		key   string // what links count: a claim is shared across pods, an emptyDir is not
		usage container.VolumeUsage
	}
	var mounts []mount
	links := make(map[string]int64)
	for _, pod := range pods {
		sources := make(map[string]corev1.Volume, len(pod.Spec.Volumes))
		for _, v := range pod.Spec.Volumes {
			sources[v.Name] = v
		}
		for _, c := range slices.Concat(pod.Spec.InitContainers, pod.Spec.Containers) {
			for _, vm := range c.VolumeMounts {
				size, measured := used[pod.Namespace+"/"+pod.Name+"/"+vm.Name]
				if !measured {
					continue
				}
				m := mount{
					id:    pod.Namespace + ":" + pod.Name + ":" + c.Name,
					usage: container.VolumeUsage{Name: vm.Name, Destination: vm.MountPath, Size: size},
				}
				source := sources[vm.Name]
				switch {
				case source.PersistentVolumeClaim != nil:
					m.usage.Name = source.PersistentVolumeClaim.ClaimName
					m.key = pod.Namespace + "/pvc/" + m.usage.Name
				case source.EmptyDir != nil, source.Ephemeral != nil:
					m.key = pod.Namespace + "/" + pod.Name + "/" + vm.Name
				default:
					continue
				}
				mounts = append(mounts, m)
				links[m.key]++
			}
		}
	}

	volumes := make(map[string][]container.VolumeUsage)
	for _, m := range mounts {
		m.usage.Links = links[m.key]
		volumes[m.id] = append(volumes[m.id], m.usage)
	}
	return volumes
}
