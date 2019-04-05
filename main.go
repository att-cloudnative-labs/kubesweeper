package main

import (
	"fmt"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type bin int

func (b bin) String() string {
	return fmt.Sprintf("%b", b)
}

type DeleteFunc func(deployment *v1.Deployment, restarts int) (bool, error)

func DeleteCrash(deployment *v1.Deployment, restarts int) (bool, error) {
	// do your crash loop logic in here

	return true, nil
}

func DeleteGeneric(deployment *v1.Deployment, restarts int) (bool, error) {
	// do your generic logic in here

	return true, nil
}

func main() {
	// initialize the config, from yaml or environment variables
	var kleanerConfig = ConfigObj
	// create the map that will hold the reasons and the config object
	var waitingReasons = make(map[string]CrawlerConfigDetails)

	// fill the map
	for _, conf := range kleanerConfig.Reasons {
		waitingReasons[conf.Reason] = conf
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

		for _, item := range pods.Items {
		StatusLoop:
			for _, status := range item.Status.ContainerStatuses {
				waiting := status.State.Waiting
				if waiting != nil {
					reason := waiting.Reason

					crawlerConfigDetails, ok := waitingReasons[reason]
					if ok {

						rs, err := clientset.AppsV1().ReplicaSets(item.Namespace).Get(item.OwnerReferences[0].Name, metav1.GetOptions{})
						if err != nil {
							fmt.Printf("Error retrieving ReplicaSets. Error: %s\n", err.Error())
							continue StatusLoop
						}

						if rs.OwnerReferences != nil {
							deploy, err := clientset.AppsV1().Deployments(item.Namespace).Get(rs.OwnerReferences[0].Name, metav1.GetOptions{})

							if err != nil {
								// do some more error handling
							}

							// this is where the new interface method(s) come in.
							// The restart threshold will be 0 if not specified in the config, so have to handle that case.
							_, err = crawlerConfigDetails.DeleteFunction(deploy, crawlerConfigDetails.RestartThreshold)

							if err != nil {
								// do even more error handling if you don't change the return signature
							}
						}
					}

					// Tremaine - you still have to untangle all of this and put the right logic into the right Delete function
					//_, reasonInWaitingReasons := waitingReasons[reason]
					//if reasonInWaitingReasons || (reason == "CrashLoopBackOff" && status.RestartCount > restartThreshold) {
					//	if reason == "CrashLoopBackOff" {
					//		fmt.Printf("%s / %s has %s restarts, which is over the %s restart limit.", item.Namespace,
					//			item.Name, strconv.Itoa(int(status.RestartCount)), strconv.Itoa(int(restartThreshold)))
					//	} else if reasonInWaitingReasons {
					//		fmt.Printf("%s / %s has a status of %s, which is configured to be deleted.", item.Namespace,
					//			item.Name, reason)
					//	}
					//
					//	rs, err := clientset.AppsV1().ReplicaSets(item.Namespace).Get(item.OwnerReferences[0].Name, metav1.GetOptions{})
					//	if err != nil {
					//		fmt.Printf("Error retrieving ReplicaSets. Error: %s\n", err.Error())
					//		continue StatusLoop
					//	}
					//
					//	if rs.OwnerReferences != nil {
					//		deploy, err := clientset.AppsV1().Deployments(item.Namespace).Get(rs.OwnerReferences[0].Name, metav1.GetOptions{})
					//		if err != nil {
					//			fmt.Printf("Error retrieving Deployments. Error: %s\n", err.Error())
					//			continue StatusLoop
					//		}
					//		if deploy != nil {
					//			if deploy.Name != "" {
					//				policy := metav1.DeletePropagationForeground
					//				gracePeriodSeconds := int64(0)
					//				fmt.Printf("About to delete %s/%s and its associated resources.\n", item.Namespace, deploy.Name)
					//				err := clientset.AppsV1().Deployments(item.Namespace).Delete(rs.OwnerReferences[0].Name, &metav1.DeleteOptions{PropagationPolicy: &policy, GracePeriodSeconds: &gracePeriodSeconds})
					//				if err != nil {
					//					fmt.Printf("%s/%s, Error: %s \n", item.Namespace, deploy.Name, err.Error())
					//					continue StatusLoop
					//				}
					//			} else {
					//				fmt.Println("No deployment name.")
					//			}
					//		}
					//	} else {
					//		fmt.Println("No replica set owner reference.")
					//	}
					//}
				}
			}
		}
	}
}
