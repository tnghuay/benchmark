package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

func isContainerTerminated(clientset *kubernetes.Clientset, namespace, podName, containerName string) (bool, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == containerName && status.State.Terminated != nil {
			return true, nil
		}
	}
	return false, nil
}

func main() {
	// Get config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}


	metricsClient, err := metricsv.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	namespace := "default"

	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil || len(pods.Items) == 0 {
		log.Fatalf("Error getting pods or no pods found: %v", err)
	}

	pod := pods.Items[0]
	container := pod.Spec.Containers[0]
	containerName := container.Name
	podName := pod.Name

	fmt.Printf("Container Name: %s\n", containerName)

	var peakMemory int64 = 0             // bytes
	var totalCPU int64 = 0               // millicores
	var cpuSamples int64 = 0
	var startTime, endTime time.Time     // for duration

	for {
		terminated, err := isContainerTerminated(clientset, namespace, podName, containerName)
		if err != nil {
			log.Printf("Error checking container status: %v", err)
			break
		}
		if terminated {
			fmt.Println("Container stopped.")

			// Get start & end time after termination
			pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
			if err == nil {
				for _, status := range pod.Status.ContainerStatuses {
					if status.Name == containerName && status.State.Terminated != nil {
						startTime = status.State.Terminated.StartedAt.Time
						endTime = status.State.Terminated.FinishedAt.Time
					}
				}
			}
			break
		}

		// Get metrics
		podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			log.Printf("Error getting pod metrics: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}

		for _, c := range podMetrics.Containers {
			if c.Name == containerName {
				mem := c.Usage.Memory().Value()        // bytes
				cpu := c.Usage.Cpu().MilliValue()      // millicores

				if mem > peakMemory {
					peakMemory = mem
				}

				totalCPU += cpu
				cpuSamples++
				break
			}
		}

		time.Sleep(10 * time.Second)
	}

	// Output 
	fmt.Printf("Peak Memory: %.2f MiB\n", float64(peakMemory)/(1024*1024))

	if cpuSamples > 0 {
		avgCPU := float64(totalCPU) / float64(cpuSamples)
		fmt.Printf("Average CPU: %.2f millicores\n", avgCPU)
	} else {
		fmt.Println("No CPU samples collected.")
	}

	if !startTime.IsZero() && !endTime.IsZero() {
		durationMs := endTime.Sub(startTime).Milliseconds()
		fmt.Printf("Duration: %d ms\n", durationMs)
	} else {
		fmt.Println("Duration not available.")
	}
}