# Builder stage to
FROM golang:1.13 as builder

# Add the project in the image
ADD . /build
WORKDIR /build

# Install needed dependencies for the build
RUN apt-get update -q && apt-get install -qy \
  build-essential \
  libsystemd-dev

# Get dependencies and build the static binary
RUN go test
RUN go build -a -tags static_all

# Real image
FROM debian:latest

EXPOSE 9154
WORKDIR /

# Copy the binary from the build image to the real one
COPY --from=builder /build/postfix_exporter .

ENTRYPOINT ["/postfix_exporter"]
