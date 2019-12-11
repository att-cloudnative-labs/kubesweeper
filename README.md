# Kubesweeper

Automatically iterates through resources in a lab Kubernetes cluster and acts according to [certain conditions outlined here](#configuration-defaults). As of now, Kubesweeper will delete deployments and their associated resources if the waiting reason and/or pod restart counts dictate.

If your lab Kubernetes clusters are filling up with non-Running pods, then Kubesweeper's automatic deletion
can assist. Future iterations of this project can involve other actions based on crawling through Kubernetes cluster resources, such as generating reports per namespace without actually deleting. 

Please note that Kubesweeper is intended for use in lab—not production, customer-facing—clusters.

<p align="center">
  <img src="https://travis-ci.org/att-cloudnative-labs/kubesweeper.svg?branch=master">	
  <a href="https://goreportcard.com/report/github.com/att-cloudnative-labs/kubesweeper" alt="Go Report Card">
    <img src="https://goreportcard.com/badge/github.com/att-cloudnative-labs/kubesweeper">
  </a>	
</p>
<p align="center">
    <a href="https://github.com/att-cloudnative-labs/kubesweeper/graphs/contributors" alt="Contributors">
		<img src="https://img.shields.io/github/contributors/att-cloudnative-labs/kubesweeper.svg">
	</a>
	<a href="https://github.com/att-cloudnative-labs/kubesweeper/commits/master" alt="Commits">
		<img src="https://img.shields.io/github/commit-activity/m/att-cloudnative-labs/kubesweeper.svg">
	</a>
	<a href="https://github.com/att-cloudnative-labs/kubesweeper/pulls" alt="Open pull requests">
		<img src="https://img.shields.io/github/issues-pr-raw/att-cloudnative-labs/kubesweeper.svg">
	</a>
	<a href="https://github.com/att-cloudnative-labs/kubesweeper/pulls" alt="Closed pull requests">
    	<img src="https://img.shields.io/github/issues-pr-closed-raw/att-cloudnative-labs/kubesweeper.svg">
	</a>
	<a href="https://github.com/att-cloudnative-labs/kubesweeper/issues" alt="Issues">
		<img src="https://img.shields.io/github/issues-raw/att-cloudnative-labs/kubesweeper.svg">
	</a>
	</p>
<p align="center">
	<a href="https://github.com/att-cloudnative-labs/kubesweeper/stargazers" alt="Stars">
		<img src="https://img.shields.io/github/stars/att-cloudnative-labs/kubesweeper.svg?style=social">
	</a>	
	<a href="https://github.com/att-cloudnative-labs/kubesweeper/watchers" alt="Watchers">
		<img src="https://img.shields.io/github/watchers/att-cloudnative-labs/kubesweeper.svg?style=social">
	</a>	
	<a href="https://github.com/att-cloudnative-labs/kubesweeper/network/members" alt="Forks">
		<img src="https://img.shields.io/github/forks/att-cloudnative-labs/kubesweeper.svg?style=social">
	</a>	
</p>

## Deployment as a Kubernetes CronJob
If the desired cluster does not have Knative installed, then Kubesweeper can be installed as a Kubernetes CronJob.

1. Build docker image
```bash
$ docker build -t kubesweeper .
```
2. Create Kubernetes resources from ```install``` directory
```bash
$ kubectl apply -f install/
```

Note that step 2 must be run in the context of the Kubernetes cluster. After that command is run, the appropriate Kubernetes resources will be created from the .yaml files in ```install```.

## Deployment as a Knative CronJobSource
If you wish to deploy Kubesweeper on Knative as a CronJobSource, you can use Helm. For information on installing Helm, please refer to the [Helm quickstart guide](https://helm.sh/docs/using_helm/). After installing Helm, the following steps can be manually run:

1. Build docker image
```bash
$ docker build -t kubesweeper .
```
2. Run helm template to install Kubesweeper
```bash
$ helm template kubesweeper --set image=<KUBESWEEPER_IMAGE> | kubectl create -f -
```

In lieu of step 2, a Makefile can be used to pull values from ./helm/kubesweeper/values.yaml:

```bash
$ make
```

## Configuration Defaults

Under the ```configs``` folder, the ```config.yaml``` has the following default configurations:

* Pod waiting reasons
  * CrashLoopBackOff
  * ImagePullBackOff
  * ErrImagePull
  * Completed
  * Failed
* Pod restart threshold
  * 144
    * If the pod restart threshold is at least this number *and* has a pod waiting reason of ```CrashLoopBackOff```, then Kubesweeper will delete the associated resources

Helm function configurations can be found in ```~/helm/kubesweeper/values.yaml```.

* name
  * Name to use for deployment
* image
  * Image used in deployment
* cron
  * Cron expression used to schedule Kubesweeper
    * Any valid cron expression can be used
* namespace
  * Namespace job will be deployed in

## Contributing

1. [Fork Kubesweeper](https://github.com/att-cloudnative-labs/kubesweeper/fork)
2. Create your feature branch (`git checkout -b feature/fooBar`)
3. Commit your changes (`git commit -am 'Add some fooBar'`)
4. Push to the branch (`git push origin feature/fooBar`)
5. Create a new Pull Request

## Additional info

<p align="center">
  <a href="https://github.com/att-cloudnative-labs" alt="AT&T Cloud Native Labs">
    <img src="./images/cloud_native_labs.png" height="50%" width="50%">
  </a>	
</p>

Maintained and in-use by the Platform Team @ AT&T Entertainment Cloud Native Labs.

Distributed under the AT&T MIT license. See ``LICENSE`` for more information.
