#!/bin/sh

docker run -i -v `pwd`:/postfix_exporter ubuntu:16.04 /bin/sh << 'EOF'
set -ex

# Install prerequisites for the build process.
apt-get update -q
apt-get install -yq git golang-go libsystemd-dev

mkdir /go
export GOPATH=/go
cd /postfix_exporter

go get -d ./...
go build --ldflags '-extldflags "-static"'
strip postfix_exporter
EOF
