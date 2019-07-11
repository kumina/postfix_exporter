FROM golang:latest AS build-env

ENV GO111MODULE=on

RUN apt-get update && apt-get install -y libsystemd-dev

WORKDIR /go/src/app

COPY . .

RUN go build -o /go/bin/exporter


FROM scratch

COPY --from=build-env /go/bin/exporter /go/bin/

WORKDIR /go/bin

ENTRYPOINT ["/go/bin/exporter"]
