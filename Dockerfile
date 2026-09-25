# Build on the native platform and cross-compile for the target, which is
# much faster than building under QEMU emulation.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder
ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION=dev
# supercronic is built from source so its integrity is verified by the Go
# checksum database instead of trusting an unverified binary download.
ARG SUPERCRONIC_VERSION=v0.2.49
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
      go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/ ./cmd/... && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
      go install -trimpath -ldflags="-s -w" github.com/aptible/supercronic@${SUPERCRONIC_VERSION} && \
    find /go/bin -type f -name supercronic -exec cp {} /out/supercronic \;

FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata su-exec

COPY --from=builder /out/ /usr/local/bin/
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod 0755 /usr/local/bin/docker-entrypoint.sh

ENV TZ=UTC
ENV CRON_SCHEDULE="0 */6 * * *"
ENV JW_WORKDIR=/data
ENV JW_COMMAND="jwb-index --download --update --lang E /data"
ENV RUN_ON_STARTUP=true
# Downloads are written as this user and group instead of root. Set both to
# 0 to keep running as root.
ENV PUID=1000
ENV PGID=1000

# New volumes inherit this ownership, so they need no migration
RUN mkdir -p /data && chown 1000:1000 /data
WORKDIR /data
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
