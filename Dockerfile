ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm as builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN go build -v -o /usr/local/bin/app ./cmd/api

FROM debian:bookworm

RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates

COPY --from=builder /usr/local/bin/app /usr/local/bin/app

CMD ["/usr/local/bin/app"]