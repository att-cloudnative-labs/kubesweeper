package main

import (
	"fmt"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/rest"
)

type bin int

func (b bin) String() string {
	return fmt.Sprintf("%b", b)
}

type DeleteFunc func(deploymentInterface typedappsv1.DeploymentInterface, deployment *v1.Deployment, restarts int, restartThreshold int) (bool, error)

func DeleteCrash(deploymentInterface typedappsv1.DeploymentInterface, deployment *v1.Deployment, restarts int, restartThreshold int) (bool, error) {
	// do your crash loop logic in here
	if restartThreshold > 0 && restarts >= restartThreshold {
		return DeleteGeneric(deploymentInterface, deployment, restarts, restartThreshold)
	} else {
		fmt.Printf("%s/%s is in CrashLoop state, but not deleting. Restarts/Threshold = %s/%s", deployment.Namespace, deployment.Name, restarts, restartThreshold)
		return false, nil
	}
}

func DeleteGeneric(deploymentInterface typedappsv1.DeploymentInterface, deployment *v1.Deployment, restarts int, restartThreshold int) (bool, error) {
	// do your generic logic in here
	policy := metav1.DeletePropagationForeground
	gracePeriodSeconds := int64(0)
	fmt.Printf("About to delete %s/%s and its associated resources.\n", deployment.Namespace, deployment.Name)

	err := deploymentInterface.Delete(deployment.Name, &metav1.DeleteOptions{PropagationPolicy: &policy, GracePeriodSeconds: &gracePeriodSeconds})
	if err != nil {
		fmt.Printf("%s/%s, Error: %s \n", deployment.Namespace, deployment.Name, err.Error())
		return false, err
	}

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

	// fail fast if not in the cluster
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

		for _, pod := range pods.Items {
		StatusLoop:
			for _, status := range pod.Status.ContainerStatuses {
				waiting := status.State.Waiting
				if waiting != nil {
					reason := waiting.Reason

					crawlerConfigDetails, ok := waitingReasons[reason]
					if ok {

						rs, err := clientset.AppsV1().ReplicaSets(pod.Namespace).Get(pod.OwnerReferences[0].Name, metav1.GetOptions{})
						if err != nil {
							fmt.Printf("Error retrieving ReplicaSets. Error: %s\n", err.Error())
							continue StatusLoop
						}

						if rs.OwnerReferences != nil {
							deploy, err := clientset.AppsV1().Deployments(pod.Namespace).Get(rs.OwnerReferences[0].Name, metav1.GetOptions{})
							if err != nil {
								fmt.Printf("Error retrieving Deployments. Error: %s\n", err.Error())
								continue StatusLoop
							}
							if deploy != nil && deploy.Name != "" { // indicates something to be deleted
								// this is where the new interface method(s) come in.
								// The restart threshold will be 0 if not specified in the config, so have to handle that case.
								_, err = crawlerConfigDetails.DeleteFunction(clientset.AppsV1().Deployments(pod.Namespace), deploy, int(status.RestartCount), crawlerConfigDetails.RestartThreshold)

								if err != nil {
									// do even more error handling if you don't change the return signature
								}

							} else {
								fmt.Println("No deployment found.")
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
