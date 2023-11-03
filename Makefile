VERSION := $(shell git describe --tags --always HEAD)
GOFLAGS = -ldflags "-X main.Version=$(VERSION)"

VALIDATE_KUBERNETES_VERSION ?= 1.23.0

default: test

lint:
	golangci-lint run

build:
	go build $(GOFLAGS) ./cmd/binderhub-amazon
	go build $(GOFLAGS) ./cmd/binderhub-oracle

test: build
	go test ./...

clean:
	rm -f binderhub-amazon binderhub-oracle

container:
	podman build -t binderhub-container-registry-helper .

helm:
	helm package helm-chart

helm-check-values:
	helm lint helm-chart/
	helm template helm-chart/ --values ci/helm-check-values.yaml | kubeconform -strict -verbose -kubernetes-version $(VALIDATE_KUBERNETES_VERSION)

update-deps:
	go get -t -u ./...
	go mod tidy

check-tags-updated:
	go run ./ci/check_tags_updated.go helm-chart/
