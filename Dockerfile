FROM golang:1.16-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

RUN apk add --no-cache --virtual .build-deps build-base elogind-dev

COPY . .

# Force the go compiler to use modules
ENV GO111MODULE=on
RUN go test
RUN go build -o /bin/postfix_exporter

FROM alpine
EXPOSE 9154
WORKDIR /
COPY --from=builder /bin/postfix_exporter /bin/
ENTRYPOINT ["/bin/postfix_exporter"]
