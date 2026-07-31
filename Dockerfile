# Local build. The published images come from GoReleaser via
# Dockerfile.goreleaser, which assembles an already version-stamped binary.
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Without these the binary reports "dev", unlike every other way it is built.
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown
RUN CGO_ENABLED=0 go build \
        -ldflags="-s -w \
            -X github.com/nexdns/cli/internal/version.Version=${VERSION} \
            -X github.com/nexdns/cli/internal/version.Commit=${COMMIT} \
            -X github.com/nexdns/cli/internal/version.Date=${DATE}" \
        -o nexdns ./cmd/nexdns

FROM alpine:3.23

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/nexdns /usr/local/bin/nexdns

ENTRYPOINT ["nexdns"]
