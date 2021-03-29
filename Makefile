
IMAGE := "prometheus-multi-tenant-proxy"
REGISTRY := "partitio"
RELEASE_DATE := "$(shell date --utc '+%Y-%m-%d-Z %H:%M')"
VERSION := $(shell git describe --tags `git rev-list --tags --max-count=1`)
COMMIT := $(shell git rev-parse --short HEAD)

BUILD_ARGS := --build-arg=RELEASE_DATE=$(RELEASE_DATE) \
	--build-arg=VERSION=$(VERSION) \
	--build-arg=COMMIT=$(COMMIT)

.PHONY:
tests:
	@go test -v ./...

.PHONY:
docker-build:
	@docker image build -t $(REGISTRY)/$(IMAGE) $(BUILD_ARGS) -f build/package/Dockerfile .

.PHONY:
docker-push:
	@docker image push $(REGISTRY)/$(IMAGE)

.PHONY:
docker: docker-build docker-push
