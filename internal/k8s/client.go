package k8s

import (
	"context"
	"io"
	"strings"
	"time"

	"os"

	"github.com/amir20/dozzle/internal/container"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rs/zerolog/log"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type k8sClient struct {
	client    *kubernetes.Clientset
	namespace string
}

func NewK8sClient(namespace string) (*k8sClient, error) {
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

	return &k8sClient{
		client:    clientset,
		namespace: namespace,
	}, nil
}
func (k *k8sClient) ListContainers(ctx context.Context, filter container.ContainerFilter) ([]container.Container, error) {
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
				State:   string(pod.Status.Phase),
				Tty:     c.TTY,
				Group:   pod.Name,
			})
		}
	}
	return containers, nil
}

func (k *k8sClient) FindContainer(ctx context.Context, id string) (container.Container, error) {
	// Implementation to find a specific container by ID
	//
	k.client.CoreV1()
	return container.Container{}, nil
}

func (k *k8sClient) ContainerLogs(ctx context.Context, id string, since time.Time, stdType container.StdType) (io.ReadCloser, error) {
	podName, containerName := parsePodContainerID(id)
	opts := &corev1.PodLogOptions{
		Container: containerName,
		Follow:    true,
		Previous:  false,
		SinceTime: &metav1.Time{Time: since},
	}

	return k.client.CoreV1().Pods(k.namespace).GetLogs(podName, opts).Stream(ctx)
}

func (k *k8sClient) ContainerEvents(ctx context.Context, ch chan<- container.ContainerEvent) error {
	watch, err := k.client.CoreV1().Pods(k.namespace).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	go func() {
		for event := range watch.ResultChan() {
			// Convert and send kubernetes events to container events
			// Implementation details depend on your ContainerEvent struct
		}
	}()

	return nil
}

func (k *k8sClient) ContainerLogsBetweenDates(ctx context.Context, id string, start time.Time, end time.Time, stdType container.StdType) (io.ReadCloser, error) {
	podName, containerName := parsePodContainerID(id)
	opts := &corev1.PodLogOptions{
		Container: containerName,
		SinceTime: &metav1.Time{Time: start},
	}

	return k.client.CoreV1().Pods(k.namespace).GetLogs(podName, opts).Stream(ctx)
}

func (k *k8sClient) ContainerStats(ctx context.Context, id string, ch chan<- container.ContainerStat) error {
	// Implementation to stream container stats
	return nil
}

func (k *k8sClient) Ping(ctx context.Context) error {
	_, err := k.client.CoreV1().Pods(k.namespace).List(ctx, metav1.ListOptions{Limit: 1})
	return err
}

func (k *k8sClient) Host() container.Host {
	// Return host information
	return container.Host{}
}

func (k *k8sClient) ContainerActions(ctx context.Context, action container.ContainerAction, containerID string) error {
	// Implementation for container actions (start, stop, restart, etc.)
	return nil
}

// Helper function to parse pod and container names from container ID
func parsePodContainerID(id string) (string, string) {
	parts := strings.Split(id, ":")
	return parts[0], parts[1]
}
