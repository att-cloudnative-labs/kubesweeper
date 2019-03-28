package main

import (
	"fmt"
	"strconv"
	"log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

		for i := 0; i < len(pods.Items); i++ {
			for j := 0; j < len(pods.Items[i].Status.ContainerStatuses); j++ {
				if pods.Items[i].Status.ContainerStatuses[j].State.Waiting != nil {
					if pods.Items[i].Status.ContainerStatuses[j].State.Waiting.Reason != "" {
						if pods.Items[i].Status.ContainerStatuses[j].State.Waiting.Reason == "CrashLoopBackOff" {
							if pods.Items[i].Status.ContainerStatuses[j].RestartCount > restartThreshold {
								fmt.Println(pods.Items[i].Namespace + "/" + pods.Items[i].Name + " has " +
									strconv.Itoa(int(pods.Items[i].Status.ContainerStatuses[j].RestartCount)) + " restarts, " +
									"which is over the " + strconv.Itoa(int(restartThreshold)) + " restart limit.\n")
								rs, _ := clientset.AppsV1().ReplicaSets(pods.Items[i].Namespace).Get(pods.Items[i].OwnerReferences[0].Name, metav1.GetOptions{})
								if rs != nil {
									if rs.OwnerReferences != nil {
										deploy, _ := clientset.AppsV1().Deployments(pods.Items[i].Namespace).Get(rs.OwnerReferences[0].Name, metav1.GetOptions{})
										if deploy != nil {
											if deploy.Name != "" {
												policy := metav1.DeletePropagationForeground
												gracePeriodSeconds := int64(0)
												fmt.Println("About to delete " + pods.Items[i].Namespace + "/" +
													deploy.Name + " and its associated resources.\n")
												err := clientset.AppsV1().Deployments(pods.Items[i].Namespace).Delete(rs.OwnerReferences[0].Name, &metav1.DeleteOptions{PropagationPolicy: &policy, GracePeriodSeconds: &gracePeriodSeconds})
												if err != nil {
													fmt.Println(pods.Items[i].Namespace + "/" + deploy.Name + "error: \n")
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
		}
	}
}