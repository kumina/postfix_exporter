#!/bin/sh

docker run -i -v `pwd`:/postfix_exporter golang:1.12 /bin/sh << 'EOF'
set -ex

# Install prerequisites for the build process.
apt-get update -q
apt-get install -yq libsystemd-dev

cd /postfix_exporter

go get -d ./...
go build -a -tags static_all
strip postfix_exporter
EOF
