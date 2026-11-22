# Local build. Release images are built by GoReleaser from Dockerfile.goreleaser.
FROM golang:1.26.11-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown
RUN CGO_ENABLED=0 go build \
        -ldflags="-s -w \
            -X github.com/nexdns/cli/internal/version.Version=${VERSION} \
            -X github.com/nexdns/cli/internal/version.Commit=${COMMIT} \
            -X github.com/nexdns/cli/internal/version.Date=${DATE}" \
        -o nexdns ./cmd/nexdns

FROM alpine:3.23.8

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/nexdns /usr/local/bin/nexdns

ENTRYPOINT ["nexdns"]
