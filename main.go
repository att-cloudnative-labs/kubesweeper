package main

import (
	"fmt"
	"strconv"
	"time"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type bin int

func (b bin) String() string {
	return fmt.Sprintf("%b", b)
}

type DeleteFunc func(clientset *kubernetes.Clientset, deployment *v1.Deployment, restarts int, restartThreshold int) (bool, error)

func DeleteCrash(clientset *kubernetes.Clientset, deployment *v1.Deployment, restarts int, restartThreshold int) (bool, error) {
	// pod is in a CrashLoopBackOff state
	// check if restart number meets or exceeds the restart threshold
	// the restart threshold will be 0 if not specified in the config, so have to handle that case
	if restartThreshold > 0 && restarts >= restartThreshold {
		return DeleteGeneric(clientset, deployment, restarts, restartThreshold)
	} else {
		fmt.Printf("%s/%s is in a CrashLoopBackOff state, but doesn't meet the restart threshold. "+
			"Restarts/Threshold = %v/%v\n", deployment.Namespace, deployment.Name, restarts, restartThreshold)
		return false, nil
	}
}

func DeleteGeneric(clientset *kubernetes.Clientset, deployment *v1.Deployment, restarts int, restartThreshold int) (bool, error) {
	// pod is in a state defined in config.yaml
	policy := metav1.DeletePropagationForeground
	gracePeriodSeconds := int64(0)
	fmt.Printf("About to delete %s/%s and its associated resources.\n", deployment.Namespace, deployment.Name)
	err := clientset.AppsV1().Deployments(deployment.Namespace).Delete(deployment.Name, &metav1.DeleteOptions{PropagationPolicy: &policy, GracePeriodSeconds: &gracePeriodSeconds})
	if err != nil {
		fmt.Printf("%s/%s, Error: %s \n", deployment.Namespace, deployment.Name, err.Error())
		return false, err
	}
	_, err = DeleteLeftoverResources(clientset, deployment)
	if err != nil {
		fmt.Printf("%s/%s, Error: %s \n", deployment.Namespace, deployment.Name, err.Error())
		return false, err
	}

	fmt.Printf("Deleted %s/%s and its associated resources.\n", deployment.Namespace, deployment.Name)

	return true, nil
}

func DeleteLeftoverResources(clientset *kubernetes.Clientset, deployment *v1.Deployment) (bool, error) {
	// TODO: make each one an if statement based on configuration
	_, err := DeleteIngress(clientset, deployment)
	if err != nil {
		fmt.Printf("%s/%s, Error calling DeleteIngress: %s \n", deployment.Namespace, deployment.Name, err.Error())
		return false, err
	}

	_, err = DeleteService(clientset, deployment)
	if err != nil {
		fmt.Printf("%s/%s, Error calling DeleteService: %s \n", deployment.Namespace, deployment.Name, err.Error())
		return false, err
	}

	_, err = DeleteHpa(clientset, deployment)
	if err != nil {
		fmt.Printf("%s/%s, Error calling DeleteHpa: %s \n", deployment.Namespace, deployment.Name, err.Error())
		return false, err
	}

	return true, err
}

func DeleteIngress(clientset *kubernetes.Clientset, deployment *v1.Deployment) (bool, error) {
	policy := metav1.DeletePropagationForeground
	gracePeriodSeconds := int64(0)
	fmt.Printf("About to delete the ingress of %s/%s.\n", deployment.Namespace, deployment.Name)

	err := clientset.ExtensionsV1beta1().Ingresses(deployment.Namespace).Delete(deployment.Name, &metav1.DeleteOptions{PropagationPolicy: &policy, GracePeriodSeconds: &gracePeriodSeconds})
	if err != nil {
		fmt.Printf("%s/%s, Error deleting ingress: %s \n", deployment.Namespace, deployment.Name, err.Error())
		return false, err
	}
	fmt.Printf("Deleted the ingress of %s/%s.\n", deployment.Namespace, deployment.Name)

	return true, nil
}

func DeleteService(clientset *kubernetes.Clientset, deployment *v1.Deployment) (bool, error) {
	policy := metav1.DeletePropagationForeground
	gracePeriodSeconds := int64(0)
	fmt.Printf("About to delete the service of %s/%s.\n", deployment.Namespace, deployment.Name)

	err := clientset.CoreV1().Services(deployment.Namespace).Delete(deployment.Name, &metav1.DeleteOptions{PropagationPolicy: &policy, GracePeriodSeconds: &gracePeriodSeconds})
	if err != nil {
		fmt.Printf("%s/%s, Error deleting service: %s \n", deployment.Namespace, deployment.Name, err.Error())
		return false, err
	}
	fmt.Printf("Deleted the service of %s/%s.\n", deployment.Namespace, deployment.Name)

	return true, nil
}

func DeleteHpa(clientset *kubernetes.Clientset, deployment *v1.Deployment) (bool, error) {
	policy := metav1.DeletePropagationForeground
	gracePeriodSeconds := int64(0)
	fmt.Printf("About to delete the hpa of %s/%s.\n", deployment.Namespace, deployment.Name)

	err := clientset.AutoscalingV1().HorizontalPodAutoscalers(deployment.Namespace).Delete(deployment.Name, &metav1.DeleteOptions{PropagationPolicy: &policy, GracePeriodSeconds: &gracePeriodSeconds})
	if err != nil {
		fmt.Printf("%s/%s, Error deleting hpa: %s \n", deployment.Namespace, deployment.Name, err.Error())
		return false, err
	}
	fmt.Printf("Deleted the hpa of %s/%s.\n", deployment.Namespace, deployment.Name)

	return true, nil
}

func main() {
	// initialize the config, from yaml or environment variables
	var kleanerConfig = ConfigObj
	// create the map that will hold the reasons and the config object
	var waitingReasons = make(map[string]SweeperConfigDetails)

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

	fmt.Println("Beginning the sweep.")

	// get a list of pods for all namespaces
	pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Println("Got list of pods.")

	for _, pod := range pods.Items {
	StatusLoop:
		for _, status := range pod.Status.ContainerStatuses {
			// CHECK #1: Deployment age
			if pod.Namespace != "kube-system" && pod.Namespace != "default" && pod.Namespace != "nats-cluster" {
				rs, err := clientset.AppsV1().ReplicaSets(pod.Namespace).Get(pod.OwnerReferences[0].Name, metav1.GetOptions{})
				if err != nil {
					fmt.Printf("Error retrieving ReplicaSets. Error: %s\n", err.Error())
					continue StatusLoop
				}
				deploy, err := clientset.AppsV1().Deployments(pod.Namespace).Get(rs.OwnerReferences[0].Name, metav1.GetOptions{})
				if err != nil {
					fmt.Printf("Error retrieving Deployments. Error: %s\n", err.Error())
					continue StatusLoop
				}
				if deploy != nil && deploy.Name != "" {
					fmt.Println(deploy.GetNamespace() + "/" + deploy.GetName())
					fmt.Println(deploy.GetCreationTimestamp())
					if deploy.GetCreationTimestamp().AddDate(0, 0, kleanerConfig.DayLimit).Before(time.Now()) {
						fmt.Println("I found an old deployment past " + strconv.Itoa(kleanerConfig.DayLimit) + " days!")
						//if deploy.GetName() == "policy-2-1-0" {
							_, err = DeleteGeneric(clientset, deploy, int(status.RestartCount), 0)
						//}
					}
				}
			}

			// CHECK #2: Pod "Waiting" state
			waiting := status.State.Waiting
			if waiting != nil { // There is a bad pod state
				reason := waiting.Reason
				if SweeperConfigDetails, ok := waitingReasons[reason]; ok {
					fmt.Printf("Waiting reason match. %s/%s has a waiting reason of: %s\n", pod.Namespace,
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
						if deploy != nil && deploy.Name != "" && pod.Namespace != "kube-system" && pod.Namespace != "default" { // indicates something to be deleted
							_, err = SweeperConfigDetails.DeleteFunction(clientset, deploy, int(status.RestartCount), SweeperConfigDetails.RestartThreshold)
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
