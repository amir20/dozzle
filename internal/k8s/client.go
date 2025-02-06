package k8s

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"os"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rs/zerolog/log"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sClient struct {
	client    *kubernetes.Clientset
	namespace string
	host      container.Host
}

func NewK8sClient(namespace string) (*K8sClient, error) {
	var config *rest.Config
	var err error

	// Check if we're running in cluster
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		log.Info().Msg("Running in-cluster mode")
	} else {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = os.Getenv("HOME") + "/.kube/config"
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
		log.Info().Msgf("Running in local mode with kubeconfig: %s", kubeconfig)

	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	id, ok := os.LookupEnv("HOSTNAME")
	host := container.Host{}
	var node *corev1.Node
	if ok {
		pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), id, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		log.Debug().Str("node", pod.Spec.NodeName).Msg("Found pod")
		node, err = clientset.CoreV1().Nodes().Get(context.Background(), pod.Spec.NodeName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	} else {
		nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		if len(nodes.Items) == 0 {
			return nil, fmt.Errorf("no nodes found")
		}

		node = &nodes.Items[0]
	}

	host.ID = node.Status.NodeInfo.MachineID
	host.Name = node.Name
	host.MemTotal = node.Status.Capacity.Memory().Value()
	host.NCPU = int(node.Status.Capacity.Cpu().Value())
	host.Swarm = false
	host.DockerVersion = node.Status.NodeInfo.ContainerRuntimeVersion
	host.Type = "k8s"

	return &K8sClient{
		client:    clientset,
		namespace: namespace,
		host:      host,
	}, nil
}
func (k *K8sClient) ListContainers(ctx context.Context, filter container.ContainerFilter) ([]container.Container, error) {
	pods, err := k.client.CoreV1().Pods(k.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var containers []container.Container
	for _, pod := range pods.Items {
		for _, c := range pod.Spec.Containers {
			containers = append(containers, container.Container{
				ID:      pod.Name + ":" + c.Name,
				Name:    c.Name,
				Image:   c.Image,
				Created: pod.CreationTimestamp.Time,
				State:   phaseToState(pod.Status.Phase),
				Tty:     c.TTY,
				Host:    k.host.ID,
			})
		}
	}
	return containers, nil
}

func phaseToState(phase corev1.PodPhase) string {
	switch phase {
	case corev1.PodPending:
		return "created"
	case corev1.PodRunning:
		return "running"
	case corev1.PodSucceeded:
		return "exited"
	case corev1.PodFailed:
		return "exited"
	case corev1.PodUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

func (k *K8sClient) FindContainer(ctx context.Context, id string) (container.Container, error) {
	log.Debug().Str("id", id).Msg("Finding container")
	podName, containerName := parsePodContainerID(id)

	pod, err := k.client.CoreV1().Pods(k.namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return container.Container{}, err
	}

	for _, c := range pod.Spec.Containers {
		if c.Name == containerName {
			return container.Container{
				ID:        pod.Name + ":" + c.Name,
				Name:      c.Name,
				Image:     c.Image,
				Created:   pod.CreationTimestamp.Time,
				State:     phaseToState(pod.Status.Phase),
				StartedAt: pod.Status.StartTime.Time,
				Command:   strings.Join(c.Command, " "),
				Host:      k.host.ID,
				Tty:       c.TTY,
				Stats:     utils.NewRingBuffer[container.ContainerStat](300),
			}, nil
		}
	}

	return container.Container{}, fmt.Errorf("container %s not found in pod %s", containerName, podName)
}

func (k *K8sClient) ContainerLogs(ctx context.Context, id string, since time.Time, stdType container.StdType) (io.ReadCloser, error) {
	podName, containerName := parsePodContainerID(id)
	opts := &corev1.PodLogOptions{
		Container:  containerName,
		Follow:     true,
		Previous:   false,
		Timestamps: true,
		SinceTime:  &metav1.Time{Time: since},
	}

	return k.client.CoreV1().Pods(k.namespace).GetLogs(podName, opts).Stream(ctx)
}

func (k *K8sClient) ContainerLogsBetweenDates(ctx context.Context, id string, start time.Time, end time.Time, stdType container.StdType) (io.ReadCloser, error) {
	podName, containerName := parsePodContainerID(id)
	opts := &corev1.PodLogOptions{
		Container:  containerName,
		Follow:     false,
		Timestamps: true,
		SinceTime:  &metav1.Time{Time: start},
	}

	return k.client.CoreV1().Pods(k.namespace).GetLogs(podName, opts).Stream(ctx)
}

func (k *K8sClient) ContainerEvents(ctx context.Context, ch chan<- container.ContainerEvent) error {
	watch, err := k.client.CoreV1().Pods(k.namespace).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for event := range watch.ResultChan() {
		log.Debug().Interface("event.type", event.Type).Msg("Received kubernetes event")
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			continue
		}

		if pod.Status.StartTime == nil {
			log.Debug().Str("pod", pod.Name).Msg("Pod not started yet")
			continue
		}

		name := ""
		if event.Type == "ADDED" {
			name = "start"
		} else if event.Type == "DELETED" {
			name = "die"
		} else if event.Type == "MODIFIED" {
			if time.Now().Sub(pod.Status.StartTime.Time) < 5*time.Second {
				name = "start"
			} else {
				log.Debug().Str("pod", pod.Name).Msg("No changes to pod to report")
				continue
			}
		}

		log.Debug().Interface("event.Type", event.Type).Str("name", name).Interface("StartTime", pod.Status.StartTime).Msg("Sending container event")

		for _, c := range pod.Spec.Containers {
			ch <- container.ContainerEvent{
				Name:    name,
				ActorID: pod.Name + ":" + c.Name,
				Host:    k.host.ID,
				Time:    time.Now(),
			}
		}
	}

	return nil
}

func (k *K8sClient) ContainerStats(ctx context.Context, id string, ch chan<- container.ContainerStat) error {
	// Implementation to stream container stats
	return nil
}

func (k *K8sClient) Ping(ctx context.Context) error {
	_, err := k.client.CoreV1().Pods(k.namespace).List(ctx, metav1.ListOptions{Limit: 1})
	return err
}

func (k *K8sClient) Host() container.Host {
	return k.host
}

func (k *K8sClient) ContainerActions(ctx context.Context, action container.ContainerAction, containerID string) error {
	// Implementation for container actions (start, stop, restart, etc.)
	return nil
}

// Helper function to parse pod and container names from container ID
func parsePodContainerID(id string) (string, string) {
	parts := strings.Split(id, ":")
	return parts[0], parts[1]
}
