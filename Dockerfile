FROM golang:1.13 AS builder
WORKDIR /src

# avoid downloading the dependencies on succesive builds
RUN apt-get update -qq && apt-get install -qqy \
  build-essential \
  libsystemd-dev
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.24.0
COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

COPY . .

# Force the go compiler to use modules
ENV GO111MODULE=on
RUN go test ./...
RUN golangci-lint run ./...
RUN go build -o /bin/postfix_exporter

FROM debian:latest
EXPOSE 9154
WORKDIR /
COPY --from=builder /bin/postfix_exporter /bin/
ENTRYPOINT ["/bin/postfix_exporter"]
