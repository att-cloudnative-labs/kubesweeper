build:
	docker build -t kubecrawler .
	helm template kubecrawler --set image=kubecrawler | kubectl create -f -
go-build:
	go build
