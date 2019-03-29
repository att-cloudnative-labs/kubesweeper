package main

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"strconv"
)

type bin int

func (b bin) String() string {
	return fmt.Sprintf("%b", b)
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

	for {
		pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}

		restartThreshold := int32(100)

		for _, item := range pods.Items {
			for _, status := range item.Status.ContainerStatuses {
				waiting := status.State.Waiting

				if waiting != nil {
					reason := waiting.Reason
					if reason == "CrashLoopBackOff" && status.RestartCount > restartThreshold {
						fmt.Println(item.Namespace + "/" + item.Name + " has " +
							strconv.Itoa(int(status.RestartCount)) + " restarts, " +
							"which is over the " + strconv.Itoa(int(restartThreshold)) + " restart limit.\n")
						rs, err := clientset.AppsV1().ReplicaSets(item.Namespace).Get(item.OwnerReferences[0].Name, metav1.GetOptions{})
						// could handle error here instead, like:
						if err != nil {
							fmt.Printf("Error retrieving ReplicaSets. Error: %s", err.Error())
							continue
						}

						if rs.OwnerReferences != nil {
							deploy, err := clientset.AppsV1().Deployments(item.Namespace).Get(rs.OwnerReferences[0].Name, metav1.GetOptions{})
							if err != nil {
								fmt.Printf("Error retrieving ReplicaSets. Error: %s", err.Error())
								continue
							}
							if deploy != nil {
								if deploy.Name != "" {
									policy := metav1.DeletePropagationForeground
									gracePeriodSeconds := int64(0)
									fmt.Printf("About to delete %s/%s and its associated resources.\n", item.Namespace, deploy.Name)
									err := clientset.AppsV1().Deployments(item.Namespace).Delete(rs.OwnerReferences[0].Name, &metav1.DeleteOptions{PropagationPolicy: &policy, GracePeriodSeconds: &gracePeriodSeconds})
									if err != nil {
										fmt.Printf("%s/%s, Error: %s \n", item.Namespace, deploy.Name, err.Error())
										continue
									}
								} else {
									fmt.Printf("No deployment name!\n")
								}
							}
						} else {
							fmt.Printf("No replica set owner reference!\n")
						}

					}
				}
			}
		}
	}
}
