VERSION := $(shell git describe --tags --always HEAD)
GOFLAGS = -ldflags "-X main.Version=$(VERSION)"

build:
	go build -o oci-container-repositories $(GOFLAGS) .
