FROM golang:1.8
ADD . /go/src/github.com/kumina/postfix_exporter
WORKDIR /go/src/github.com/kumina/postfix_exporter
RUN apt-get update -qq && apt-get install -qqy \
  build-essential \
  libsystemd-dev
RUN go get -v ./...
RUN go build

FROM debian:latest
EXPOSE 9154
WORKDIR /
COPY --from=0 /go/src/github.com/kumina/postfix_exporter/postfix_exporter .
ENTRYPOINT ["/postfix_exporter"]
