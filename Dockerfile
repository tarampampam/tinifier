# Image page: <https://hub.docker.com/_/golang>
FROM golang:1.15.6-alpine as builder

# can be passed with any prefix (like `v1.2.3@GITHASH`)
# e.g.: `docker build --build-arg "APP_VERSION=v1.2.3@GITHASH" .`
ARG APP_VERSION="undefined@docker"

RUN set -x \
    && mkdir /src \
    && apk add --no-cache ca-certificates \
    && update-ca-certificates

WORKDIR /src

COPY ./go.mod ./go.sum ./

# Burn modules cache
RUN set -x \
    && go mod download \
    && go mod verify

COPY . .

# arguments to pass on each go tool link invocation
ENV LDFLAGS="-s -w -X tinifier/internal/pkg/version.version=$APP_VERSION"

RUN set -x \
    && go version \
    && CGO_ENABLED=0 go build -trimpath -ldflags "$LDFLAGS" -o /tmp/tinifier ./cmd/tinifier/main.go \
    && /tmp/tinifier version \
    && /tmp/tinifier -h

# prepare rootfs for runtime
RUN set -x \
    && mkdir -p /tmp/rootfs/etc/ssl \
    && mkdir -p /tmp/rootfs/bin \
    && cp -R /etc/ssl/certs /tmp/rootfs/etc/ssl/certs \
    && echo 'appuser:x:10001:10001::/nonexistent:/sbin/nologin' > /tmp/rootfs/etc/passwd \
    && mv /tmp/tinifier /tmp/rootfs/bin/tinifier

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
USER appuser

ENTRYPOINT ["/bin/tinifier"]
