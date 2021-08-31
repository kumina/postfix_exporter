FROM golang:1.16 AS builder
WORKDIR /src

# avoid downloading the dependencies on succesive builds
RUN apt-get update -qq && apt-get install -qqy \
  build-essential \
  libsystemd-dev

COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

COPY . .

# Force the go compiler to use modules
ENV GO111MODULE=on
RUN go test
RUN go build -o /bin/postfix_exporter

FROM debian:latest
EXPOSE 9154
WORKDIR /
COPY --from=builder /bin/postfix_exporter /bin/
ENTRYPOINT ["/bin/postfix_exporter"]
