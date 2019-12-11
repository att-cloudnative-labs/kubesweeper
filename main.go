package main

import (
	"fmt"
	"strconv"
	"time"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	typedautoscalingv1 "k8s.io/client-go/kubernetes/typed/autoscaling/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
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
		fmt.Printf("%s/%s is in a CrashLoopBackOff state, but doesn't meet the restart threshold. "+
			"Restarts/Threshold = %v/%v\n", deployment.Namespace, deployment.Name, restarts, restartThreshold)
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

func DeleteIngress(ingressInterface v1beta1.IngressInterface, deployment *v1.Deployment) (bool, error) {
	policy := metav1.DeletePropagationForeground
	gracePeriodSeconds := int64(0)
	fmt.Printf("About to delete the ingress of %s/%s.\n", deployment.Namespace, deployment.Name)

	err := ingressInterface.Delete(deployment.Name, &metav1.DeleteOptions{PropagationPolicy: &policy, GracePeriodSeconds: &gracePeriodSeconds})
	if err != nil {
		fmt.Printf("%s/%s, Error deleting ingress: %s \n", deployment.Namespace, deployment.Name, err.Error())
		return false, err
	}
	fmt.Printf("Deleted the ingress of %s/%s.\n", deployment.Namespace, deployment.Name)

	return true, nil
}

func DeleteService(serviceInterface corev1.ServiceInterface, deployment *v1.Deployment) (bool, error) {
	policy := metav1.DeletePropagationForeground
	gracePeriodSeconds := int64(0)
	fmt.Printf("About to delete the service of %s/%s.\n", deployment.Namespace, deployment.Name)

	err := serviceInterface.Delete(deployment.Name, &metav1.DeleteOptions{PropagationPolicy: &policy, GracePeriodSeconds: &gracePeriodSeconds})
	if err != nil {
		fmt.Printf("%s/%s, Error deleting service: %s \n", deployment.Namespace, deployment.Name, err.Error())
		return false, err
	}
	fmt.Printf("Deleted the service of %s/%s.\n", deployment.Namespace, deployment.Name)

	return true, nil
}

func DeleteHpa(hpaInterface typedautoscalingv1.HorizontalPodAutoscalerInterface, deployment *v1.Deployment) (bool, error) {
	policy := metav1.DeletePropagationForeground
	gracePeriodSeconds := int64(0)
	fmt.Printf("About to delete the hpa of %s/%s.\n", deployment.Namespace, deployment.Name)

	err := hpaInterface.Delete(deployment.Name, &metav1.DeleteOptions{PropagationPolicy: &policy, GracePeriodSeconds: &gracePeriodSeconds})
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
						// if deploy.GetName() == "currentprogram-1-1-1" {
						// fmt.Println("HELLO!")
						_, err = DeleteGeneric(clientset.AppsV1().Deployments(pod.Namespace), deploy, int(status.RestartCount), 0)
						if err != nil {
							fmt.Printf("Error deleting generic. Error: %s\n", err.Error())
							continue StatusLoop
						}
						_, err = DeleteIngress(clientset.ExtensionsV1beta1().Ingresses(deploy.GetNamespace()), deploy)
						if err != nil {
							fmt.Printf("Error deleting ingress. Error: %s\n", err.Error())
							continue StatusLoop
						}
						_, err = DeleteService(clientset.CoreV1().Services(pod.Namespace), deploy)
						if err != nil {
							fmt.Printf("Error deleting service. Error: %s\n", err.Error())
							continue StatusLoop
						}
						_, err = DeleteHpa(clientset.AutoscalingV1().HorizontalPodAutoscalers(pod.Namespace), deploy)
						if err != nil {
							fmt.Printf("Error deleting hpa. Error: %s\n", err.Error())
							continue StatusLoop
						}
						// }
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
							_, err = SweeperConfigDetails.DeleteFunction(clientset.AppsV1().Deployments(pod.Namespace), deploy, int(status.RestartCount), SweeperConfigDetails.RestartThreshold)
							if err != nil {
								fmt.Printf("Error deleting Deployment. Error: %s\n", err.Error())
								continue StatusLoop
							}
							_, err = DeleteIngress(clientset.ExtensionsV1beta1().Ingresses(deploy.GetNamespace()), deploy)
							if err != nil {
								fmt.Printf("Error deleting ingress. Error: %s\n", err.Error())
								continue StatusLoop
							}
							_, err = DeleteService(clientset.CoreV1().Services(pod.Namespace), deploy)
							if err != nil {
								fmt.Printf("Error deleting service. Error: %s\n", err.Error())
								continue StatusLoop
							}
							_, err = DeleteHpa(clientset.AutoscalingV1().HorizontalPodAutoscalers(pod.Namespace), deploy)
							if err != nil {
								fmt.Printf("Error deleting hpa. Error: %s\n", err.Error())
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
