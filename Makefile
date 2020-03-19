default:	test

test:	golangci-lint *.go
	go test -v -race ./...

fmt:
	gofmt -w .

golangci-lint: *.go
ifeq (, $(shell which golangci-lint))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.24.0
endif
	golangci-lint run --fix ./...

mod:
	go mod tidy

docker:
	docker build -t postfix_exporter .

all: fmt mod test

.PHONY: imports test fmt mod docker all default
