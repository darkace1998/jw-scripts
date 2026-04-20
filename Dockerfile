FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/ ./cmd/...

FROM alpine:3.20
ARG SUPERCRONIC_VERSION=v0.2.33
ARG TARGETARCH

RUN apk add --no-cache ca-certificates tzdata curl && \
    arch="${TARGETARCH:-amd64}" && \
    case "$arch" in \
      amd64|arm64) ;; \
      *) echo "Unsupported architecture: $arch" && exit 1 ;; \
    esac && \
    curl -fsSL -o /usr/local/bin/supercronic "https://github.com/aptible/supercronic/releases/download/${SUPERCRONIC_VERSION}/supercronic-linux-${arch}" && \
    chmod +x /usr/local/bin/supercronic

COPY --from=builder /out/ /usr/local/bin/
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

ENV TZ=UTC
ENV CRON_SCHEDULE="0 */6 * * *"
ENV JW_WORKDIR=/data
ENV JW_COMMAND="jwb-index --download --update --lang E /data"
ENV RUN_ON_STARTUP=true

WORKDIR /data
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
