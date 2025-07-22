package main

import (
	"fmt"
	"context"
	"log"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	// appsv1 "k8s.io/api/apps/v1"
	// corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
	// metrics "k8s.io/metrics/pkg/apis/metrics/v1beta1"

)

func isContainerTerminated(clientset *kubernetes.Clientset, namespace, podName, containerName string) (bool, error) {
	pod, err := clientset.CoreV1().Pods("default").Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == containerName {
			if status.State.Terminated != nil {
				return true, nil
			}
		}
	}
	return false, nil
}

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// list ra các pod
	pods, err := clientset.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error getting pods: %v", err)
	}

	// Lấy container đầu tiên trong pod[0]
	// Lấy duration
	pod := pods.Items[0]
	container := pod.Spec.Containers[0]

	
	if len(pod.Status.ContainerStatuses) > 0  {			// kiểm tra xem pod có bất kỳ container nào được ghi nhận trạng thái hay chưa
		status := pod.Status.ContainerStatuses[0]
		if status.State.Terminated != nil {				// lấy status của container đã terminate
			startTime := status.State.Terminated.StartedAt.Time
			endTime := status.State.Terminated.FinishedAt.Time
			duration := endTime.Sub(startTime)
			fmt.Println("Duration: %d ms\n", duration.Milliseconds())
		}
	}

	// Lấy memory
	// memoryQuantity := container.Usage.Memory().Value()
	// fmt.Printf("memory: %dB\n", memoryQuantity)

	// Lấy CPU
	// cpuQuantity := container.Usage.Cpu().MilliValue()
	// fmt.Printf("CPU: %dmCPU\n", cpuQuantity)
	
	// Lấy name của container
	name := container.Name
	fmt.Printf("Name: %s\n", name)

	// Lấy peak memory
	metricsClient, err := metricsv.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	var peakMemory int64 = 0
	containerName := container.Name
	podName := pod.Name

	for {
		terminated, err := isContainerTerminated(clientset, "default", podName, containerName)
		if err != nil {
			log.Printf("Error checking container status: %v", err)
			break
		}
		if terminated {
			fmt.Println("Container stopped")
			break
		}

		podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses("default").Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			log.Printf("Error getting pod metrics: %v\n", err)
			time.Sleep(10 * time.Second)
			continue 
		}
		for _, c := range podMetrics.Containers {
			if c.Name == containerName {
				mem := c.Usage.Memory().Value()
				if mem > peakMemory {
					peakMemory = mem
				}
				break // chỉ lấy container đã chọn
			}
		}
		time.Sleep(10 * time.Second)
	}
	fmt.Printf("Peak Memory: %dB\n", peakMemory)
}