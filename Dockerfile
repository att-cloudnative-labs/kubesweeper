# First stage: Build the Go binary
FROM golang:1.12.1 as builder

# copy necessary files
COPY /main.go /app/main.go
COPY /config.go /app/config.go
COPY /configs /app/configs/
COPY go.mod /app/go.mod
COPY go.sum /app/go.sum

WORKDIR /app

RUN GIT_TERMINAL_PROMPT=1 GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build --installsuffix cgo --ldflags="-s" -o kubecrawler

# The second stage: Just what's needed
FROM scratch
# copy the binary and the settings into the container
COPY --from=builder /app/kubecrawler /app/kubecrawler
COPY --from=builder /app/configs/ /app/configs/

# Run it
ENTRYPOINT ["/app/kubecrawler"]