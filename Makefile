default:	test

test:	*.go
	go test -v -race ./...

fmt:
	gofmt -w .

imports: *.go
ifeq (, $(shell which golangci-lint))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.24.0
endif
	golangci-lint run ./...

mod:
	go mod tidy

docker:
	docker build -t postfix_exporter .

all: fmt mod test

.PHONY: imports test fmt mod docker all default
