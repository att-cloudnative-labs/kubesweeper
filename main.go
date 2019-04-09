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
	// pod is in a CrashLoopBackOff state
	// check if restart number meets or exceeds the restart threshold
	// the restart threshold will be 0 if not specified in the config, so have to handle that case
	if restartThreshold > 0 && restarts >= restartThreshold {
		return DeleteGeneric(deploymentInterface, deployment, restarts, restartThreshold)
	} else {
		fmt.Printf("%s/%s is in a CrashLoopBackOff state, but doesn't meet the restart threshold. " +
			"Restarts/Threshold = %v/%v", deployment.Namespace, deployment.Name, restarts, restartThreshold)
		return false, nil
	}
}

func DeleteGeneric(deploymentInterface typedappsv1.DeploymentInterface, deployment *v1.Deployment, restarts int, restartThreshold int) (bool, error) {
	// pod is in a state defined in config.yaml
	policy := metav1.DeletePropagationForeground
	gracePeriodSeconds := int64(0)
	fmt.Printf("About to delete %s/%s and its associated resources.\n", deployment.Namespace, deployment.Name)

	err := deploymentInterface.Delete(deployment.Name, &metav1.DeleteOptions{PropagationPolicy: &policy, GracePeriodSeconds: &gracePeriodSeconds})
	if err != nil {
		fmt.Printf("%s/%s, Error: %s \n", deployment.Namespace, deployment.Name, err.Error())
		return false, err
	}
	fmt.Printf("Deleted %s/%s and its associated resources.\n", deployment.Namespace, deployment.Name)

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
		// get a list of pods for all namespaces
		pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Println("Got list of pods.")

		for _, pod := range pods.Items {
		StatusLoop:
			for _, status := range pod.Status.ContainerStatuses {
				waiting := status.State.Waiting
				if waiting != nil {
					reason := waiting.Reason
					if crawlerConfigDetails, ok := waitingReasons[reason]; ok {
						fmt.Printf("Waiting reason match. %s/%s has a waiting reason of: %s", pod.Namespace,
							pod.OwnerReferences[0].Name, reason)
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
								_, err = crawlerConfigDetails.DeleteFunction(clientset.AppsV1().Deployments(pod.Namespace), deploy, int(status.RestartCount), crawlerConfigDetails.RestartThreshold)
								if err != nil {
									fmt.Printf("Error deleting Deployment. Error: %s\n", err.Error())
									continue StatusLoop
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
