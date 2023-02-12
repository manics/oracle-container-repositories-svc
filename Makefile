VERSION := $(shell git describe --tags --always HEAD)
GOFLAGS = -ldflags "-X main.Version=$(VERSION)"

default: test

build:
	go build $(GOFLAGS) ./cmd/binderhub-amazon
	go build $(GOFLAGS) ./cmd/binderhub-oracle

test: build
	go test ./...

clean:
	rm -f binderhub-amazon binderhub-oracle

container:
	podman build -t binderhub-container-registry .
