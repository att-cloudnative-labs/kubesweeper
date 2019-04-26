build:
	docker build -t kubesweeper .
	helm template kubesweeper --set image=kubesweeper | kubectl create -f -
go-build:
	go build
