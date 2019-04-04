package main

import (
	"fmt"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type bin int

func (b bin) String() string {
	return fmt.Sprintf("%b", b)
}

func main() {
	NUMBER_RESTARTS := 100

	waitingReasons := map[string]struct{}{
		"ErrImagePull": 	{},
		"Completed": 		{},
		"Failed": 			{},
		"ImagePullBackOff": {},
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("Beginning the crawl.")

	for {
		pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}

		restartThreshold := int32(NUMBER_RESTARTS)

		for _, item := range pods.Items {
			StatusLoop:
			for _, status := range item.Status.ContainerStatuses {
				waiting := status.State.Waiting
				if waiting != nil {
					reason := waiting.Reason
					_, reasonInWaitingReasons := waitingReasons[reason]
					if reasonInWaitingReasons || (reason == "CrashLoopBackOff" && status.RestartCount > restartThreshold) {
						if reason == "CrashLoopBackOff" {
							fmt.Printf("%s / %s has %s restarts, which is over the %s restart limit.", item.Namespace,
								item.Name, strconv.Itoa(int(status.RestartCount)), strconv.Itoa(int(restartThreshold)))
						} else if reasonInWaitingReasons {
							fmt.Printf("%s / %s has a status of %s, which is configured to be deleted.", item.Namespace,
								item.Name, reason)
						}

						rs, err := clientset.AppsV1().ReplicaSets(item.Namespace).Get(item.OwnerReferences[0].Name, metav1.GetOptions{})
						if err != nil {
							fmt.Printf("Error retrieving ReplicaSets. Error: %s\n", err.Error())
							continue StatusLoop
						}

						if rs.OwnerReferences != nil {
							deploy, err := clientset.AppsV1().Deployments(item.Namespace).Get(rs.OwnerReferences[0].Name, metav1.GetOptions{})
							if err != nil {
								fmt.Printf("Error retrieving Deployments. Error: %s\n", err.Error())
								continue StatusLoop
							}
							if deploy != nil {
								if deploy.Name != "" {
									policy := metav1.DeletePropagationForeground
									gracePeriodSeconds := int64(0)
									fmt.Printf("About to delete %s/%s and its associated resources.\n", item.Namespace, deploy.Name)
									err := clientset.AppsV1().Deployments(item.Namespace).Delete(rs.OwnerReferences[0].Name, &metav1.DeleteOptions{PropagationPolicy: &policy, GracePeriodSeconds: &gracePeriodSeconds})
									if err != nil {
										fmt.Printf("%s/%s, Error: %s \n", item.Namespace, deploy.Name, err.Error())
										continue StatusLoop
									}
								} else {
									fmt.Println("No deployment name.")
								}
							}
						} else {
							fmt.Println("No replica set owner reference.")
						}
					}
				}
			}
		}
	}
}
