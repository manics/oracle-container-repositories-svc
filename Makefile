VERSION := $(shell git describe --tags --always HEAD)
GOFLAGS = -ldflags "-X main.Version=$(VERSION)"

default: test

build:
	go build $(GOFLAGS) .

test: build
	go test -v ./...

clean:
	rm -f oracle-container-repositories-svc
