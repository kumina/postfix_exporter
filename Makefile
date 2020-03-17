default:	test

test:	*.go
	go test -v -race ./...

fmt:
	gofmt -w .

.PHONY: imports
imports: *.go
ifeq (, $(shell which golangci-lint))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.24.0
endif
	golangci-lint run ./...

mod:
	go mod tidy

all: fmt mod test
