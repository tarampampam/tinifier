# syntax=docker/dockerfile:1

# -✂- this stage is used to compile the application -------------------------------------------------------------------
FROM docker.io/library/golang:1.24-alpine AS compile

# can be passed with any prefix (like `v1.2.3@FOO`), e.g.: `docker build --build-arg "APP_VERSION=v1.2.3@FOO" .`
ARG APP_VERSION="undefined@docker"

# copy the source code
COPY . /src
WORKDIR /src

RUN set -x \
    # build the app itself
    && go generate -skip readme ./... \
    && CGO_ENABLED=0 go build \
      -trimpath \
      -ldflags "-s -w -X gh.tarampamp.am/tinifier/v5/internal/version.version=${APP_VERSION}" \
      -o ./tinifier \
      ./cmd/tinifier/ \
    && ./tinifier --help

# prepare rootfs for runtime
WORKDIR /tmp/rootfs
RUN set -x \
    && mkdir -p \
        ./etc/ssl \
        ./bin \
    && cp -R /etc/ssl/certs ./etc/ssl/certs \
    && echo 'appuser:x:10001:10001::/nonexistent:/sbin/nologin' > ./etc/passwd \
    && echo 'appuser:x:10001:' > ./etc/group \
    && mv /src/tinifier ./bin/tinifier

# -✂- and this is the final stage -------------------------------------------------------------------------------------
FROM scratch AS runtime

ARG APP_VERSION="undefined@docker"

LABEL \
    # docs: <https://github.com/opencontainers/image-spec/blob/master/annotations.md>
    org.opencontainers.image.title="tinifier" \
    org.opencontainers.image.description="CLI client for images compressing using tinypng.com API" \
    org.opencontainers.image.url="https://github.com/tarampampam/tinifier" \
    org.opencontainers.image.source="https://github.com/tarampampam/tinifier" \
    org.opencontainers.image.vendor="tarampampam" \
    org.opencontainers.version="$APP_VERSION" \
    org.opencontainers.image.licenses="MIT"

# import from builder
COPY --from=compile /tmp/rootfs /

# use an unprivileged user
USER 10001:10001

ENTRYPOINT ["/bin/tinifier"]
