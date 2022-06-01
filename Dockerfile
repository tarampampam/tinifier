# syntax=docker/dockerfile:1.2

# Image page: <https://hub.docker.com/_/golang>
FROM --platform=${TARGETPLATFORM:-linux/amd64} golang:1.18.2-alpine as builder

# can be passed with any prefix (like `v1.2.3@GITHASH`)
# e.g.: `docker build --build-arg "APP_VERSION=v1.2.3@GITHASH" .`
ARG APP_VERSION="undefined@docker"

RUN set -x \
    && mkdir /src \
    && apk add --no-cache ca-certificates \
    && update-ca-certificates

WORKDIR /src

COPY . .

# arguments to pass on each go tool link invocation
ENV LDFLAGS="-s -w -X github.com/tarampampam/tinifier/v3/internal/pkg/version.version=$APP_VERSION"

RUN set -x \
    && go version \
    && CGO_ENABLED=0 go build -trimpath -ldflags "$LDFLAGS" -o /tmp/tinifier ./cmd/tinifier/ \
    && /tmp/tinifier version \
    && /tmp/tinifier -h

# prepare rootfs for runtime
RUN mkdir -p /tmp/rootfs

WORKDIR /tmp/rootfs

RUN set -x \
    && mkdir -p \
        ./etc/ssl \
        ./bin \
    && cp -R /etc/ssl/certs ./etc/ssl/certs \
    && echo 'appuser:x:10001:10001::/nonexistent:/sbin/nologin' > ./etc/passwd \
    && echo 'appuser:x:10001:' > ./etc/group \
    && mv /tmp/tinifier ./bin/tinifier

# use empty filesystem
FROM scratch

ARG APP_VERSION="undefined@docker"

LABEL \
    org.opencontainers.image.title="tinifier" \
    org.opencontainers.image.description="CLI client for images compressing using tinypng.com API" \
    org.opencontainers.image.url="https://github.com/tarampampam/tinifier" \
    org.opencontainers.image.source="https://github.com/tarampampam/tinifier" \
    org.opencontainers.image.vendor="tarampampam" \
    org.opencontainers.image.version="$APP_VERSION" \
    org.opencontainers.image.licenses="MIT"

# Import from builder
COPY --from=builder /tmp/rootfs /

# Use an unprivileged user
USER appuser:appuser

ENTRYPOINT ["/bin/tinifier"]
