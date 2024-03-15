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
	go test ./... -count=1

# Assumes docker.io/localstack/localstack running on port 4566
# Run all tests including integration tests
test-integration: export AWS_ENDPOINT = http://localhost:4566
test-integration: export AWS_ACCESS_KEY_ID = AWS_ACCESS_KEY_ID
test-integration: export AWS_SECRET_ACCESS_KEY = AWS_SECRET_ACCESS_KEY
test-integration: export AWS_REGION = us-east-1
test-integration: build
	go test ./... --tags=integration -count=1

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
