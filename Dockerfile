FROM golang:latest AS build-env

ENV GO111MODULE=on

RUN apt-get update && apt-get install -y libsystemd-dev

WORKDIR /go/src/app

COPY . .

RUN go build -o /bin/postfix_exporter


FROM debian:latest

COPY --from=build-env /bin/postfix_exporter /bin/postfix_exporter

WORKDIR /

EXPOSE 9154/tcp

ENTRYPOINT ["/bin/postfix_exporter"]
