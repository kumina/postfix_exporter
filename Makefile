DOCKER_IMAGE_NAME ?= postfix-exporter
DOCKER_IMAGE_TAG  ?= latest
GOOS              ?= $(shell uname -s | tr A-Z a-z)
GOARCH            ?= $(subst x86_64,amd64,$(patsubst i%86,386,$(shell uname -m)))

all: build

build:
	@echo ">> building $(GOOS) $(GOARCH)"
	@go get -v -u ./...
	@go build -v

docker: build
	@echo ">> building docker image"
	@docker build -t "$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" ./

clean:
	@rm -f ./prometheus_exporter

.PHONY: all build docker clean
